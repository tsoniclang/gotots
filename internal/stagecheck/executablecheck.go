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
	capacity, err := executableLedgerCapacity(
		sourcePackage, selections, inventory,
	)
	if err != nil {
		return executableVerificationSummary{}, err
	}
	arena := newExecutableLedgerArena()
	actual, summary, err := executableLedgerForPackage(
		sourcePackage, graph, selections, inventory, arena,
		capacity,
	)
	if err != nil {
		return executableVerificationSummary{}, err
	}
	if err := deriveExecutableLedgerForPackage(
		packageID, evidence, graph, selections, inventory, actual,
	); err != nil {
		return executableVerificationSummary{}, err
	}
	if err := verifyCompactExecutableLedgerDifference(
		"executable-region/"+packageID.String(),
		actual,
	); err != nil {
		return executableVerificationSummary{}, err
	}
	return summary, nil
}

func executableLedgerCapacity(
	sourcePackage *source.LoadedPackage,
	selections []scope.DefinitionSelection,
	inventory *executable.Inventory,
) (compactExecutableCapacity, error) {
	var capacity compactExecutableCapacity
	files := sourcePackage.Files()
	fileIDs := make([]identity.FileID, len(files))
	for index, file := range files {
		fileIDs[index] = file.ID()
	}
	var err error
	capacity.additionalOccurrences, err =
		inventory.AdditionalOccurrenceCountForFiles(fileIDs)
	if err != nil {
		return compactExecutableCapacity{}, err
	}
	for _, selection := range selections {
		if selection.Depth() != contract.DepthFullSemantic {
			continue
		}
		region, present := inventory.For(selection.Definition())
		if !present {
			return compactExecutableCapacity{}, &VerificationError{
				Stage: "executable-region",
				Reason: "full definition has no region " +
					selection.Definition().String(),
			}
		}
		capacity.regions++
		capacity.members += region.MemberCount()
		capacity.definitionReferences += region.ReferenceCount()
		capacity.implicitOperations +=
			region.ImplicitOperationCount()
	}
	return capacity, nil
}

func canonicalExecutableOccurrence(
	graph *structure.Graph,
	inventory *executable.Inventory,
	id identity.OccurrenceID,
) (structure.OccurrenceRef, bool, bool, error) {
	if reference, present := graph.ResidentOccurrenceRef(id); present {
		return reference, false, true, nil
	}
	reference, present, err := inventory.AdditionalOccurrenceRef(id)
	if err != nil {
		return structure.OccurrenceRef{}, false, false, err
	}
	return reference, present, present, nil
}

func executableLedgerForPackage(
	sourcePackage *source.LoadedPackage,
	graph *structure.Graph,
	selections []scope.DefinitionSelection,
	inventory *executable.Inventory,
	arena *executableLedgerArena,
	capacity compactExecutableCapacity,
) (*compactExecutableLedger, executableVerificationSummary, error) {
	actual := newSizedCompactExecutableLedger(arena, capacity)
	additional := map[structure.OccurrenceRef]bool{}
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
				reference := occurrence
				if additional[reference] {
					return &VerificationError{
						Stage: "executable-region",
						Reason: "additional occurrence is duplicated " +
							occurrence.ID().String(),
					}
				}
				additional[reference] = true
				adjustExecutableRecordDifference(
					&actual.additionalOccurrences,
					reference,
					1,
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
		regionReference := arena.definition(region.Definition())
		adjustExecutableRecordDifference(
			&actual.regions,
			regionReference,
			1,
		)
		memberSet := map[structure.OccurrenceRef]bool{}
		if err := region.VisitMembers(func(
			index int,
			member identity.OccurrenceID,
		) error {
			memberReference, _, present, lookupErr :=
				canonicalExecutableOccurrence(
					graph, inventory, member,
				)
			if lookupErr != nil {
				return lookupErr
			}
			if !present {
				return &VerificationError{
					Stage: "executable-region",
					Reason: fmt.Sprintf(
						"region %s member %s has no canonical payload",
						region.ID(), member,
					),
				}
			}
			if memberSet[memberReference] {
				return &VerificationError{
					Stage: "executable-region",
					Reason: fmt.Sprintf(
						"region %s repeats member %s", region.ID(), member,
					),
				}
			}
			memberSet[memberReference] = true
			adjustExecutableRecordDifference(
				&actual.members,
				compactExecutableMember{
					region:     regionReference,
					ordinal:    index,
					occurrence: memberReference,
				},
				1,
			)
			return nil
		}); err != nil {
			return nil, executableVerificationSummary{}, err
		}
		if err := region.VisitReferences(func(
			reference executable.DefinitionReference,
		) error {
			adjustExecutableRecordDifference(
				&actual.definitionReferences,
				compactDefinitionReference{
					region:  regionReference,
					parent:  reference.Parent(),
					edge:    uint16(reference.Edge()),
					ordinal: reference.Ordinal(),
					child:   arena.definition(reference.Child()),
				},
				1,
			)
			return nil
		}); err != nil {
			return nil, executableVerificationSummary{}, err
		}
		if err := region.VisitImplicitOperations(func(
			operation executable.ImplicitOperation,
		) error {
			adjustExecutableRecordDifference(
				&actual.implicitOperations,
				compactImplicitOperation{
					region: regionReference,
					kind:   operation.Kind(),
					pkg:    operation.Package(),
				},
				1,
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
	inventory *executable.Inventory,
	expected *compactExecutableLedger,
) error {
	definitionAt := map[identity.OccurrenceID]identity.DefinitionID{}
	for _, file := range evidence.files {
		for definition := range file.definitions {
			if !definition.Root().IsZero() {
				definitionAt[definition.Root()] = definition
			}
		}
	}
	for _, selection := range selections {
		if selection.Depth() != contract.DepthFullSemantic {
			continue
		}
		regionReference := expected.arena.definition(selection.Definition())
		adjustExecutableRecordDifference(
			&expected.regions,
			regionReference,
			-1,
		)
		if selection.Definition().ImplicitOp().Valid() {
			if selection.Definition().ImplicitOp() !=
				identity.ImplicitDefinitionPackageInit {
				return &VerificationError{
					Stage: "executable-region",
					Reason: "unknown full implicit definition " +
						selection.Definition().String(),
				}
			}
			if selection.Definition().Package() != packageID {
				return &VerificationError{
					Stage: "executable-region",
					Reason: "implicit definition has the wrong package " +
						selection.Definition().String(),
				}
			}
			adjustExecutableRecordDifference(
				&expected.implicitOperations,
				compactImplicitOperation{
					region: regionReference,
					kind: executable.
						ImplicitOperationCoordinatePackageInitialization,
					pkg: selection.Definition().Package(),
				},
				-1,
			)
			continue
		}
		derived := evidence.files[selection.Definition().File()]
		if derived == nil {
			return &VerificationError{
				Stage: "executable-region",
				Reason: "full definition lacks package-local verifier evidence " +
					selection.Definition().String(),
			}
		}
		node := derived.definitions[selection.Definition()]
		if node == nil {
			return &VerificationError{
				Stage: "executable-region",
				Reason: "independent derivation lacks full definition " +
					selection.Definition().String(),
			}
		}
		builder := independentExecutableBuilder{
			file:         derived,
			region:       regionReference,
			current:      selection.Definition(),
			definitionAt: definitionAt,
			graph:        graph,
			inventory:    inventory,
			ledger:       expected,
		}
		entries, err := independentDefinitionEntries(node)
		if err != nil {
			return err
		}
		root, present := graph.ResidentOccurrenceRef(
			selection.Definition().Root(),
		)
		if !present {
			return fmt.Errorf(
				"definition root %s is absent", selection.Definition().Root(),
			)
		}
		for _, entry := range entries {
			nested, isDefinition, err := builder.nested(entry.node)
			if err != nil {
				return err
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
				return err
			}
		}
	}
	return nil
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
	region       executableLedgerDefinitionRef
	current      identity.DefinitionID
	definitionAt map[identity.OccurrenceID]identity.DefinitionID
	graph        *structure.Graph
	inventory    *executable.Inventory
	ledger       *compactExecutableLedger
	memberIndex  int
	additional   map[structure.OccurrenceRef]bool
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
	reference, additional, present, err :=
		canonicalExecutableOccurrence(
			b.graph, b.inventory, occurrence.id,
		)
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf(
			"independent executable occurrence %s has no canonical payload",
			occurrence.id,
		)
	}
	if occurrenceLedgerRecordFromOccurrence(reference.Occurrence()) !=
		occurrenceLedgerRecordFromDerived(occurrence) {
		return fmt.Errorf(
			"independent executable occurrence %s conflicts with canonical payload",
			occurrence.id,
		)
	}
	if additional {
		if b.additional == nil {
			b.additional = map[structure.OccurrenceRef]bool{}
		}
		if !b.additional[reference] {
			b.additional[reference] = true
			adjustExecutableRecordDifference(
				&b.ledger.additionalOccurrences,
				reference,
				-1,
			)
		}
	}
	adjustExecutableRecordDifference(
		&b.ledger.members,
		compactExecutableMember{
			region:     b.region,
			ordinal:    b.memberIndex,
			occurrence: reference,
		},
		-1,
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
	adjustExecutableRecordDifference(
		&b.ledger.definitionReferences,
		compactDefinitionReference{
			region:  b.region,
			parent:  parent,
			edge:    uint16(edge),
			ordinal: ordinal,
			child:   b.ledger.arena.definition(child),
		},
		-1,
	)
}
