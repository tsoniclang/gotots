package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (n *File) TypeRepresentation(
	typeName *types.TypeName,
	facet api.TypeRepresentationFacet,
) ([]api.RootRequest, error) {
	if typeName == nil || !facet.Valid() {
		return nil, &api.NameError{
			Reason: "type-representation request is invalid",
		}
	}
	if typeName.Pkg() != nil &&
		typeName.Parent() != nil &&
		typeName.Parent() != typeName.Pkg().Scope() {
		placement, err := n.generatedArtifactPlacement(typeName.Type())
		if err != nil {
			return nil, err
		}
		if placement.kind != api.GeneratedArtifactPlacementLexical ||
			placement.anchor != typeName {
			return nil, &api.NameError{
				Name:   typeName.Name(),
				Reason: "local type representation has no exact lexical owner",
			}
		}
		request, err := api.NewLexicalTypeRepresentationRequest(
			placement.lexicalOwner,
			typeName,
			facet,
		)
		if err != nil {
			return nil, err
		}
		return []api.RootRequest{request}, nil
	}
	request, err := api.NewTypeRepresentationRequest(typeName, facet)
	if err != nil {
		return nil, err
	}
	return []api.RootRequest{request}, nil
}

func (n *File) AnonymousStructTypeRepresentation(
	structType *types.Struct,
	facet api.TypeRepresentationFacet,
) ([]api.RootRequest, error) {
	if structType == nil || structType.NumFields() == 0 || !facet.Valid() {
		return nil, &api.NameError{
			Reason: "anonymous-struct type representation is invalid",
		}
	}
	binding, err := n.anonymousStructBinding(structType)
	if err != nil {
		return nil, err
	}
	request, err := api.NewGeneratedTypeRepresentationRequest(
		binding.owner,
		facet,
	)
	if err != nil {
		return nil, err
	}
	requests := []api.RootRequest{request}
	if binding.owner.Placement() == api.GeneratedArtifactPlacementLexical {
		return requests, nil
	}
	if n.artifactOwner.Valid() &&
		n.artifactOwner != api.MustGeneratedArtifactOwner(binding.owner) {
		dependency, dependencyErr :=
			api.NewGeneratedArtifactDependencyRequest(
				binding.owner,
				api.ArtifactFacetInstanceTypeSurface,
			)
		if dependencyErr != nil {
			return nil, dependencyErr
		}
		requests = append(requests, dependency)
	}
	return requests, nil
}
