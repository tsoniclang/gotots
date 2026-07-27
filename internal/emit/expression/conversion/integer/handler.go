package integer

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, bool, error) {
	targetType := context.TypesInfo().TypeOf(source)
	targetCarrier, represented := integervalue.Describe(
		context.TypesSizes(),
		targetType,
	)
	if !represented || !isPredeclaredIntegerType(context.TypesInfo(), source.Fun, targetType) {
		return api.ExpressionEmission{}, false, nil
	}
	if source.Ellipsis != token.NoPos || len(source.Args) != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(targetType, expected) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand := source.Args[0]
	operandFacts, ok := context.TypesInfo().Types[operand]
	if !ok || operandFacts.Type == nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(
				context.WithRole(api.RoleConversionOperand),
				api.CategoryExpression,
				operand,
			)
	}
	if operandFacts.Value != nil && operandFacts.Value.Kind() == constant.Int {
		resultFacts, resultExists := context.TypesInfo().Types[source]
		if !resultExists ||
			resultFacts.Value == nil ||
			!constant.Compare(
				operandFacts.Value,
				token.EQL,
				resultFacts.Value,
			) {
			return api.ExpressionEmission{}, true,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		target, err := constantvalue.EmitValue(
			context.WithRole(api.RoleConversionOperand),
			source,
			targetType,
			resultFacts.Value,
		)
		return target, true, err
	}
	sourceCarrier, ok := integervalue.Describe(
		context.TypesSizes(),
		operandFacts.Type,
	)
	if !ok || !integervalue.CanConvert(sourceCarrier, targetCarrier) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := children.Expression(
		context.
			WithRole(api.RoleConversionOperand).
			WithExpectedType(operandFacts.Type),
		operand,
	)
	return target, true, err
}

func isPredeclaredIntegerType(
	info *types.Info,
	source ast.Expr,
	target types.Type,
) bool {
	name, ok := source.(*ast.Ident)
	if !ok {
		return false
	}
	object, ok := info.Uses[name].(*types.TypeName)
	if !ok || object != types.Universe.Lookup(object.Name()) {
		return false
	}
	return types.Identical(object.Type(), target)
}
