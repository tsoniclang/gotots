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
	firedExact := map[string]bool{}
	for _, packageGraph := range graph.Packages() {
		disposition, present := dispositions[packageGraph.ID()]
		if !present {
			return nil, fmt.Errorf("scope package %s is absent from source universe", packageGraph.ID())
		}
		for _, definition := range packageGraph.Definitions() {
			values := map[providercontract.SelectionFactKind]bool{}
			var factIDs []selectionfacts.ID
			for _, kind := range selected.RequestedFacts(definition.ID(), packageGraph.ID()) {
				value, exists := facts.Value(definition.ID(), kind)
				if !exists {
					return nil, fmt.Errorf(
						"definition %s lacks requested selection fact %s",
						definition.ID(), kind,
					)
				}
				values[kind] = value
				id, _ := selectionfacts.NewID(definition.ID(), kind)
				factIDs = append(factIDs, id)
			}
			provider, witness, err := selected.Bind(providercontract.Query{
				Definition: definition.ID(),
				Package:    packageGraph.ID(),
				Intrinsic:  disposition == source.DispositionUnsafeIntrinsic,
				Facts:      values,
			})
			if err != nil {
				return nil, err
			}
			depth := provider.Depth()
			if err := validateCompatibility(definition, depth); err != nil {
				return nil, err
			}
			record := DefinitionSelection{
				definition: definition.ID(), provider: provider, depth: depth,
				contractID: selected.ID(), contractFingerprint: selected.Fingerprint(),
				witness: witness, facts: factIDs,
			}
			out.records = append(out.records, record)
			firedExact[witness.RuleID] = true
		}
	}
	for _, rule := range selected.Rules() {
		if rule.Selector() == providercontract.SelectorExactDefinition &&
			!firedExact[rule.ID()] {
			return nil, fmt.Errorf("stale exact-definition rule %s", rule.ID())
		}
	}
	sort.Slice(out.records, func(i, j int) bool {
		return out.records[i].definition.String() < out.records[j].definition.String()
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
	if len(out.records) != len(graph.Definitions()) {
		return nil, fmt.Errorf(
			"selection cardinality %d does not match definition cardinality %d",
			len(out.records), len(graph.Definitions()),
		)
	}
	return out, nil
}

func validateCompatibility(
	definition structure.ImplementationDefinition,
	depth providercontract.EvidenceDepth,
) error {
	if !depth.Valid() {
		return fmt.Errorf("definition %s has invalid evidence depth", definition.ID())
	}
	switch definition.Kind() {
	case identity.DefinitionBodylessDecl:
		if depth == providercontract.DepthFullSemantic {
			return fmt.Errorf("bodyless definition %s cannot be full-semantic", definition.ID())
		}
	case identity.DefinitionImplicit:
		if definition.ID().SyntheticRole().Valid() &&
			depth != providercontract.DepthExternalBoundary {
			return fmt.Errorf("synthetic definition %s must be external-boundary", definition.ID())
		}
	}
	return nil
}
