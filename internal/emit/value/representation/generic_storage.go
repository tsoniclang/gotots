package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) genericStorageType(
	context api.Context,
	source ast.Node,
	parameter *types.TypeParam,
	facet api.GenericRepresentationFacet,
	runtime api.RuntimeSymbol,
) (api.TypeEmission, error) {
	selectionRequests, err := context.RequireGenericParameterRepresentation(
		parameter,
		facet,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	logical, err := owner.children.RepresentedType(
		context,
		source,
		parameter,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	projection, err := context.Names().Runtime(runtime, api.ImportPhaseType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			projection.EntityName(context.Factory()),
			[]tsgo.TypeNode{logical.Value()},
		),
		api.CombineRequests(
			selectionRequests,
			logical.Requests(),
			projection.Requests(),
		)...,
	), nil
}
