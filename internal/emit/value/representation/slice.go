package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func scalarSlice(
	context api.Context,
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	return slicevalue.Scalar(context.TypesSizes(), sourceType)
}

func isScalarSlice(context api.Context, sourceType types.Type) bool {
	_, _, ok := scalarSlice(context, sourceType)
	return ok
}

func sliceZero(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	_, sourceElementType, ok := scalarSlice(context, sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	elementType, requests, err := sliceElementTarget(context, source, sourceType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		sourceElementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		zero.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(
					runtimeslice.MemberName(runtimeslice.MemberNil),
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{elementType},
			[]tsgo.Expression{zero.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			requests,
			runtime.Requests(),
			zero.Requests(),
		),
	)
}

func sliceElementTarget(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (tsgo.TypeNode, []api.PlacementRequest, error) {
	_, elementType, ok := scalarSlice(context, sourceType)
	if !ok {
		return nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	alias, ok := api.PrimitiveAliasFor(context.TypesSizes(), elementType)
	if !ok {
		return nil, nil,
			api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().Primitive(alias)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().TypeReferenceNode(
		context.Factory().Identifier(reference.Name()),
		nil,
	), reference.Requests(), nil
}
