package anonymousstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Resolve(sourceType types.Type) (*types.Struct, bool) {
	if sourceType == nil {
		return nil, false
	}
	structType, ok := types.Unalias(sourceType).(*types.Struct)
	return structType, ok
}

func Emit(
	context api.Context,
	_ api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	structType, ok := Resolve(sourceType)
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().AnonymousStruct(
		structType,
		api.AnonymousStructDemandDefinition,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			nil,
		),
		reference.Requests()...,
	), nil
}
