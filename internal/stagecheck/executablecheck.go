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
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func verifyExecutableRegions(
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	selections *scope.DefinitionSelections,
	inventory *executable.Inventory,
) error {
	actual, err := executableLedger(graph, inventory)
	if err != nil {
		return err
	}
	expected, err := deriveExecutableLedger(
		universe, plan, graph, selections, inventory,
	)
	if err != nil {
		return err
	}
	return compareLedgers("executable-region", actual, expected)
}

func executableLedger(
	graph *structure.Graph,
	inventory *executable.Inventory,
) (*structuralLedger, error) {
	actual := newStructuralLedger()
	additional := map[identity.OccurrenceID]structure.Occurrence{}
	for _, occurrence := range inventory.AdditionalOccurrences() {
		if _, structural := graph.ResidentOccurrence(
			occurrence.ID(),
		); structural {
			return nil, &VerificationError{
				Stage: "executable-region",
				Reason: "additional occurrence duplicates structural payload " +
					occurrence.ID().String(),
			}
		}
		if _, duplicate := additional[occurrence.ID()]; duplicate {
			return nil, &VerificationError{
				Stage: "executable-region",
				Reason: "additional occurrence is duplicated " +
					occurrence.ID().String(),
			}
		}
		additional[occurrence.ID()] = occurrence
		addRecord(
			&actual.additionalOccurrences,
			occurrenceLedgerRecordFromOccurrence(occurrence),
		)
	}
	for _, region := range inventory.Regions() {
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
				if _, present = additional[member]; !present {
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
			return nil, err
		}
		for _, reference := range region.References() {
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
		}
		for _, operation := range region.ImplicitOperations() {
			addRecord(
				&actual.implicitOperations,
				implicitOperationLedgerRecord{
					region: region.ID(),
					kind:   operation.Kind(),
					pkg:    operation.Package(),
				},
			)
		}
	}
	return actual, nil
}

func deriveExecutableLedger(
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	selections *scope.DefinitionSelections,
	inventory *executable.Inventory,
) (*structuralLedger, error) {
	loadedFiles := map[identity.FileID]*source.LoadedFile{}
	for _, pkg := range universe.Packages() {
		for _, file := range pkg.Files() {
			loadedFiles[file.ID()] = file
		}
	}
	derivedFiles := map[identity.FileID]*derivedFile{}
	definitionNodes := map[identity.DefinitionID]ast.Node{}
	definitionAt := map[identity.OccurrenceID]identity.DefinitionID{}
	for _, definition := range graph.ResidentDefinitions() {
		if !definition.ID().Root().IsZero() {
			definitionAt[definition.ID().Root()] = definition.ID()
		}
	}
	expected := newStructuralLedger()
	for _, selection := range selections.Records() {
		_, hasRegion := inventory.For(selection.Definition())
		full := selection.Depth() == contract.DepthFullSemantic
		if full != hasRegion {
			return nil, &VerificationError{
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
		derived, node, err := independentlyLoadDefinition(
			selection.Definition(),
			plan,
			loadedFiles,
			derivedFiles,
			definitionNodes,
		)
		if err != nil {
			return nil, err
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

func independentlyLoadDefinition(
	definition identity.DefinitionID,
	plan *sourceplan.Plan,
	loadedFiles map[identity.FileID]*source.LoadedFile,
	derivedFiles map[identity.FileID]*derivedFile,
	definitionNodes map[identity.DefinitionID]ast.Node,
) (*derivedFile, ast.Node, error) {
	fileID := definition.File()
	if plan != nil {
		decision, present := plan.For(fileID)
		if !present || decision.Kind() != sourceplan.KindLocalSyntax {
			return nil, nil, &VerificationError{
				Stage: "executable-region",
				Reason: "full definition lacks local structural source " +
					definition.String(),
			}
		}
	}
	derived := derivedFiles[fileID]
	if derived == nil {
		file := loadedFiles[fileID]
		if file == nil {
			return nil, nil, &VerificationError{
				Stage:  "executable-region",
				Reason: "full definition file is absent " + fileID.String(),
			}
		}
		var err error
		derived, err = deriveFile(file)
		if err != nil {
			return nil, nil, err
		}
		derivedFiles[fileID] = derived
		for id, node := range derived.definitions {
			definitionNodes[id] = node
		}
	}
	node := definitionNodes[definition]
	if node == nil {
		return nil, nil, &VerificationError{
			Stage: "executable-region",
			Reason: "independent derivation lacks full definition " +
				definition.String(),
		}
	}
	return derived, node, nil
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
