// Selection ledger: the immutable output of the scope phase — one record per
// explicit and implicit unit, each carrying its provider, depth, and exact
// rule witness.
package scope

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

// UnitSelection is one source-spanned unit's immutable selection record.
type UnitSelection struct {
	Unit     identity.SourceUnitID
	Provider Provider
	Depth    source.EvidenceDepth
	Witness  BindingWitness
}

// ImplicitSelection is one implicit unit's immutable selection record.
type ImplicitSelection struct {
	Unit     identity.ImplicitUnitID
	Provider Provider
	Depth    source.EvidenceDepth
	Witness  BindingWitness
}

// Selection is the immutable, total per-unit evidence-depth selection over
// the complete explicit and implicit unit ledger.
type Selection struct {
	contractID          string
	contractFingerprint string
	units               []UnitSelection
	implicit            []ImplicitSelection
	depths              map[identity.SourceUnitID]source.EvidenceDepth
	implicitDepths      map[identity.ImplicitUnitID]source.EvidenceDepth
}

// ContractID is the identity of the request-selected contract artifact.
func (s *Selection) ContractID() string { return s.contractID }

// ContractFingerprint binds the selection to the contract that produced it.
func (s *Selection) ContractFingerprint() string { return s.contractFingerprint }

// Units is the ordered source-unit selection ledger (immutable copy).
func (s *Selection) Units() []UnitSelection { return append([]UnitSelection(nil), s.units...) }

// ImplicitUnits is the ordered implicit-unit selection ledger (immutable
// copy).
func (s *Selection) ImplicitUnits() []ImplicitSelection {
	return append([]ImplicitSelection(nil), s.implicit...)
}

// Depths is the per-unit depth map consumed by source.Finalize (fresh copy).
func (s *Selection) Depths() map[identity.SourceUnitID]source.EvidenceDepth {
	out := make(map[identity.SourceUnitID]source.EvidenceDepth, len(s.depths))
	for id, depth := range s.depths {
		out[id] = depth
	}
	return out
}

// ImplicitDepths is the per-implicit-unit depth map consumed by
// source.Finalize (fresh copy).
func (s *Selection) ImplicitDepths() map[identity.ImplicitUnitID]source.EvidenceDepth {
	out := make(map[identity.ImplicitUnitID]source.EvidenceDepth, len(s.implicitDepths))
	for id, depth := range s.implicitDepths {
		out[id] = depth
	}
	return out
}

// Select produces the immutable evidence-depth selection for every censused
// explicit and implicit unit of the universe under the given contract. The
// selection is total and disjoint by construction: exactly one record per
// unit. Every exact-selector rule must bind at least one unit — an exact rule
// naming nothing in the universe is a stale contract and fails closed.
func Select(u *source.Universe, contract ProviderContract) (*Selection, error) {
	if contract.version != ContractVersion {
		return nil, &SelectionError{Reason: fmt.Sprintf("contract version %d unsupported", contract.version)}
	}
	out := &Selection{
		contractID:          contract.id,
		contractFingerprint: contract.Fingerprint(),
		depths:              map[identity.SourceUnitID]source.EvidenceDepth{},
		implicitDepths:      map[identity.ImplicitUnitID]source.EvidenceDepth{},
	}
	fired := map[string]bool{}
	bind := func(q UnitQuery) (Provider, source.EvidenceDepth, BindingWitness, error) {
		provider, witness, err := contract.Bind(q)
		if err != nil {
			return ProviderInvalid, source.DepthInvalid, BindingWitness{}, err
		}
		depth := depthOf(provider)
		if !depth.Valid() {
			return ProviderInvalid, source.DepthInvalid, BindingWitness{}, &SelectionError{Reason: "no valid depth for unit " + q.unitString()}
		}
		fired[witness.RuleID] = true
		return provider, depth, witness, nil
	}
	for _, pkg := range u.Packages() {
		for _, file := range pkg.Files() {
			for _, unit := range file.Units() {
				provider, depth, witness, err := bind(UnitQuery{
					Unit: unit.ID(), Package: pkg.ID(), OwnerClass: pkg.ID().Owner().Class(),
					Disposition: pkg.Disposition(), Kind: unit.Kind(), CDependent: unit.CDependent(),
				})
				if err != nil {
					return nil, err
				}
				if _, dup := out.depths[unit.ID()]; dup {
					return nil, &SelectionError{Reason: "duplicate selection for unit " + unit.ID().String()}
				}
				out.depths[unit.ID()] = depth
				out.units = append(out.units, UnitSelection{Unit: unit.ID(), Provider: provider, Depth: depth, Witness: witness})
			}
		}
		for _, imp := range pkg.ImplicitUnits() {
			provider, depth, witness, err := bind(UnitQuery{
				Implicit: imp, Package: pkg.ID(), OwnerClass: pkg.ID().Owner().Class(),
				Disposition: pkg.Disposition(), Kind: identity.UnitImplicitExecutable,
			})
			if err != nil {
				return nil, err
			}
			if _, dup := out.implicitDepths[imp]; dup {
				return nil, &SelectionError{Reason: "duplicate selection for implicit unit " + imp.String()}
			}
			out.implicitDepths[imp] = depth
			out.implicit = append(out.implicit, ImplicitSelection{Unit: imp, Provider: provider, Depth: depth, Witness: witness})
		}
	}
	for _, rule := range contract.rules {
		if rule.selector != SelectorNamespace && !fired[rule.ID()] {
			return nil, &SelectionError{Reason: "contract " + contract.id + " exact rule " + rule.ID() +
				" binds nothing in the universe (stale contract)"}
		}
	}
	sort.Slice(out.units, func(i, j int) bool { return out.units[i].Unit.String() < out.units[j].Unit.String() })
	sort.Slice(out.implicit, func(i, j int) bool { return out.implicit[i].Unit.String() < out.implicit[j].Unit.String() })
	return out, nil
}
