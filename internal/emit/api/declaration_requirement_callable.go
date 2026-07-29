package api

import "go/types"

func NewCooperativeCallableRequirement(
	facet CallableFacet,
) (DeclarationRequirement, error) {
	if !facet.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "cooperative callable facet is invalid",
		}
	}
	return DeclarationRequirement{
		owner:         facet.Owner(),
		kind:          DeclarationRequirementCooperativeCallable,
		callableFacet: facet,
	}, nil
}

func NewCooperativeCallableRequest(
	facet CallableFacet,
) (RootRequest, error) {
	requirement, err := NewCooperativeCallableRequirement(facet)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewCallableABIRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactCallableABI,
		DeclarationRequirementCallableABI,
		"callable ABI",
	)
}

func NewCallableABIRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewCallableABIRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) CooperativeCallable() (
	CallableFacet,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementCooperativeCallable {
		return CallableFacet{}, false
	}
	return r.callableFacet, true
}

func (r DeclarationRequirement) CallableABI() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementCallableABI,
		GeneratedArtifactCallableABI,
	)
}

func (r DeclarationRequirement) validCooperativeCallable() bool {
	if !r.owner.Valid() ||
		r.operation != NamedStructOperationInvalid ||
		r.typeName != nil ||
		r.variable != nil ||
		r.constant != nil ||
		r.projection != types.Invalid ||
		r.generated != nil ||
		r.anonymousDemand != AnonymousStructDemandInvalid ||
		r.mapDemand != MapSpecializationDemandInvalid ||
		r.genericOperation != nil ||
		!r.callableFacet.Valid() {
		return false
	}
	return r.owner == r.callableFacet.Owner()
}
