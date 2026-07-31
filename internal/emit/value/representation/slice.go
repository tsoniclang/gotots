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
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	_, sourceElementType, ok := scalarSlice(context, sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	elementType, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleSliceElementType),
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
	callee := context.Factory().PropertyAccessExpression(
		context.Factory().Identifier(runtime.Name()),
		nil,
		context.Factory().Identifier(
			runtimeslice.MemberName(runtimeslice.MemberNil),
		),
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		nil,
		context.Factory().CallExpression(
			callee,
			nil,
			[]tsgo.TypeNode{elementType.Value()},
			nil,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			elementType.Requests(),
			runtime.Requests(),
		),
	)
}
