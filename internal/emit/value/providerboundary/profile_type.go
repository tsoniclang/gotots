package providerboundary

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitProfileInterfaceType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
	nonNil bool,
) (api.TypeEmission, bool, error) {
	profile := context.ProviderProfile()
	if len(profile) == 0 {
		return api.TypeEmission{}, false, nil
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil {
		return api.TypeEmission{}, false, nil
	}
	_, selected, err := providerProfileInterfaceCertificate(
		named,
		profile,
	)
	if err != nil || !selected {
		return api.TypeEmission{}, false, err
	}
	reference, found, referenceErr :=
		context.Names().ProviderProfileInterfaceBridge(named, profile)
	if referenceErr != nil || !found {
		if referenceErr != nil {
			return api.TypeEmission{}, true, referenceErr
		}
		return api.TypeEmission{}, true, boundaryInvariant(
			context,
			"provider profile-interface type bridge is absent",
		)
	}
	target := api.DirectType(
		context.Factory().TypeReferenceNode(
			reference.Contract().EntityName(context.Factory()),
			nil,
		),
		reference.Requests()...,
	)
	if nonNil {
		return target, true, nil
	}
	return api.DirectType(
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target.Value(),
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		target.Requests()...,
	), true, nil
}
