package unary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
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
		return emitConstant(context, source)
	}
	if result, ok, err := emitInteger(context, children, source); ok || err != nil {
		return result, err
	}
	if result, ok, err := emitFloat(context, children, source); ok || err != nil {
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

// emitConstant materializes a constant unary expression from its folded checker
// value through the single value owner, so a negated integer or float constant
// (`-5`, `-4.5`) renders at its target representation without re-evaluating the
// operator or the source spelling.
func emitConstant(
	context api.Context,
	source *ast.UnaryExpr,
) (api.ExpressionEmission, error) {
	typeAndValue := context.TypesInfo().Types[source]
	if typeAndValue.Value == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := context.ExpectedType()
	if targetType == nil {
		targetType = typeAndValue.Type
	}
	if typeAndValue.Type == nil ||
		!types.AssignableTo(typeAndValue.Type, targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return constantvalue.EmitValue(context, source, targetType, typeAndValue.Value)
}

// emitFloat handles runtime float64 unary negation and identity. float32 needs
// rounding and stays a boundary here.
func emitFloat(
	context api.Context,
	children api.ChildEmitter,
	source *ast.UnaryExpr,
) (api.ExpressionEmission, bool, error) {
	if source.Op != token.SUB && source.Op != token.ADD {
		return api.ExpressionEmission{}, false, nil
	}
	resultType := context.TypesInfo().TypeOf(source)
	carrier, ok := floatvalue.Describe(resultType)
	if !ok || carrier.Bits() != 64 ||
		!types.AssignableTo(context.TypesInfo().TypeOf(source.X), resultType) {
		return api.ExpressionEmission{}, false, nil
	}
	operand, err := children.Expression(
		context.WithRole(api.RoleUnaryOperand).WithExpectedType(resultType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if source.Op == token.ADD {
		identity, err := api.NewExpressionEmission(
			operand.Before(),
			operand.Value(),
			operand.Requests(),
		)
		return identity, true, err
	}
	result, err := api.NewExpressionEmission(
		operand.Before(),
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			operand.Value(),
		),
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
