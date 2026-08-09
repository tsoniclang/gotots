package rawpointer

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Type(context api.Context, nullable bool) (api.TypeEmission, error) {
	reference, err := context.Names().TsonicCore(tsoniccore.SymbolRawPointer)
	if err != nil {
		return api.TypeEmission{}, err
	}
	target := tsgo.TypeNode(context.Factory().TypeReferenceNode(
		reference.EntityName(context.Factory()),
		nil,
	))
	if nullable {
		target = context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		})
	}
	return api.DirectType(target, reference.Requests()...), nil
}
