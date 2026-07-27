package float

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// Emit handles float32 and float64 arithmetic and ordering. IEEE-754 semantics
// carry directly to TypeScript number operators: `/` by zero yields ±Infinity or
// NaN (never a panic, unlike integer division), NaN compares false under every
// ordering, and signed zeros compare equal. A float32 arithmetic result is
// rounded to binary32 through the goFloat32 helper — double rounding
// binary64→binary32 is exact for these operations — while comparisons produce a
// boolean and need no rounding. Equality (`==`/`!=`) on non-primitive types is
// owned by the value-equality path before this handler runs; scalar float
// equality is handled here.
func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	operandType, carrier, ok := floatOperation(context, source)
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
	value := tsgo.Expression(context.Factory().BinaryExpression(
		nil,
		left.Value(),
		nil,
		context.Factory().BinaryOperatorToken(operator),
		right.Value(),
	))
	requests := api.CombineRequests(left.Requests(), right.Requests())
	if carrier.Bits() == 32 && isArithmetic(source.Op) {
		reference, err := context.Names().Runtime(
			api.RuntimeFloat32Round,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		value = context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		)
		requests = api.CombineRequests(requests, reference.Requests())
	}
	return api.DirectExpression(value, requests...), true, nil
}

func isArithmetic(operator token.Token) bool {
	switch operator {
	case token.ADD, token.SUB, token.MUL, token.QUO:
		return true
	default:
		return false
	}
}

// floatOperation returns the operand type and carrier when this binary
// expression is a float arithmetic or ordering operation whose operands are both
// the same float type.
func floatOperation(
	context api.Context,
	source *ast.BinaryExpr,
) (types.Type, floatvalue.Carrier, bool) {
	switch source.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO,
		token.LSS, token.LEQ, token.GTR, token.GEQ,
		token.EQL, token.NEQ:
	default:
		return nil, floatvalue.Carrier{}, false
	}
	left := context.TypesInfo().TypeOf(source.X)
	right := context.TypesInfo().TypeOf(source.Y)
	leftCarrier, leftOK := floatvalue.Describe(left)
	rightCarrier, rightOK := floatvalue.Describe(right)
	if !leftOK || !rightOK || leftCarrier.Bits() != rightCarrier.Bits() {
		return nil, floatvalue.Carrier{}, false
	}
	return left, leftCarrier, true
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
