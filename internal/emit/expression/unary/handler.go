package unary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.UnaryExpr,
) (api.ExpressionEmission, error) {
	if source.Op == token.AND {
		return children.Address(context, source)
	}
	if source.Op == token.SUB && context.TypesInfo().Types[source].Value != nil {
		return children.IntegerConstant(context, source)
	}
	if result, ok, err := emitInteger(context, children, source); ok || err != nil {
		return result, err
	}
	resultType, resultIsBasic := boolType(context.TypesInfo().TypeOf(source))
	operandType, operandIsBasic := boolType(context.TypesInfo().TypeOf(source.X))
	if source.Op != token.NOT ||
		!resultIsBasic ||
		resultType.Kind() != types.Bool ||
		!operandIsBasic ||
		operandType.Kind() != types.Bool {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(types.Typ[types.Bool]),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		operand.Before(),
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			operand.Value(),
		),
		operand.Requests(),
	)
}

func emitInteger(
	context api.Context,
	children api.ChildEmitter,
	source *ast.UnaryExpr,
) (api.ExpressionEmission, bool, error) {
	resultType := context.TypesInfo().TypeOf(source)
	operandType := context.TypesInfo().TypeOf(source.X)
	carrier, ok := integervalue.Describe(context.TypesSizes(), resultType)
	if !ok ||
		!types.AssignableTo(operandType, resultType) ||
		!integervalue.SupportsUnary(
			context.IntegerRepresentation(),
			carrier,
			source.Op,
		) {
		return api.ExpressionEmission{}, false, nil
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(resultType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target := operand.Value()
	switch source.Op {
	case token.ADD:
		if context.IntegerRepresentation() == api.IntegerRepresentationNumber {
			target = context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindPlusToken,
				target,
			)
		}
	case token.SUB:
		target = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			target,
		)
	case token.XOR:
		target = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindTildeToken,
			target,
		)
		switch {
		case integervalue.RequiresUint32Normalization(
			context.IntegerRepresentation(),
			carrier,
		):
			zero, literalErr := api.IntegerLiteral(
				context.Factory(),
				api.IntegerRepresentationNumber,
				"0",
			)
			if literalErr != nil {
				return api.ExpressionEmission{}, true, literalErr
			}
			target = context.Factory().BinaryExpression(
				nil,
				target,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorGreaterThanGreaterThanGreaterThanToken,
				),
				zero,
			)
		case !carrier.Signed():
			mask, ok := integervalue.UnsignedMask(carrier)
			if !ok {
				return api.ExpressionEmission{}, true,
					api.Unsupported(context, api.CategoryExpression, source)
			}
			maskLiteral, literalErr := api.IntegerLiteral(
				context.Factory(),
				context.IntegerRepresentation(),
				mask,
			)
			if literalErr != nil {
				return api.ExpressionEmission{}, true, literalErr
			}
			target = context.Factory().BinaryExpression(
				nil,
				target,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorAmpersandToken,
				),
				maskLiteral,
			)
		}
	default:
		return api.ExpressionEmission{}, false, nil
	}
	result, err := api.NewExpressionEmission(
		operand.Before(),
		target,
		operand.Requests(),
	)
	return result, true, err
}

func boolType(sourceType types.Type) (*types.Basic, bool) {
	if sourceType == nil {
		return nil, false
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return basic, ok
}
