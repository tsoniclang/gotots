package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func CopyAggregate(
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
			Reason: "aggregate copy requires target and source slices",
		}
	}
	targetName, sourceName, countName, snapshotName, indexName, err :=
		copyTemporaryNames(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := context.Factory().Identifier(targetName)
	sourceValue := context.Factory().Identifier(sourceName)
	count := context.Factory().Identifier(countName)
	snapshot := context.Factory().Identifier(snapshotName)
	index := context.Factory().Identifier(indexName)
	next, err := context.Values().Transfer(
		context.WithRole(api.RoleSliceElement),
		nil,
		elementType,
		elementType,
		api.ValueTransferCopy,
		api.DirectExpression(sliceCall(
			context,
			sourceValue,
			runtimeslice.MemberName(runtimeslice.MemberGet),
			index,
		)),
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
	statements := append([]tsgo.Statement(nil), before...)
	statements = append(
		statements,
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			targetName,
			operands[0],
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			sourceName,
			operands[1],
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			countName,
			context.Factory().CallExpression(
				sliceProperty(
					context,
					context.Factory().Identifier("Math"),
					"min",
				),
				nil,
				nil,
				[]tsgo.Expression{
					sliceProperty(
						context,
						target,
						runtimeslice.MemberName(runtimeslice.MemberLength),
					),
					sliceProperty(
						context,
						sourceValue,
						runtimeslice.MemberName(runtimeslice.MemberLength),
					),
				},
				tsgo.NodeFlagsNone,
			),
		),
		sliceVariable(
			context,
			tsgo.NodeFlagsLet,
			snapshotName,
			sourceValue,
		),
		context.Factory().IfStatement(
			sliceBinary(
				context,
				count,
				tsgo.BinaryOperatorGreaterThanToken,
				sliceNumber(context, "0"),
			),
			context.Factory().Block(
				copyAggregateStatements(
					context,
					target,
					sourceValue,
					count,
					snapshot,
					index,
					next,
					targetElement.Value(),
					allocation.Name(),
				),
				true,
			),
			nil,
		),
	)
	return api.NewExpressionEmission(
		statements,
		count,
		api.CombineRequests(
			requests,
			targetElement.Requests(),
			next.Requests(),
			allocation.Requests(),
		),
	)
}

func copyTemporaryNames(
	context api.Context,
) (string, string, string, string, string, error) {
	names := make([]string, 5)
	for index := range names {
		name, err := context.Names().Temporary(api.TemporarySliceConstruction)
		if err != nil {
			return "", "", "", "", "", err
		}
		names[index] = name
	}
	return names[0], names[1], names[2], names[3], names[4], nil
}

func copyAggregateStatements(
	context api.Context,
	target tsgo.Expression,
	source tsgo.Expression,
	count tsgo.Expression,
	snapshot tsgo.Expression,
	index tsgo.Identifier,
	next api.ExpressionEmission,
	targetElement tsgo.TypeNode,
	allocationName string,
) []tsgo.Statement {
	statements := []tsgo.Statement{sliceAssign(
		context,
		snapshot,
		context.Factory().CallExpression(
			context.Factory().Identifier(allocationName),
			nil,
			[]tsgo.TypeNode{targetElement},
			[]tsgo.Expression{
				count,
				context.Factory().NullLiteral(),
			},
			tsgo.NodeFlagsNone,
		),
	)}
	statements = append(statements, sliceLoop(
		context,
		index,
		count,
		"0",
		append(
			next.Before(),
			context.Factory().ExpressionStatement(sliceCall(
				context,
				snapshot,
				runtimeslice.MemberName(runtimeslice.MemberSet),
				index,
				next.Value(),
			)),
		),
	))
	statements = append(statements, sliceLoop(
		context,
		index,
		count,
		"0",
		[]tsgo.Statement{context.Factory().ExpressionStatement(sliceCall(
			context,
			target,
			runtimeslice.MemberName(runtimeslice.MemberSet),
			index,
			sliceCall(
				context,
				snapshot,
				runtimeslice.MemberName(runtimeslice.MemberGet),
				index,
			),
		))},
	))
	return statements
}
