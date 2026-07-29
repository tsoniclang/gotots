package generic

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, bool, error) {
	parameter, ok := api.GenericTypeParameter(sourceType)
	if !ok {
		return api.TypeEmission{}, false, nil
	}
	name, available := context.GenericParameterName(parameter)
	if !available {
		return api.TypeEmission{}, true,
			api.Unsupported(context, api.CategoryType, source)
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(name),
			nil,
		),
	), true, nil
}
