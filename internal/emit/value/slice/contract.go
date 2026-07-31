package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Resolve(
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	if sourceType == nil {
		return nil, nil, false
	}
	sourceSlice, ok := types.Unalias(sourceType).(*types.Slice)
	if !ok {
		return nil, nil, false
	}
	return sourceSlice, sourceSlice.Elem(), true
}

func RangeLength(
	context api.Context,
	receiver tsgo.Expression,
) tsgo.Expression {
	return context.Factory().PropertyAccessExpression(
		receiver,
		nil,
		context.Factory().Identifier(
			runtimeslice.MemberName(runtimeslice.MemberLength),
		),
		tsgo.NodeFlagsNone,
	)
}

func RangeElement(
	context api.Context,
	source ast.Node,
	elementType types.Type,
	receiver tsgo.Expression,
	index tsgo.Expression,
) (api.ExpressionEmission, error) {
	return context.ContainerStorage().FromContainerStorage(
		context.WithRole(api.RoleSliceElement),
		source,
		elementType,
		api.DirectExpression(context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				receiver,
				nil,
				context.Factory().Identifier(
					runtimeslice.MemberName(runtimeslice.MemberGet),
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{index},
			tsgo.NodeFlagsNone,
		)),
	)
}

func Source(
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	if defined, ok := definedtype.ResolveSlice(sourceType); ok {
		sliceType, _ := defined.Slice()
		return sliceType, sliceType.Elem(), true
	}
	return Resolve(sourceType)
}

func Project(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	defined, ok := definedtype.ResolveSlice(sourceType)
	if !ok {
		return value, nil
	}
	return defined.Project(context, value)
}

func Wrap(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	defined, ok := definedtype.ResolveSlice(sourceType)
	if !ok {
		return value, nil
	}
	return defined.Wrap(context, value)
}
