package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ClearAggregate(
	context api.Context,
	source ast.Node,
	elementType types.Type,
	receiver api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	receiverName, err := context.Names().Temporary(
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
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := context.Factory().Identifier(receiverName)
	index := context.Factory().Identifier(indexName)
	before := append(
		receiver.Before(),
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			receiverName,
			receiver.Value(),
		),
		sliceLoop(
			context,
			index,
			sliceProperty(
				context,
				target,
				runtimeslice.MemberName(runtimeslice.MemberLength),
			),
			"0",
			append(
				zero.Before(),
				context.Factory().ExpressionStatement(sliceCall(
					context,
					target,
					runtimeslice.MemberName(runtimeslice.MemberSet),
					index,
					zero.Value(),
				)),
			),
		),
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().VoidExpression(
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		),
		api.CombineRequests(receiver.Requests(), zero.Requests()),
	)
}
