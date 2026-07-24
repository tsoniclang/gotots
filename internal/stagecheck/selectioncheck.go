package stagecheck

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func verifySelections(
	universe *source.Universe,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	selected contract.Contract,
	selectedPackages map[identity.PackageID]bool,
) error {
	dispositions := map[identity.PackageID]source.LanguageDisposition{}
	definitions := map[identity.DefinitionID]bool{}
	packages := map[identity.DefinitionID]identity.PackageID{}
	for _, pkg := range universe.Packages() {
		dispositions[pkg.ID()] = pkg.Disposition()
	}
	for _, indexed := range graph.DefinitionCensus() {
		definition := indexed.ID()
		if definitions[definition] {
			return &VerificationError{
				Stage:  "definition-selection",
				Reason: "duplicate definition " + definition.String(),
			}
		}
		definitions[definition] = true
		packages[definition] = indexed.Package()
	}
	exactProblems := newProblemSet()
	for _, rule := range selected.Rules() {
		if rule.Selector() != contract.SelectorExactDefinition {
			continue
		}
		target := rule.Definition()
		if selectedPackages != nil &&
			!selectedPackages[target.Package()] {
			continue
		}
		if !definitions[target] {
			exactProblems.add("missing exact-rule target " + target.String())
		}
	}
	if err := exactProblems.verificationError(
		"definition-selection",
		"exact-rule target join failed",
	); err != nil {
		return err
	}
	actualFacts := map[selectionfacts.ID]selectionfacts.Fact{}
	if err := facts.VisitFacts(func(fact selectionfacts.Fact) error {
		if fact.ID().IsZero() ||
			fact.ProducerDigest() == "" ||
			fact.EvidenceDigest() == "" {
			return &VerificationError{
				Stage:  "selection-fact",
				Reason: "selection fact has invalid identity or evidence",
			}
		}
		if _, duplicate := actualFacts[fact.ID()]; duplicate {
			return &VerificationError{
				Stage:  "selection-fact",
				Reason: "duplicate selection fact " + fact.ID().String(),
			}
		}
		actualFacts[fact.ID()] = fact
		return nil
	}); err != nil {
		return err
	}
	expectedFacts := map[selectionfacts.ID]bool{}
	for definition, pkg := range packages {
		for _, kind := range selected.RequestedFacts(definition, pkg) {
			id, err := selectionfacts.NewID(definition, kind)
			if err != nil {
				return err
			}
			expectedFacts[id] = true
		}
	}
	factProblems := newProblemSet()
	for id := range expectedFacts {
		if _, present := actualFacts[id]; !present {
			factProblems.add("missing " + id.String())
		}
	}
	for id := range actualFacts {
		if !expectedFacts[id] {
			factProblems.add("unexpected " + id.String())
		}
	}
	if err := factProblems.verificationError(
		"selection-fact", "fact-set join failed",
	); err != nil {
		return err
	}
	records := map[identity.DefinitionID]scope.DefinitionSelection{}
	for _, record := range selections.Records() {
		if _, duplicate := records[record.Definition()]; duplicate {
			return &VerificationError{
				Stage:  "definition-selection",
				Reason: "duplicate selection " + record.Definition().String(),
			}
		}
		records[record.Definition()] = record
	}
	problems := newProblemSet()
	for definitionID := range definitions {
		record, present := records[definitionID]
		if !present {
			problems.add("missing " + definitionID.String())
			continue
		}
		pkg := packages[definitionID]
		values := map[contract.SelectionFactKind]bool{}
		var factIDs []selectionfacts.ID
		for _, kind := range selected.RequestedFacts(definitionID, pkg) {
			id, _ := selectionfacts.NewID(definitionID, kind)
			values[kind] = actualFacts[id].Value()
			factIDs = append(factIDs, id)
		}
		provider, witness, err := selected.Bind(contract.Query{
			Definition: definitionID,
			Package:    pkg,
			Intrinsic: dispositions[pkg] ==
				source.DispositionUnsafeIntrinsic,
			Facts: values,
		})
		if err != nil {
			problems.add(err.Error())
			continue
		}
		if record.Provider() != provider ||
			record.Depth() != provider.Depth() ||
			record.ContractID() != selected.ID() ||
			record.ContractFingerprint() != selected.Fingerprint() ||
			record.Witness().RuleID != witness.RuleID ||
			record.Witness().Selector != witness.Selector ||
			record.Witness().Condition != witness.Condition ||
			!sameFactKinds(record.Witness().Facts, witness.Facts) ||
			!sameFactIDs(record.Facts(), factIDs) {
			problems.add(
				"selection payload mismatch " + definitionID.String(),
			)
		}
		if definitionID.Kind() == identity.DefinitionBodylessDecl &&
			record.Depth() == contract.DepthFullSemantic {
			problems.add(
				"bodyless definition selected full " + definitionID.String(),
			)
		}
		if definitionID.SyntheticRole().Valid() &&
			record.Depth() != contract.DepthExternalBoundary {
			problems.add(
				"synthetic definition has non-external depth " +
					definitionID.String(),
			)
		}
	}
	for definition := range records {
		if _, present := definitions[definition]; !present {
			problems.add(
				"selection without definition " + definition.String(),
			)
		}
	}
	return problems.verificationError(
		"definition-selection",
		"selection exact join failed",
	)
}

func sameFactKinds(
	left, right []contract.SelectionFactKind,
) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sameFactIDs(left, right []selectionfacts.ID) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
