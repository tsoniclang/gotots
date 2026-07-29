package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func AppendString(
	context api.Context,
	targetElement api.TypeEmission,
	source ast.Node,
	elementType types.Type,
	operands []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	if len(operands) != 2 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "string spread append requires slice and string operands",
		}
	}
	receiverName, textName, bytesName, indexName, err :=
		stringSpreadNames(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	allocation, err := context.Names().Runtime(
		api.RuntimeSliceStorage,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	appendSlice, err := context.Names().Runtime(
		api.RuntimeSliceAppendSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver := context.Factory().Identifier(receiverName)
	text := context.Factory().Identifier(textName)
	bytesValue := context.Factory().Identifier(bytesName)
	index := context.Factory().Identifier(indexName)
	character := tsgo.Expression(sliceCall(
		context,
		text,
		"charCodeAt",
		index,
	))
	if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
		character = context.Factory().CallExpression(
			context.Factory().Identifier("BigInt"),
			nil,
			nil,
			[]tsgo.Expression{character},
			tsgo.NodeFlagsNone,
		)
	}
	statements := append([]tsgo.Statement(nil), before...)
	statements = append(statements, zero.Before()...)
	statements = append(
		statements,
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			receiverName,
			operands[0],
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			textName,
			operands[1],
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			bytesName,
			context.Factory().CallExpression(
				context.Factory().Identifier(allocation.Name()),
				nil,
				[]tsgo.TypeNode{targetElement.Value()},
				[]tsgo.Expression{
					sliceProperty(context, text, "length"),
					context.Factory().NullLiteral(),
				},
				tsgo.NodeFlagsNone,
			),
		),
		sliceLoop(
			context,
			index,
			sliceProperty(context, text, "length"),
			"0",
			[]tsgo.Statement{context.Factory().ExpressionStatement(
				sliceCall(
					context,
					bytesValue,
					runtimeslice.MemberName(runtimeslice.MemberSet),
					index,
					character,
				),
			)},
		),
	)
	target := context.Factory().CallExpression(
		context.Factory().Identifier(appendSlice.Name()),
		nil,
		[]tsgo.TypeNode{targetElement.Value()},
		[]tsgo.Expression{receiver, bytesValue, zero.Value()},
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		statements,
		target,
		api.CombineRequests(
			requests,
			targetElement.Requests(),
			zero.Requests(),
			allocation.Requests(),
			appendSlice.Requests(),
		),
	)
}

func stringSpreadNames(
	context api.Context,
) (string, string, string, string, error) {
	names := make([]string, 4)
	for index := range names {
		name, err := context.Names().Temporary(
			api.TemporarySliceConstruction,
		)
		if err != nil {
			return "", "", "", "", err
		}
		names[index] = name
	}
	return names[0], names[1], names[2], names[3], nil
}
