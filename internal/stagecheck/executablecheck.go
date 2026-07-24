package stagecheck

import (
	"fmt"
	"go/ast"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

type executableVerificationIndex struct {
	byPackage       map[identity.PackageID][]scope.DefinitionSelection
	fullDefinitions int
}

func indexExecutableVerification(
	graph *structure.Graph,
	selections *scope.DefinitionSelections,
	selectedPackages map[identity.PackageID]bool,
) (*executableVerificationIndex, error) {
	definitionPackages := map[identity.DefinitionID]identity.PackageID{}
	for _, record := range graph.DefinitionCensus() {
		definitionPackages[record.ID()] = record.Package()
	}
	index := &executableVerificationIndex{
		byPackage: map[identity.PackageID][]scope.DefinitionSelection{},
	}
	seen := map[identity.DefinitionID]bool{}
	for _, selection := range selections.Records() {
		packageID, present := definitionPackages[selection.Definition()]
		if !present {
			return nil, &VerificationError{
				Stage: "executable-region",
				Reason: "selection names an absent definition " +
					selection.Definition().String(),
			}
		}
		if selectedPackages != nil && !selectedPackages[packageID] {
			continue
		}
		if seen[selection.Definition()] {
			return nil, &VerificationError{
				Stage: "executable-region",
				Reason: "selection repeats definition " +
					selection.Definition().String(),
			}
		}
		seen[selection.Definition()] = true
		index.byPackage[packageID] = append(
			index.byPackage[packageID], selection,
		)
		if selection.Depth() == contract.DepthFullSemantic {
			index.fullDefinitions++
		}
	}
	return index, nil
}

type executableVerificationSummary struct {
	regions               int
	additionalOccurrences int
}

func (summary *executableVerificationSummary) add(
	other executableVerificationSummary,
) {
	summary.regions += other.regions
	summary.additionalOccurrences += other.additionalOccurrences
}

func verifyExecutablePackage(
	packageID identity.PackageID,
	sourcePackage *source.LoadedPackage,
	evidence *independentPackageEvidence,
	graph *structure.Graph,
	selections []scope.DefinitionSelection,
	inventory *executable.Inventory,
) (executableVerificationSummary, error) {
	actual, summary, err := executableLedgerForPackage(
		sourcePackage, graph, selections, inventory,
	)
	if err != nil {
		return executableVerificationSummary{}, err
	}
	expected, err := deriveExecutableLedgerForPackage(
		packageID, evidence, graph, selections,
	)
	if err != nil {
		return executableVerificationSummary{}, err
	}
	if err := compareLedgers(
		"executable-region/"+packageID.String(),
		actual,
		expected,
	); err != nil {
		return executableVerificationSummary{}, err
	}
	return summary, nil
}

func executableLedgerForPackage(
	sourcePackage *source.LoadedPackage,
	graph *structure.Graph,
	selections []scope.DefinitionSelection,
	inventory *executable.Inventory,
) (*structuralLedger, executableVerificationSummary, error) {
	actual := newStructuralLedger()
	additional := map[identity.OccurrenceID]bool{}
	var summary executableVerificationSummary
	for _, file := range sourcePackage.Files() {
		err := inventory.VisitAdditionalOccurrenceRefsForFile(
			file.ID(),
			func(occurrence structure.OccurrenceRef) error {
				if _, structural := graph.ResidentOccurrence(
					occurrence.ID(),
				); structural {
					return &VerificationError{
						Stage: "executable-region",
						Reason: "additional occurrence duplicates structural payload " +
							occurrence.ID().String(),
					}
				}
				if additional[occurrence.ID()] {
					return &VerificationError{
						Stage: "executable-region",
						Reason: "additional occurrence is duplicated " +
							occurrence.ID().String(),
					}
				}
				additional[occurrence.ID()] = true
				addRecord(
					&actual.additionalOccurrences,
					occurrenceLedgerRecordFromRef(occurrence),
				)
				summary.additionalOccurrences++
				return nil
			},
		)
		if err != nil {
			return nil, executableVerificationSummary{}, err
		}
	}
	for _, selection := range selections {
		region, hasRegion := inventory.For(selection.Definition())
		full := selection.Depth() == contract.DepthFullSemantic
		if full != hasRegion {
			return nil, executableVerificationSummary{}, &VerificationError{
				Stage: "executable-region",
				Reason: fmt.Sprintf(
					"%s full=%t region=%t",
					selection.Definition(), full, hasRegion,
				),
			}
		}
		if !full {
			continue
		}
		summary.regions++
		addRecord(
			&actual.executableRegions,
			executableRegionLedgerRecord{id: region.ID()},
		)
		memberSet := map[identity.OccurrenceID]bool{}
		if err := region.VisitMembers(func(
			index int,
			member identity.OccurrenceID,
		) error {
			if memberSet[member] {
				return &VerificationError{
					Stage: "executable-region",
					Reason: fmt.Sprintf(
						"region %s repeats member %s", region.ID(), member,
					),
				}
			}
			memberSet[member] = true
			if _, present := graph.ResidentOccurrence(member); !present {
				if present = additional[member]; !present {
					return &VerificationError{
						Stage: "executable-region",
						Reason: fmt.Sprintf(
							"region %s member %s has no canonical payload",
							region.ID(), member,
						),
					}
				}
			}
			addRecord(&actual.executableMembers, executableMemberLedgerRecord{
				region: region.ID(), ordinal: index, occurrence: member,
			})
			return nil
		}); err != nil {
			return nil, executableVerificationSummary{}, err
		}
		if err := region.VisitReferences(func(
			reference executable.DefinitionReference,
		) error {
			addRecord(
				&actual.definitionReferences,
				definitionReferenceLedgerRecord{
					region:  region.ID(),
					parent:  reference.Parent(),
					edge:    reference.Edge(),
					ordinal: reference.Ordinal(),
					child:   reference.Child(),
				},
			)
			return nil
		}); err != nil {
			return nil, executableVerificationSummary{}, err
		}
		if err := region.VisitImplicitOperations(func(
			operation executable.ImplicitOperation,
		) error {
			addRecord(
				&actual.implicitOperations,
				implicitOperationLedgerRecord{
					region: region.ID(),
					kind:   operation.Kind(),
					pkg:    operation.Package(),
				},
			)
			return nil
		}); err != nil {
			return nil, executableVerificationSummary{}, err
		}
	}
	return actual, summary, nil
}

func deriveExecutableLedgerForPackage(
	packageID identity.PackageID,
	evidence *independentPackageEvidence,
	graph *structure.Graph,
	selections []scope.DefinitionSelection,
) (*structuralLedger, error) {
	definitionAt := map[identity.OccurrenceID]identity.DefinitionID{}
	for _, file := range evidence.files {
		for definition := range file.definitions {
			if !definition.Root().IsZero() {
				definitionAt[definition.Root()] = definition
			}
		}
	}
	expected := newStructuralLedger()
	for _, selection := range selections {
		if selection.Depth() != contract.DepthFullSemantic {
			continue
		}
		regionID, _ := identity.NewExecutableRegionID(
			selection.Definition(),
		)
		addRecord(
			&expected.executableRegions,
			executableRegionLedgerRecord{id: regionID},
		)
		if selection.Definition().ImplicitOp().Valid() {
			if selection.Definition().ImplicitOp() !=
				identity.ImplicitDefinitionPackageInit {
				return nil, &VerificationError{
					Stage: "executable-region",
					Reason: "unknown full implicit definition " +
						selection.Definition().String(),
				}
			}
			if selection.Definition().Package() != packageID {
				return nil, &VerificationError{
					Stage: "executable-region",
					Reason: "implicit definition has the wrong package " +
						selection.Definition().String(),
				}
			}
			addRecord(
				&expected.implicitOperations,
				implicitOperationLedgerRecord{
					region: regionID,
					kind: executable.
						ImplicitOperationCoordinatePackageInitialization,
					pkg: selection.Definition().Package(),
				},
			)
			continue
		}
		derived := evidence.files[selection.Definition().File()]
		if derived == nil {
			return nil, &VerificationError{
				Stage: "executable-region",
				Reason: "full definition lacks package-local verifier evidence " +
					selection.Definition().String(),
			}
		}
		node := derived.definitions[selection.Definition()]
		if node == nil {
			return nil, &VerificationError{
				Stage: "executable-region",
				Reason: "independent derivation lacks full definition " +
					selection.Definition().String(),
			}
		}
		builder := independentExecutableBuilder{
			file:         derived,
			region:       regionID,
			current:      selection.Definition(),
			definitionAt: definitionAt,
			graph:        graph,
			ledger:       expected,
		}
		entries, err := independentDefinitionEntries(node)
		if err != nil {
			return nil, err
		}
		root, present := graph.ResidentOccurrence(
			selection.Definition().Root(),
		)
		if !present {
			return nil, fmt.Errorf(
				"definition root %s is absent", selection.Definition().Root(),
			)
		}
		for _, entry := range entries {
			nested, isDefinition, err := builder.nested(entry.node)
			if err != nil {
				return nil, err
			}
			if isDefinition {
				builder.reference(
					root.ID(), entry.edge, entry.ordinal, nested,
				)
				continue
			}
			if err := builder.visit(
				entry.node, root.ID(), entry.edge, entry.ordinal,
			); err != nil {
				return nil, err
			}
		}
	}
	return expected, nil
}

func verifyExecutableSummary(
	inventory *executable.Inventory,
	fullDefinitions int,
	actual executableVerificationSummary,
) error {
	if actual.regions != fullDefinitions ||
		inventory.RegionCount() != fullDefinitions {
		return &VerificationError{
			Stage: "executable-region",
			Reason: fmt.Sprintf(
				"verified/full/inventory region cardinality is %d/%d/%d",
				actual.regions,
				fullDefinitions,
				inventory.RegionCount(),
			),
		}
	}
	if actual.additionalOccurrences !=
		inventory.AdditionalOccurrenceCount() {
		return &VerificationError{
			Stage: "executable-region",
			Reason: fmt.Sprintf(
				"verified/inventory additional occurrence cardinality is %d/%d",
				actual.additionalOccurrences,
				inventory.AdditionalOccurrenceCount(),
			),
		}
	}
	return nil
}

type independentExecutableBuilder struct {
	file         *derivedFile
	region       identity.ExecutableRegionID
	current      identity.DefinitionID
	definitionAt map[identity.OccurrenceID]identity.DefinitionID
	graph        *structure.Graph
	ledger       *structuralLedger
	memberIndex  int
	additional   map[identity.OccurrenceID]bool
}

func (b *independentExecutableBuilder) visit(
	node ast.Node,
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
) error {
	occurrence, err := b.file.occurrence(
		node, parent, edge, ordinal,
	)
	if err != nil {
		return err
	}
	if structural, present := b.graph.ResidentOccurrence(
		occurrence.id,
	); present {
		if occurrenceLedgerRecordFromOccurrence(structural) !=
			occurrenceLedgerRecordFromDerived(occurrence) {
			return fmt.Errorf(
				"independent executable occurrence %s conflicts with structure",
				occurrence.id,
			)
		}
	} else {
		if b.additional == nil {
			b.additional = map[identity.OccurrenceID]bool{}
		}
		if !b.additional[occurrence.id] {
			b.additional[occurrence.id] = true
			addRecord(
				&b.ledger.additionalOccurrences,
				occurrenceLedgerRecordFromDerived(occurrence),
			)
		}
	}
	addRecord(
		&b.ledger.executableMembers,
		executableMemberLedgerRecord{
			region:     b.region,
			ordinal:    b.memberIndex,
			occurrence: occurrence.id,
		},
	)
	b.memberIndex++
	children, err := independentChildren(node, occurrence.kind)
	if err != nil {
		return err
	}
	for _, child := range children {
		nested, isDefinition, err := b.nested(child.node)
		if err != nil {
			return err
		}
		if isDefinition {
			b.reference(
				occurrence.id, child.edge, child.ordinal, nested,
			)
			continue
		}
		if err := b.visit(
			child.node, occurrence.id, child.edge, child.ordinal,
		); err != nil {
			return err
		}
	}
	return nil
}

func (b *independentExecutableBuilder) nested(
	node ast.Node,
) (identity.DefinitionID, bool, error) {
	kind, err := independentKind(node)
	if err != nil {
		return identity.DefinitionID{}, false, err
	}
	span := b.file.span(node)
	spanID, err := identity.NewSpanID(
		b.file.file.ID(), span.Start.Offset, span.End.Offset,
	)
	if err != nil {
		return identity.DefinitionID{}, false, err
	}
	occurrence, err := identity.NewOccurrenceID(spanID, uint16(kind))
	if err != nil {
		return identity.DefinitionID{}, false, err
	}
	definition, present := b.definitionAt[occurrence]
	return definition, present && definition != b.current, nil
}

func (b *independentExecutableBuilder) reference(
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
	child identity.DefinitionID,
) {
	addRecord(
		&b.ledger.definitionReferences,
		definitionReferenceLedgerRecord{
			region:  b.region,
			parent:  parent,
			edge:    edge,
			ordinal: ordinal,
			child:   child,
		},
	)
}
