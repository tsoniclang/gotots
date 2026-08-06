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
	identity, err := callableabi.PackageFunctionIdentity(
		function.Pkg().Path(),
		function.Name(),
	)
	if err != nil {
		return callableabi.Callable{}, err
	}
	return callableabi.New(identity, parameters)
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
	projection := callableabi.ProjectionIdentity
	nilPolicy := callableabi.NilPolicyNotApplicable
	pointer, pointerParameter := types.Unalias(
		function.Signature().Params().At(index).Type(),
	).(*types.Pointer)
	if pointerParameter {
		pointeeType, supported := representationcontract.PrimitiveTypeScriptType(
			pointer.Elem(),
			scalar,
		)
		if supported {
			projection, nilPolicy, err = verifyPrimitivePointerProjection(
				function,
				project,
				target,
				index,
				pointer.Elem(),
				pointeeType,
				targetType,
			)
			if err != nil {
				return callableabi.Parameter{}, err
			}
		}
	}
	return callableabi.NewParameter(projection, nilPolicy, targetType)
}

func verifyPrimitivePointerProjection(
	function *types.Func,
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	index int,
	pointee types.Type,
	pointeeType string,
	targetType string,
) (callableabi.Projection, callableabi.NilPolicy, error) {
	alias, _ := representationcontract.PrimitiveAliasFor(pointee)
	aliasName, err := representationcontract.PrimitiveAliasName(alias)
	if err != nil {
		return callableabi.ProjectionInvalid, callableabi.NilPolicyInvalid, err
	}
	primitive, primitiveOK, err := project.CallableParameterPrimitive(target, index)
	if err != nil {
		return callableabi.ProjectionInvalid, callableabi.NilPolicyInvalid, err
	}
	if primitiveOK && primitive.Kind() == projectPrimitiveKind(pointeeType) {
		if primitive.Optional() {
			return callableabi.ProjectionPointeeValue,
				callableabi.NilPolicyPreserve, nil
		}
		return callableabi.ProjectionPointeeValue,
			callableabi.NilPolicyRejectAtBoundary, nil
	}
	canonicalPointer := fmt.Sprintf(
		"GoPointer<%s, %s> | undefined",
		aliasName,
		aliasName,
	)
	if targetType == canonicalPointer {
		return callableabi.ProjectionIdentity,
			callableabi.NilPolicyNotApplicable, nil
	}
	return callableabi.ProjectionInvalid, callableabi.NilPolicyInvalid, &Error{
		Operation: "join callable ABI",
		Subject:   function.FullName(),
		Reason: fmt.Sprintf(
			"parameter %d type %q is neither %q nor the canonical pointer representation",
			index,
			targetType,
			pointeeType,
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
