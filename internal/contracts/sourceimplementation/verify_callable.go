package sourceimplementation

import (
	"fmt"
	"go/types"
	"slices"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	representationcontract "github.com/tsoniclang/gotots/internal/contracts/representation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func verifyCallableABIs(
	selected *load.Package,
	project *tsgo.ProjectInspection,
	exports []tsgo.ProjectExport,
	compilation CompilationDocument,
) ([]callableabi.Callable, error) {
	integer, err := representationcontract.ParseIntegerRepresentation(
		compilation.Integers,
	)
	if err != nil {
		return nil, &Error{Operation: "join callable ABI", Reason: err.Error()}
	}
	scalar, err := representationcontract.NewScalarABIFromSizes(
		integer,
		selected.TypesSizes(),
	)
	if err != nil {
		return nil, &Error{Operation: "join callable ABI", Reason: err.Error()}
	}
	result := make([]callableabi.Callable, 0)
	for _, target := range exports {
		function, ok := selected.Types().Scope().Lookup(target.Name()).(*types.Func)
		if !ok {
			continue
		}
		callable, err := verifyCallableABI(
			function,
			project,
			target,
			scalar,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, callable)
	}
	slices.SortFunc(result, func(left, right callableabi.Callable) int {
		return strings.Compare(left.Identity(), right.Identity())
	})
	return result, nil
}

func verifyCallableABI(
	function *types.Func,
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	scalar representationcontract.ScalarABI,
) (callableabi.Callable, error) {
	signature := function.Signature()
	parameterCount, err := project.CallableParameterCount(target)
	if err != nil {
		return callableabi.Callable{}, err
	}
	if parameterCount != signature.Params().Len() {
		return callableabi.Callable{}, &Error{
			Operation: "join callable ABI",
			Subject:   function.FullName(),
			Reason: fmt.Sprintf(
				"implementation has %d parameters, want %d",
				parameterCount,
				signature.Params().Len(),
			),
		}
	}
	typeParameterCount, err := project.CallableTypeParameterCount(target)
	if err != nil {
		return callableabi.Callable{}, err
	}
	if typeParameterCount != signature.TypeParams().Len() {
		return callableabi.Callable{}, &Error{
			Operation: "join callable ABI",
			Subject:   function.FullName(),
			Reason:    "implementation type-parameter cardinality differs",
		}
	}
	parameters := make([]callableabi.Parameter, parameterCount)
	for index := range parameterCount {
		parameters[index], err = verifyCallableParameter(
			function,
			project,
			target,
			index,
			scalar,
		)
		if err != nil {
			return callableabi.Callable{}, err
		}
	}
	resultType, err := verifyCallableResult(function, project, target, scalar)
	if err != nil {
		return callableabi.Callable{}, err
	}
	identity, err := callableabi.PackageFunctionIdentity(
		function.Pkg().Path(),
		function.Name(),
	)
	if err != nil {
		return callableabi.Callable{}, err
	}
	return callableabi.New(identity, parameters, resultType)
}

func verifyCallableParameter(
	function *types.Func,
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	index int,
	scalar representationcontract.ScalarABI,
) (callableabi.Parameter, error) {
	targetType, err := project.CallableParameterTypeString(target, index)
	if err != nil {
		return callableabi.Parameter{}, err
	}
	pointer, pointerParameter := types.Unalias(
		function.Signature().Params().At(index).Type(),
	).(*types.Pointer)
	if pointerParameter {
		primitiveType, supported := representationcontract.PrimitiveTypeScriptType(
			pointer.Elem(),
			scalar,
		)
		if supported {
			primitive, primitiveOK, primitiveErr :=
				project.CallableParameterPrimitive(target, index)
			if primitiveErr != nil {
				return callableabi.Parameter{}, primitiveErr
			}
			if primitiveOK {
				return callableabi.Parameter{}, &Error{
					Operation: "join callable ABI",
					Subject:   function.FullName(),
					Reason: fmt.Sprintf(
						"parameter %d projects a pointer as primitive %d",
						index,
						primitive.Kind(),
					),
				}
			}
			expected := "Pointer<" + primitiveType + "> | undefined"
			if targetType != expected {
				return callableabi.Parameter{}, &Error{
					Operation: "join callable ABI",
					Subject:   function.FullName(),
					Reason: fmt.Sprintf(
						"parameter %d type %q differs from %q",
						index,
						targetType,
						expected,
					),
				}
			}
		}
	} else if primitiveType, supported :=
		representationcontract.PrimitiveTypeScriptType(
			function.Signature().Params().At(index).Type(),
			scalar,
		); supported {
		primitive, primitiveOK, primitiveErr :=
			project.CallableParameterPrimitive(target, index)
		if primitiveErr != nil {
			return callableabi.Parameter{}, primitiveErr
		}
		if !primitiveOK || primitive.Optional() ||
			primitive.Kind() != projectPrimitiveKind(primitiveType) {
			return callableabi.Parameter{}, &Error{
				Operation: "join callable ABI",
				Subject:   function.FullName(),
				Reason: fmt.Sprintf(
					"parameter %d does not preserve primitive type %q",
					index,
					primitiveType,
				),
			}
		}
	}
	return callableabi.NewParameter(targetType)
}

func verifyCallableResult(
	function *types.Func,
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	scalar representationcontract.ScalarABI,
) (string, error) {
	targetType, err := project.CallableReturnTypeString(target)
	if err != nil {
		return "", err
	}
	results := function.Signature().Results()
	if results.Len() == 0 {
		if targetType != "void" {
			return "", callableResultError(function, targetType, "void")
		}
		return targetType, nil
	}
	if results.Len() != 1 {
		return targetType, nil
	}
	result := types.Unalias(results.At(0).Type())
	if pointer, ok := result.(*types.Pointer); ok {
		if primitive, supported := representationcontract.PrimitiveTypeScriptType(
			pointer.Elem(),
			scalar,
		); supported {
			expected := "Pointer<" + primitive + "> | undefined"
			if targetType != expected {
				return "", callableResultError(function, targetType, expected)
			}
		}
		return targetType, nil
	}
	if expected, supported := representationcontract.PrimitiveTypeScriptType(
		result,
		scalar,
	); supported && targetType != expected {
		return "", callableResultError(function, targetType, expected)
	}
	return targetType, nil
}

func callableResultError(
	function *types.Func,
	actual string,
	expected string,
) error {
	return &Error{
		Operation: "join callable ABI",
		Subject:   function.FullName(),
		Reason: fmt.Sprintf(
			"result type %q differs from %q",
			actual,
			expected,
		),
	}
}

func projectPrimitiveKind(source string) tsgo.ProjectPrimitiveKind {
	switch source {
	case "string":
		return tsgo.ProjectPrimitiveString
	case "number":
		return tsgo.ProjectPrimitiveNumber
	case "boolean":
		return tsgo.ProjectPrimitiveBoolean
	case "bigint":
		return tsgo.ProjectPrimitiveBigInt
	default:
		return tsgo.ProjectPrimitiveInvalid
	}
}
