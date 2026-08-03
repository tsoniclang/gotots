package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func aggregateSliceCall(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	elementType types.Type,
	symbol api.RuntimeSymbol,
	arguments []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	element, err := children.RepresentedType(
		context.WithRole(api.RoleSliceElementType),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			[]tsgo.TypeNode{element.Value()},
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			requests,
			element.Requests(),
			runtime.Requests(),
		),
	)
}

func variable(
	context api.Context,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func bigInt(context api.Context, value tsgo.Expression) tsgo.CallExpression {
	return context.Factory().CallExpression(
		api.TargetIntrinsicBigInt.Expression(context.Factory()),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}
