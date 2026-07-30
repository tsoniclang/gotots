package artifact

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type Contract struct {
	facets      [api.ArtifactFacetExportSurface + 1][]byte
	present     uint8
	exports     []string
	exportsSet  bool
	initialized bool
}

type ContractValueError struct {
	Facet  api.ArtifactFacet
	Reason string
}

func (e *ContractValueError) Error() string {
	if e.Facet.Valid() {
		return fmt.Sprintf(
			"construct observable contract facet %s: %s",
			e.Facet,
			e.Reason,
		)
	}
	return fmt.Sprintf("construct observable contract: %s", e.Reason)
}

func NewContract() Contract {
	return Contract{initialized: true}
}

func NewContractFacet(
	facet api.ArtifactFacet,
	encoded []byte,
) (Contract, error) {
	return NewContract().WithFacet(facet, encoded)
}

func (c Contract) WithFacet(
	facet api.ArtifactFacet,
	encoded []byte,
) (Contract, error) {
	if !c.initialized {
		return Contract{}, &ContractValueError{
			Facet:  facet,
			Reason: "base contract is absent",
		}
	}
	if !facet.Valid() {
		return Contract{}, &ContractValueError{
			Facet:  facet,
			Reason: "facet is invalid",
		}
	}
	if facet == api.ArtifactFacetExportSurface {
		return Contract{}, &ContractValueError{
			Facet:  facet,
			Reason: "export surface is owned by declaration projection",
		}
	}
	if len(encoded) == 0 {
		return Contract{}, &ContractValueError{
			Facet:  facet,
			Reason: "canonical encoding is empty",
		}
	}
	c.facets[facet] = bytes.Clone(encoded)
	c.present |= uint8(1) << facet
	return c, nil
}

func (c Contract) withOwnedExports(exports []string) (Contract, error) {
	if !c.hasFacet(api.ArtifactFacetExportSurface) {
		return Contract{}, &ContractValueError{
			Facet:  api.ArtifactFacetExportSurface,
			Reason: "export surface facet is absent",
		}
	}
	for index, name := range exports {
		if name == "" {
			return Contract{}, &ContractValueError{
				Facet:  api.ArtifactFacetExportSurface,
				Reason: "export binding name is empty",
			}
		}
		if index != 0 && exports[index-1] >= name {
			return Contract{}, &ContractValueError{
				Facet:  api.ArtifactFacetExportSurface,
				Reason: "export bindings are not unique and sorted",
			}
		}
	}
	c.exports = slices.Clone(exports)
	c.exportsSet = true
	return c, nil
}

func (c Contract) ExportedBindings() ([]string, bool) {
	if !c.hasFacet(api.ArtifactFacetExportSurface) ||
		!c.exportsSet {
		return nil, false
	}
	return slices.Clone(c.exports), true
}

func (c Contract) withOwnedFacet(
	facet api.ArtifactFacet,
	encoded []byte,
) (Contract, error) {
	if !c.initialized {
		return Contract{}, &ContractValueError{
			Facet:  facet,
			Reason: "base contract is absent",
		}
	}
	if !facet.Valid() {
		return Contract{}, &ContractValueError{
			Facet:  facet,
			Reason: "facet is invalid",
		}
	}
	if len(encoded) == 0 {
		return Contract{}, &ContractValueError{
			Facet:  facet,
			Reason: "canonical encoding is empty",
		}
	}
	c.facets[facet] = encoded
	c.present |= uint8(1) << facet
	return c, nil
}

func (c Contract) facet(
	facet api.ArtifactFacet,
) ([]byte, bool) {
	if !facet.Valid() || c.present&(uint8(1)<<facet) == 0 {
		return nil, false
	}
	return c.facets[facet], true
}

func (c Contract) hasFacet(facet api.ArtifactFacet) bool {
	_, ok := c.facet(facet)
	return ok
}

func validateArtifactContract(
	owner api.ArtifactOwner,
	contract Contract,
) (Contract, error) {
	if !contract.initialized {
		return Contract{}, &GraphError{
			Object: owner,
			Reason: "target artifact observable contract is absent",
		}
	}
	if contract.hasFacet(api.ArtifactFacetExportSurface) !=
		contract.exportsSet {
		return Contract{}, &GraphError{
			Object: owner,
			Reason: "target artifact export surface is not declaration-projected",
		}
	}
	return contract, nil
}

func changedArtifactFacets(
	current Contract,
	next Contract,
) []api.ArtifactFacet {
	var changed []api.ArtifactFacet
	for facet := api.ArtifactFacetCallableSignature; facet <= api.ArtifactFacetExportSurface; facet++ {
		currentValue, currentOK := current.facet(facet)
		nextValue, nextOK := next.facet(facet)
		if currentOK != nextOK || !bytes.Equal(currentValue, nextValue) {
			changed = append(changed, facet)
		}
	}
	return changed
}

func equalArtifactContracts(left Contract, right Contract) bool {
	return left.initialized == right.initialized &&
		left.present == right.present &&
		len(changedArtifactFacets(left, right)) == 0
}
