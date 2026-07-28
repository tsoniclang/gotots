package unary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	definedunary "github.com/tsoniclang/gotots/internal/emit/expression/unary/defined"
	unaryoperation "github.com/tsoniclang/gotots/internal/emit/expression/unary/operation"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.UnaryExpr,
) (api.ExpressionEmission, error) {
	if source.Op == token.AND {
		return children.Address(context, source)
	}
	if target, handled, err := constantvalue.EmitFolded(
		context,
		source,
	); handled {
		return target, err
	}
	if unaryConstantEvidenceIsIncomplete(context, source) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if target, handled, err := definedunary.Emit(
		context,
		children,
		source,
	); handled {
		return target, err
	}
	resultType := context.TypesInfo().TypeOf(source)
	operandType := context.TypesInfo().TypeOf(source.X)
	if resultType == nil ||
		operandType == nil ||
		!types.AssignableTo(operandType, resultType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(resultType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, handled, err := unaryoperation.Apply(
		context,
		source.Op,
		resultType,
		operand,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return target, nil
}

func unaryConstantEvidenceIsIncomplete(
	context api.Context,
	source *ast.UnaryExpr,
) bool {
	switch source.Op {
	case token.ADD, token.SUB, token.XOR, token.NOT:
	default:
		return false
	}
	result, resultExists := context.TypesInfo().Types[source]
	operand, operandExists := context.TypesInfo().Types[source.X]
	return resultExists &&
		result.Value == nil &&
		operandExists &&
		operand.Value != nil
}
