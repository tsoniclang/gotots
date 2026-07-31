package typeassertion

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	assertionoperation "github.com/tsoniclang/gotots/internal/emit/expression/typeassertion/operation"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeAssertExpr,
) (api.ExpressionEmission, error) {
	sourceType, targetType, results, ok := assertionTypes(context, source)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleConversionOperand).
			WithExpectedType(sourceType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if api.ContainsGenericTypeParameter(targetType) {
		return interfacevalue.AssertGeneric(
			context,
			source,
			sourceType,
			targetType,
			results != nil,
			receiver,
		)
	}
	return assertionoperation.Apply(
		context,
		children,
		source,
		sourceType,
		targetType,
		results != nil,
		receiver,
	)
}

func assertionTypes(
	context api.Context,
	source *ast.TypeAssertExpr,
) (types.Type, types.Type, *types.Tuple, bool) {
	if source == nil || source.X == nil || source.Type == nil {
		return nil, nil, nil, false
	}
	sourceType := context.TypesInfo().TypeOf(source.X)
	sourceInterface, ok := interfacetype.Resolve(sourceType)
	if !ok {
		return nil, nil, nil, false
	}
	targetType := context.TypesInfo().TypeOf(source.Type)
	if targetType == nil || !types.AssertableTo(sourceInterface, targetType) {
		return nil, nil, nil, false
	}
	if results := context.ExpectedResults(); results != nil {
		actual, ok := context.TypesInfo().TypeOf(source).(*types.Tuple)
		if !ok ||
			!types.Identical(actual, results) ||
			results.Len() != 2 ||
			!types.Identical(results.At(0).Type(), targetType) ||
			!types.Identical(
				results.At(1).Type(),
				types.Typ[types.Bool],
			) {
			return nil, nil, nil, false
		}
		return sourceType, targetType, results, true
	}
	actual := context.TypesInfo().TypeOf(source)
	expected := context.ExpectedType()
	if actual == nil ||
		expected == nil ||
		!types.Identical(actual, targetType) ||
		!types.AssignableTo(targetType, expected) {
		return nil, nil, nil, false
	}
	return sourceType, targetType, nil, true
}
