package defined

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, bool, error) {
	model, ok := Resolve(sourceType)
	if !ok {
		return api.TypeEmission{}, false, nil
	}
	reference, err := context.Names().TypeReference(model.TypeName())
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	target := tsgo.TypeNode(context.Factory().TypeReferenceNode(
		context.Factory().Identifier(reference.Name()),
		nil,
	))
	if model.NilCapable() {
		target = context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		})
	}
	return api.DirectType(target, reference.Requests()...), true, nil
}
