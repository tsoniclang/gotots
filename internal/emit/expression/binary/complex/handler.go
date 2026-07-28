package complex

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
	symbol, ok := complexvalue.BinarySymbol(resultCarrier, source.Op)
	if !ok {
		return api.ExpressionEmission{}, true,
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "complex binary operation has no runtime symbol",
			}
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
	if len(left.Before()) != 0 || len(right.Before()) != 0 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := complexvalue.Call(
		context,
		symbol,
		[]tsgo.Expression{left.Value(), right.Value()},
		api.CombineRequests(left.Requests(), right.Requests())...,
	)
	return target, true, err
}
