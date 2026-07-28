package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func scalarSlice(
	_ api.Context,
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	return slicevalue.Resolve(sourceType)
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
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		sourceElementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var typeArguments []tsgo.TypeNode
	var typeRequests []api.RootRequest
	elementType, requests, represented, err := scalarSliceElementTarget(
		context,
		sourceElementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if represented {
		typeArguments = []tsgo.TypeNode{elementType}
		typeRequests = requests
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
			typeArguments,
			[]tsgo.Expression{zero.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			typeRequests,
			runtime.Requests(),
			zero.Requests(),
		),
	)
}

func scalarSliceElementTarget(
	context api.Context,
	elementType types.Type,
) (tsgo.TypeNode, []api.RootRequest, bool, error) {
	alias, ok := basictype.PrimitiveAlias(context.TypesSizes(), elementType)
	if !ok {
		return nil, nil, false, nil
	}
	reference, err := context.Names().Primitive(alias)
	if err != nil {
		return nil, nil, false, err
	}
	return context.Factory().TypeReferenceNode(
		context.Factory().Identifier(reference.Name()),
		nil,
	), reference.Requests(), true, nil
}
