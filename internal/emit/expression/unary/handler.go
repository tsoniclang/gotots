package unary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.UnaryExpr,
) (tsgo.Expression, error) {
	if source.Op == token.SUB && context.TypesInfo().Types[source].Value != nil {
		return children.IntegerConstant(context, source)
	}
	resultType, resultIsBasic := boolType(context.TypesInfo().TypeOf(source))
	operandType, operandIsBasic := boolType(context.TypesInfo().TypeOf(source.X))
	if source.Op != token.NOT ||
		!resultIsBasic ||
		resultType.Kind() != types.Bool ||
		!operandIsBasic ||
		operandType.Kind() != types.Bool {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(types.Typ[types.Bool]),
		source.X,
	)
	if err != nil {
		return nil, err
	}
	return context.Factory().PrefixUnaryExpression(
		tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
		operand,
	), nil
}

func boolType(sourceType types.Type) (*types.Basic, bool) {
	if sourceType == nil {
		return nil, false
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return basic, ok
}
