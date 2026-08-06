package slicevalue

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func CopyString(
	context api.Context,
	operands []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	if len(operands) != 2 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "string copy requires destination and source operands",
		}
	}
	destinationName, err := context.Names().Temporary(
		api.TemporarySliceConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceName, err := context.Names().Temporary(
		api.TemporarySliceConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	countName, err := context.Names().Temporary(
		api.TemporarySliceConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexName, err := context.Names().Temporary(
		api.TemporarySliceConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	destination := context.Factory().Identifier(destinationName)
	source := context.Factory().Identifier(sourceName)
	count := context.Factory().Identifier(countName)
	index := context.Factory().Identifier(indexName)
	character := tsgo.Expression(sliceCall(
		context,
		source,
		"charCodeAt",
		index,
	))
	statements := append([]tsgo.Statement(nil), before...)
	statements = append(
		statements,
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			destinationName,
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
					api.TargetIntrinsicMath.Expression(context.Factory()),
					"min",
				),
				nil,
				nil,
				[]tsgo.Expression{
					sliceProperty(
						context,
						destination,
						runtimeslice.MemberName(runtimeslice.MemberLength),
					),
					sliceProperty(context, source, "length"),
				},
				tsgo.NodeFlagsNone,
			),
		),
		sliceLoop(
			context,
			index,
			count,
			"0",
			[]tsgo.Statement{context.Factory().ExpressionStatement(
				sliceCall(
					context,
					destination,
					runtimeslice.MemberName(runtimeslice.MemberSet),
					index,
					character,
				),
			)},
		),
	)
	return api.NewExpressionEmission(statements, count, requests)
}
