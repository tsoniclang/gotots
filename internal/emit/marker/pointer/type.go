package pointer

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Type(
	context api.Context,
	element api.TypeEmission,
	nullable bool,
) (api.TypeEmission, error) {
	reference, err := context.Names().TsonicCore(tsoniccore.SymbolPointer)
	if err != nil {
		return api.TypeEmission{}, err
	}
	target := tsgo.TypeNode(context.Factory().TypeReferenceNode(
		reference.EntityName(context.Factory()),
		[]tsgo.TypeNode{element.Value()},
	))
	if nullable {
		target = context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		})
	}
	return api.DirectType(
		target,
		api.CombineRequests(element.Requests(), reference.Requests())...,
	), nil
}
