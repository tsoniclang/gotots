package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitPointeeValueArgument(
	context api.Context,
	children api.ChildEmitter,
	argument ast.Expr,
	pointerType types.Type,
	parameter callableabi.Parameter,
) (api.ExpressionEmission, error) {
	pointer, ok := types.Unalias(pointerType).(*types.Pointer)
	if !ok {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			argument,
		)
	}
	if operand, ok := addressedOperand(argument); ok {
		operandType := context.TypesInfo().TypeOf(operand)
		if operandType == nil || !types.AssignableTo(operandType, pointer.Elem()) {
			return api.ExpressionEmission{}, api.Unsupported(
				context,
				api.CategoryExpression,
				argument,
			)
		}
		value, err := children.Expression(
			context.
				WithRole(api.RoleCallArgument).
				WithExpectedType(pointer.Elem()),
			operand,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return context.Values().Transfer(
			context.WithRole(api.RoleCallArgument),
			argument,
			operandType,
			pointer.Elem(),
			api.ValueTransferCopy,
			value,
		)
	}
	argumentType := context.TypesInfo().TypeOf(argument)
	if argumentType == nil || !types.AssignableTo(argumentType, pointerType) {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			argument,
		)
	}
	location, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(pointerType),
		argument,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	location, err = context.Values().Transfer(
		context.WithRole(api.RoleCallArgument),
		argument,
		argumentType,
		pointerType,
		api.ValueTransferCopy,
		location,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.PointeeValues().ProjectedPointee(
		context.WithRole(api.RoleCallArgument),
		argument,
		pointerType,
		location,
		parameter.NilPolicy(),
	)
}

func addressedOperand(source ast.Expr) (ast.Expr, bool) {
	for {
		parenthesized, ok := source.(*ast.ParenExpr)
		if !ok {
			break
		}
		source = parenthesized.X
	}
	address, ok := source.(*ast.UnaryExpr)
	return addressOperand(address, ok)
}

func addressOperand(source *ast.UnaryExpr, ok bool) (ast.Expr, bool) {
	if !ok || source.Op != token.AND || source.X == nil {
		return nil, false
	}
	return source.X, true
}
