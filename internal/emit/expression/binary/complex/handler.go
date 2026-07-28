package complex

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	binaryoperands "github.com/tsoniclang/gotots/internal/emit/expression/binary/operands"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	switch source.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO:
	default:
		return api.ExpressionEmission{}, false, nil
	}
	resultType := context.TypesInfo().TypeOf(source)
	leftType := context.TypesInfo().TypeOf(source.X)
	rightType := context.TypesInfo().TypeOf(source.Y)
	resultCarrier, resultOK := complexvalue.Describe(resultType)
	leftCarrier, leftOK := complexvalue.Describe(leftType)
	rightCarrier, rightOK := complexvalue.Describe(rightType)
	if !resultOK ||
		!leftOK ||
		!rightOK ||
		resultCarrier.Bits() != leftCarrier.Bits() ||
		resultCarrier.Bits() != rightCarrier.Bits() ||
		!types.AssignableTo(leftType, resultType) ||
		!types.AssignableTo(rightType, resultType) {
		return api.ExpressionEmission{}, false, nil
	}
	left, err := children.Expression(
		context.WithRole(api.RoleBinaryLeft).WithExpectedType(resultType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err := children.Expression(
		context.WithRole(api.RoleBinaryRight).WithExpectedType(resultType),
		source.Y,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	operands, err := binaryoperands.Preserve(
		context,
		left,
		right,
		api.TemporaryBinaryOperand,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, handled, err := Apply(
		context,
		source.Op,
		resultCarrier,
		operands.Left(),
		operands.Right(),
	)
	if err != nil || !handled {
		return target, handled, err
	}
	target, err = binaryoperands.Finish(operands, target)
	return target, true, err
}

func Apply(
	context api.Context,
	operator token.Token,
	carrier complexvalue.Carrier,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	symbol, ok := complexvalue.BinarySymbol(carrier, operator)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	target, err := complexvalue.Call(
		context,
		symbol,
		[]tsgo.Expression{left.Value(), right.Value()},
		api.CombineRequests(left.Requests(), right.Requests())...,
	)
	return target, true, err
}
