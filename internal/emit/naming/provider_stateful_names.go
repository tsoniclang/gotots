package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
)

func (n *File) providerStatefulOperation(
	owner *types.TypeName,
	capability gostdlib.FacetCapability,
	phase api.ImportPhase,
	facet api.ArtifactFacet,
) (api.NameReference, bool, error) {
	candidates, profiled, err := n.ProviderStatefulProfileCandidates(owner)
	if err != nil || !profiled {
		return api.NameReference{}, profiled, err
	}
	for _, candidate := range candidates {
		if !candidate.Profile().SupportsOperation(capability) {
			return api.NameReference{}, true, &api.NameError{
				Name: owner.Name(),
				Reason: "provider stateful profile omits named-struct operation " +
					string(capability),
			}
		}
	}
	reference, selected, err := n.providerStatefulRepresentation(
		owner,
		phase,
		facet,
	)
	if err != nil || !selected {
		return api.NameReference{}, true, err
	}
	return reference, true, nil
}

func (n *File) providerStatefulRepresentation(
	owner *types.TypeName,
	phase api.ImportPhase,
	_ api.ArtifactFacet,
) (api.NameReference, bool, error) {
	if owner == nil || owner.IsAlias() {
		return api.NameReference{}, false, nil
	}
	named, ok := types.Unalias(owner.Type()).(*types.Named)
	if !ok || named.Obj() == nil {
		return api.NameReference{}, false, nil
	}
	_, profiled, err := n.ProviderStatefulProfileCandidates(owner)
	if err != nil || !profiled {
		return api.NameReference{}, profiled, err
	}
	artifactKey, err := typeidentity.BuildKey(
		named,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	binding, err := n.owner.registry.internProviderStatefulRepresentation(
		artifactKey,
		named,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	requirement, err := api.NewProviderStatefulRepresentationRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, true, err
	}
	facet := api.ArtifactFacetInstanceTypeSurface
	if phase == api.ImportPhaseValue {
		facet = api.ArtifactFacetValueSurface
	}
	reference, err := n.generatedReference(
		binding.owner,
		binding.name,
		requirement,
		facet,
		phase,
	)
	return reference, true, err
}
