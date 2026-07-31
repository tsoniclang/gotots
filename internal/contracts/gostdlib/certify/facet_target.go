package certify

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func verifyRepresentationReferences(
	facets []gostdlib.FacetDocument,
	representations map[string]gostdlib.ProviderRepresentationDocument,
) error {
	references := make(map[string]map[string]struct{})
	for _, facet := range facets {
		if facet.RepresentationExport == "" {
			continue
		}
		if references[facet.RepresentationExport] == nil {
			references[facet.RepresentationExport] = make(map[string]struct{})
		}
		references[facet.RepresentationExport][facet.SourceIdentity] = struct{}{}
	}
	for export, representation := range representations {
		selected := references[export]
		if len(selected) != len(representation.SourceTypes) {
			return certifyError(
				"build representation",
				export,
				"facet references do not cover the represented source types",
			)
		}
		for _, identity := range representation.SourceTypes {
			if _, ok := selected[identity]; !ok {
				return certifyError(
					"build representation",
					identity,
					"represented source type has no owning facet",
				)
			}
		}
	}
	return nil
}

func validateFacetTarget(seed facetSeed, target tsgo.ProjectExport) error {
	if seed.Kind != gostdlib.FacetNamedStructOperations {
		return nil
	}
	for _, capability := range seed.Capabilities {
		members, err := facetCapabilityMembers(capability)
		if err != nil {
			return err
		}
		for _, member := range members {
			if _, ok := target.ValueMember(member); !ok {
				return certifyError(
					"build facet",
					seed.Export+"."+member,
					"named-struct capability member is absent",
				)
			}
		}
	}
	return nil
}

func facetCapabilityMembers(
	capability gostdlib.FacetCapability,
) ([]string, error) {
	switch capability {
	case gostdlib.FacetCapabilityMake:
		return []string{"$make"}, nil
	case gostdlib.FacetCapabilityZero:
		return []string{"$zero"}, nil
	case gostdlib.FacetCapabilityCopy:
		return []string{"$copy"}, nil
	case gostdlib.FacetCapabilityEqual:
		return []string{"$equal"}, nil
	case gostdlib.FacetCapabilityHash:
		return []string{"$hash"}, nil
	case gostdlib.FacetCapabilityConvert:
		return []string{"$convert"}, nil
	case gostdlib.FacetCapabilityStorage:
		return []string{"$storageOf", "$fromStorage"}, nil
	case gostdlib.FacetCapabilityRepresentation:
		return nil, nil
	case gostdlib.FacetCapabilityAssign:
		return []string{"$assign"}, nil
	default:
		return nil, certifyError(
			"build facet",
			string(capability),
			"named-struct capability is invalid",
		)
	}
}

func singleImplementationOwner(name string, owners []string) (string, error) {
	if len(owners) != 1 {
		return "", certifyError(
			"build facet",
			name,
			fmt.Sprintf("target has %d implementation owners, want one", len(owners)),
		)
	}
	return owners[0], nil
}
