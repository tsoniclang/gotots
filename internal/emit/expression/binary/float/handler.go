package float

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// Emit handles float64 arithmetic and ordering. IEEE-754 semantics carry
// directly to TypeScript number operators: `/` by zero yields ±Infinity or NaN
// (never a panic, unlike integer division), NaN compares false under every
// ordering, and signed zeros compare equal. float32 arithmetic needs rounding at
// each operation boundary and is not yet handled here, so it falls through to
// the unsupported boundary. Equality (`==`/`!=`) is owned by the value-equality
// path before this handler runs.
func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	operandType, ok := float64Operation(context, source)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	operator, ok := targetOperator(source.Op)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	left, err := children.Expression(
		context.WithRole(api.RoleBinaryLeft).WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err := children.Expression(
		context.WithRole(api.RoleBinaryRight).WithExpectedType(operandType),
		source.Y,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if len(left.Before()) != 0 || len(right.Before()) != 0 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.DirectExpression(
		context.Factory().BinaryExpression(
			nil,
			left.Value(),
			nil,
			context.Factory().BinaryOperatorToken(operator),
			right.Value(),
		),
		api.CombineRequests(left.Requests(), right.Requests())...,
	), true, nil
}

// float64Operation returns the operand type when this binary expression is a
// float64 arithmetic or ordering operation whose operands are both float64. A
// float32 operand returns false so it stays a typed boundary until float32
// rounding is emitted.
func float64Operation(
	context api.Context,
	source *ast.BinaryExpr,
) (types.Type, bool) {
	switch source.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO,
		token.LSS, token.LEQ, token.GTR, token.GEQ,
		token.EQL, token.NEQ:
	default:
		return nil, false
	}
	left := context.TypesInfo().TypeOf(source.X)
	right := context.TypesInfo().TypeOf(source.Y)
	leftCarrier, leftOK := floatvalue.Describe(left)
	rightCarrier, rightOK := floatvalue.Describe(right)
	if !leftOK || !rightOK ||
		leftCarrier.Bits() != 64 || rightCarrier.Bits() != 64 {
		return nil, false
	}
	return left, true
}

func targetOperator(operator token.Token) (tsgo.BinaryOperator, bool) {
	switch operator {
	case token.ADD:
		return tsgo.BinaryOperatorPlusToken, true
	case token.SUB:
		return tsgo.BinaryOperatorMinusToken, true
	case token.MUL:
		return tsgo.BinaryOperatorAsteriskToken, true
	case token.QUO:
		return tsgo.BinaryOperatorSlashToken, true
	case token.LSS:
		return tsgo.BinaryOperatorLessThanToken, true
	case token.LEQ:
		return tsgo.BinaryOperatorLessThanEqualsToken, true
	case token.GTR:
		return tsgo.BinaryOperatorGreaterThanToken, true
	case token.GEQ:
		return tsgo.BinaryOperatorGreaterThanEqualsToken, true
	case token.EQL:
		return tsgo.BinaryOperatorEqualsEqualsEqualsToken, true
	case token.NEQ:
		return tsgo.BinaryOperatorExclamationEqualsEqualsToken, true
	default:
		return 0, false
	}
}
