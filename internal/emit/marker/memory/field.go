package memory

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func fieldLayout(context api.Context, children api.ChildEmitter, source ast.Node, represented api.TypeEmission, name string, fieldType types.Type, offset int64) (api.ExpressionEmission, error) {
	childLayout, _, err := Layout(context, children, source, fieldType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return selectedFieldLayout(context, represented, name, offset, context.TypesSizes().Alignof(fieldType), childLayout)
}

func selectedFieldLayout(context api.Context, represented api.TypeEmission, name string, offset, alignment int64, childLayout api.ExpressionEmission) (api.ExpressionEmission, error) {
	parameter, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	selector := context.Factory().ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
		context.Factory().ParameterDeclaration(nil, nil, context.Factory().Identifier(parameter), nil, represented.Value(), nil),
	}, nil, context.Factory().EqualsGreaterThanToken(), context.Factory().PropertyAccessExpression(
		context.Factory().Identifier(parameter), nil, context.Factory().Identifier(name), tsgo.NodeFlagsNone))
	number := func(value int64) api.ExpressionEmission {
		return api.DirectExpression(context.Factory().NumericLiteral(strconv.FormatInt(value, 10), tsgo.TokenFlagsNone))
	}
	return pointermarker.Operation(context, tsoniccore.SymbolMemoryField, nil, []api.ExpressionEmission{
		api.DirectExpression(selector, represented.Requests()...), number(offset), number(alignment), childLayout,
	})
}
