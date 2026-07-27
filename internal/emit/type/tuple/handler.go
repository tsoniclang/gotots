package tuple

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType *types.Tuple,
) (api.TypeEmission, error) {
	if sourceType == nil || sourceType.Len() < 2 {
		return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
	}
	elements := make([]tsgo.TypeNode, 0, sourceType.Len())
	var requests []api.RootRequest
	for index := range sourceType.Len() {
		element, err := children.RepresentedType(
			context,
			source,
			sourceType.At(index).Type(),
		)
		if err != nil {
			return api.TypeEmission{}, err
		}
		elements = append(elements, element.Value())
		requests = append(requests, element.Requests()...)
	}
	return api.DirectType(
		context.Factory().TupleTypeNode(elements),
		requests...,
	), nil
}
