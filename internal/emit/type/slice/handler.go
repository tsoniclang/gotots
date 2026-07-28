package slice

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitSyntax(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ArrayType,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if source == nil || source.Len != nil {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	return EmitRepresented(context, children, source, sourceType)
}

func EmitRepresented(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	_, elementType, ok := slicevalue.Resolve(sourceType)
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	element, err := children.RepresentedType(
		context.WithRole(api.RoleSliceElementType),
		source,
		elementType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeSlice,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(runtime.Name()),
			[]tsgo.TypeNode{element.Value()},
		),
		api.CombineRequests(element.Requests(), runtime.Requests())...,
	), nil
}
