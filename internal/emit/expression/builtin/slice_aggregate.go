package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func aggregateSliceZeroFactory(
	context api.Context,
	source ast.Node,
	elementType types.Type,
) (tsgo.ArrowFunction, []api.RootRequest, error) {
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(zero.Before()) != 0 {
		return nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return sliceValueFactory(context, nil, zero.Value()),
		zero.Requests(),
		nil
}

func aggregateSliceCopyFactory(
	context api.Context,
	source ast.Node,
	elementType types.Type,
) (tsgo.ArrowFunction, []api.RootRequest, error) {
	value := context.Factory().Identifier("$value")
	copied, err := context.Values().Copy(
		context.WithRole(api.RoleSliceElement),
		nil,
		elementType,
		api.DirectExpression(value),
	)
	if err != nil {
		return nil, nil, err
	}
	if len(copied.Before()) != 0 {
		return nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	parameter := context.Factory().ParameterDeclaration(
		nil,
		nil,
		value,
		nil,
		nil,
		nil,
	)
	return sliceValueFactory(
		context,
		[]tsgo.ParameterDeclaration{parameter},
		copied.Value(),
	), copied.Requests(), nil
}

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

func sliceValueFactory(
	context api.Context,
	parameters []tsgo.ParameterDeclaration,
	value tsgo.Expression,
) tsgo.ArrowFunction {
	return context.Factory().ArrowFunction(
		nil,
		nil,
		parameters,
		nil,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ReturnStatement(value)},
			true,
		),
	)
}
