package sourceimplementation

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	representationcontract "github.com/tsoniclang/gotots/internal/contracts/representation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/load/callablefacts"
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
	declarations := sourceFunctionDeclarations(selected)
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
			declarations[function],
			selected.TypesInfo(),
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
	declaration *ast.FuncDecl,
	info *types.Info,
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
			declaration,
			info,
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
	declaration *ast.FuncDecl,
	info *types.Info,
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
				declaration,
				info,
			)
			if err != nil {
				return callableabi.Parameter{}, err
			}
		} else if primitive, primitiveOK, primitiveErr :=
			project.CallableParameterPrimitive(target, index); primitiveErr != nil {
			return callableabi.Parameter{}, primitiveErr
		} else if primitiveOK {
			return callableabi.Parameter{}, &Error{
				Operation: "join callable ABI",
				Subject:   function.FullName(),
				Reason: fmt.Sprintf(
					"parameter %d projects a non-primitive pointer as primitive %d",
					index,
					primitive.Kind(),
				),
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
	return callableabi.NewParameter(projection, nilPolicy, targetType)
}

func verifyPrimitivePointerProjection(
	function *types.Func,
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	index int,
	pointee types.Type,
	pointeeType string,
	declaration *ast.FuncDecl,
	info *types.Info,
) (callableabi.Projection, callableabi.NilPolicy, error) {
	primitive, primitiveOK, err := project.CallableParameterPrimitive(target, index)
	if err != nil {
		return callableabi.ProjectionInvalid, callableabi.NilPolicyInvalid, err
	}
	if primitiveOK && primitive.Kind() == projectPrimitiveKind(pointeeType) {
		evidence := callablefacts.PointeeValueEvidence(
			declaration,
			function.Signature().Params().At(index),
			info,
		)
		if evidence != callablefacts.PointeeReadEntryStable {
			return callableabi.ProjectionInvalid,
				callableabi.NilPolicyInvalid,
				&Error{
					Operation: "join callable ABI",
					Subject:   function.FullName(),
					Reason: fmt.Sprintf(
						"parameter %d has no exact pointee-value source proof",
						index,
					),
				}
		}
		if primitive.Optional() {
			return callableabi.ProjectionPointeeValue,
				callableabi.NilPolicyPreserve, nil
		}
		return callableabi.ProjectionPointeeValue,
			callableabi.NilPolicyRejectAtBoundary, nil
	}
	if !primitiveOK {
		return callableabi.ProjectionIdentity,
			callableabi.NilPolicyNotApplicable, nil
	}
	return callableabi.ProjectionInvalid, callableabi.NilPolicyInvalid, &Error{
		Operation: "join callable ABI",
		Subject:   function.FullName(),
		Reason: fmt.Sprintf(
			"parameter %d primitive target does not preserve pointee type %q",
			index,
			pointeeType,
		),
	}
}

func sourceFunctionDeclarations(
	selected *load.Package,
) map[*types.Func]*ast.FuncDecl {
	result := make(map[*types.Func]*ast.FuncDecl)
	for _, sourceFile := range selected.Files() {
		for _, declaration := range sourceFile.Syntax().Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name == nil {
				continue
			}
			object, ok := selected.TypesInfo().Defs[function.Name].(*types.Func)
			if ok {
				result[object] = function
			}
		}
	}
	return result
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
