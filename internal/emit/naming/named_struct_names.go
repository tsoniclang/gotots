package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (n *File) NamedStructOperation(
	typeName *types.TypeName,
	operation api.NamedStructOperation,
) (api.NameReference, error) {
	request, err := n.namedStructOperationRequest(typeName, operation)
	if err != nil {
		return api.NameReference{}, err
	}
	capability, err := providerNamedStructCapability(operation)
	if err != nil {
		return api.NameReference{}, err
	}
	providerReference, providerOwned, err := n.providerFacetReference(
		typeName,
		gostdlib.FacetNamedStructOperations,
		capability,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if providerOwned {
		return providerReference.WithRequests(
			api.CombineRequests(
				providerReference.Requests(),
				[]api.RootRequest{request},
			)...,
		)
	}
	reference, err := n.reference(
		typeName,
		api.ImportPhaseValue,
		api.ArtifactFacetStaticSurface,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests := append(reference.Requests(), request)
	return reference.WithRequests(requests...)
}

func (n *File) NamedStructConstructor(
	typeName *types.TypeName,
) (api.NameReference, error) {
	providerReference, providerOwned, err := n.providerFacetReference(
		typeName,
		gostdlib.FacetNamedStructOperations,
		gostdlib.FacetCapabilityMake,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if providerOwned {
		return providerReference, nil
	}
	return n.Reference(typeName)
}

func (n *File) namedStructOperationRequest(
	typeName *types.TypeName,
	operation api.NamedStructOperation,
) (api.RootRequest, error) {
	if typeName != nil &&
		typeName.Pkg() != nil &&
		typeName.Parent() != nil &&
		typeName.Parent() != typeName.Pkg().Scope() {
		placement, placementErr := n.generatedArtifactPlacement(
			typeName.Type(),
		)
		if placementErr != nil {
			return api.RootRequest{}, placementErr
		}
		if placement.kind != api.GeneratedArtifactPlacementLexical ||
			placement.anchor != typeName {
			return api.RootRequest{}, &api.NameError{
				Name:   typeName.Name(),
				Reason: "local named-struct operation has no exact lexical owner",
			}
		}
		return api.NewLexicalNamedStructOperationRequest(
			placement.lexicalOwner,
			typeName,
			operation,
		)
	}
	return api.NewNamedStructOperationRequest(
		typeName,
		operation,
	)
}
