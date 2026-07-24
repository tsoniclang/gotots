package scope

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/structure"
	providercontract "github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

// DefinitionSelection is the one provider/depth overlay record of a
// structural definition.
type DefinitionSelection struct {
	definition          identity.DefinitionID
	provider            providercontract.Provider
	depth               providercontract.EvidenceDepth
	contractID          string
	contractFingerprint string
	witness             providercontract.Witness
	facts               []selectionfacts.ID
}

func (s DefinitionSelection) Definition() identity.DefinitionID     { return s.definition }
func (s DefinitionSelection) Provider() providercontract.Provider   { return s.provider }
func (s DefinitionSelection) Depth() providercontract.EvidenceDepth { return s.depth }
func (s DefinitionSelection) ContractID() string                    { return s.contractID }
func (s DefinitionSelection) ContractFingerprint() string {
	return s.contractFingerprint
}
func (s DefinitionSelection) Witness() providercontract.Witness {
	witness := s.witness
	witness.Facts = append(
		[]providercontract.SelectionFactKind(nil), witness.Facts...,
	)
	return witness
}
func (s DefinitionSelection) Facts() []selectionfacts.ID {
	return append([]selectionfacts.ID(nil), s.facts...)
}

// DefinitionSelections is the immutable total selection overlay.
type DefinitionSelections struct {
	records []DefinitionSelection
	byID    map[identity.DefinitionID]*DefinitionSelection
}

func (s *DefinitionSelections) Records() []DefinitionSelection {
	return append([]DefinitionSelection(nil), s.records...)
}
func (s *DefinitionSelections) For(
	definition identity.DefinitionID,
) (DefinitionSelection, bool) {
	record, ok := s.byID[definition]
	if !ok {
		return DefinitionSelection{}, false
	}
	return *record, true
}

// SelectDefinitions binds every definition exactly once. It consumes only the
// structural graph, closed selection facts, package disposition, and the
// provider contract.
func SelectDefinitions(
	universe *source.Universe,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
	selected providercontract.Contract,
) (*DefinitionSelections, error) {
	dispositions := map[identity.PackageID]source.LanguageDisposition{}
	for _, pkg := range universe.Packages() {
		dispositions[pkg.ID()] = pkg.Disposition()
	}
	out := &DefinitionSelections{
		byID: map[identity.DefinitionID]*DefinitionSelection{},
	}
	definitionCount := 0
	for _, indexed := range graph.DefinitionCensus() {
		packageID := indexed.Package()
		definition := indexed.ID()
		disposition, present := dispositions[packageID]
		if !present {
			return nil, fmt.Errorf(
				"scope package %s is absent from source universe",
				packageID,
			)
		}
		definitionCount++
		values := map[providercontract.SelectionFactKind]bool{}
		var factIDs []selectionfacts.ID
		for _, kind := range selected.RequestedFacts(
			definition, packageID,
		) {
			value, exists := facts.Value(definition, kind)
			if !exists {
				return nil, fmt.Errorf(
					"definition %s lacks requested selection fact %s",
					definition, kind,
				)
			}
			values[kind] = value
			id, _ := selectionfacts.NewID(definition, kind)
			factIDs = append(factIDs, id)
		}
		provider, witness, err := selected.Bind(providercontract.Query{
			Definition: definition,
			Package:    packageID,
			Intrinsic:  disposition == source.DispositionUnsafeIntrinsic,
			Facts:      values,
		})
		if err != nil {
			return nil, err
		}
		depth := provider.Depth()
		if err := validateCompatibility(
			definition,
			depth,
			disposition,
		); err != nil {
			return nil, err
		}
		record := DefinitionSelection{
			definition: definition, provider: provider, depth: depth,
			contractID: selected.ID(), contractFingerprint: selected.Fingerprint(),
			witness: witness, facts: factIDs,
		}
		out.records = append(out.records, record)
	}
	sort.Slice(out.records, func(i, j int) bool {
		return out.records[i].definition.Compare(
			out.records[j].definition,
		) < 0
	})
	for index := range out.records {
		record := &out.records[index]
		if _, duplicate := out.byID[record.definition]; duplicate {
			return nil, fmt.Errorf(
				"duplicate selection for %s", record.definition,
			)
		}
		out.byID[record.definition] = record
	}
	if len(out.records) != definitionCount {
		return nil, fmt.Errorf(
			"selection cardinality %d does not match definition cardinality %d",
			len(out.records), definitionCount,
		)
	}
	return out, nil
}

func validateCompatibility(
	definition identity.DefinitionID,
	depth providercontract.EvidenceDepth,
	disposition source.LanguageDisposition,
) error {
	if definition.IsZero() ||
		!definition.Kind().Valid() ||
		!depth.Valid() ||
		!disposition.Valid() {
		return fmt.Errorf(
			"definition compatibility requires valid identity, depth, and disposition",
		)
	}
	intrinsicDisposition :=
		disposition == source.DispositionBuiltinUniverse ||
			disposition == source.DispositionUnsafeIntrinsic
	if depth == providercontract.DepthIntrinsic {
		if !intrinsicDisposition ||
			definition.SyntheticRole().Valid() {
			return fmt.Errorf(
				"definition %s cannot use intrinsic evidence",
				definition,
			)
		}
		return nil
	}
	if intrinsicDisposition {
		return fmt.Errorf(
			"intrinsic definition %s cannot use %s evidence",
			definition,
			depth,
		)
	}
	switch definition.Kind() {
	case identity.DefinitionBodylessDecl:
		if depth == providercontract.DepthFullSemantic {
			return fmt.Errorf("bodyless definition %s cannot be full-semantic", definition)
		}
	case identity.DefinitionImplicit:
		if definition.SyntheticRole().Valid() &&
			depth != providercontract.DepthExternalBoundary {
			return fmt.Errorf("synthetic definition %s must be external-boundary", definition)
		}
	}
	return nil
}
