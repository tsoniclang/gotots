package namedstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	if _, ok := named.Underlying().(*types.Struct); !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().TypeReference(named.Obj())
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
