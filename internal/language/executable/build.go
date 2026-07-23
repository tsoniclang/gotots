package executable

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
)

// Build traverses exactly the full-semantic definitions selected by scope.
func Build(
	graph *structure.Graph,
	index *structure.TransientIndex,
	selections *scope.DefinitionSelections,
) (*Inventory, error) {
	out := &Inventory{
		byID:         map[identity.DefinitionID]Region{},
		byOccurrence: map[identity.OccurrenceID]structure.Occurrence{},
	}
	definitions := map[identity.DefinitionID]structure.ImplementationDefinition{}
	definitionAt := map[identity.OccurrenceID]identity.DefinitionID{}
	for _, definition := range graph.ResidentDefinitions() {
		definitions[definition.ID()] = definition
		if !definition.ID().Root().IsZero() {
			definitionAt[definition.ID().Root()] = definition.ID()
		}
	}
	for _, selection := range selections.Records() {
		if selection.Depth() != contract.DepthFullSemantic {
			continue
		}
		definition, present := definitions[selection.Definition()]
		if !present {
			return nil, fmt.Errorf(
				"full selection names unknown definition %s",
				selection.Definition(),
			)
		}
		region, err := buildRegion(
			graph, index, definitionAt, definition, &out.work, out,
		)
		if err != nil {
			return nil, err
		}
		if _, duplicate := out.byID[definition.ID()]; duplicate {
			return nil, fmt.Errorf(
				"duplicate executable region %s", definition.ID(),
			)
		}
		out.byID[definition.ID()] = region
		out.regionIDs = append(out.regionIDs, definition.ID())
	}
	out.sort()
	if err := Validate(graph, selections, out); err != nil {
		return nil, err
	}
	return out, nil
}

func buildRegion(
	graph *structure.Graph,
	index *structure.TransientIndex,
	definitionAt map[identity.OccurrenceID]identity.DefinitionID,
	definition structure.ImplementationDefinition,
	work *Work,
	inventory *Inventory,
) (Region, error) {
	regionID, err := identity.NewExecutableRegionID(definition.ID())
	if err != nil {
		return Region{}, err
	}
	region := Region{id: regionID}
	if definition.Kind() == identity.DefinitionImplicit {
		if definition.ID().ImplicitOp() != identity.ImplicitDefinitionPackageInit {
			return Region{}, fmt.Errorf(
				"full implicit definition %s has no typed executable owner",
				definition.ID(),
			)
		}
		region.implicit = []ImplicitOperation{{
			kind: ImplicitOperationCoordinatePackageInitialization,
			pkg:  definition.ID().Package(),
		}}
		work.RecordAppends++
		return region, nil
	}
	if _, exists := index.DefinitionNode(definition.ID()); !exists {
		return Region{}, fmt.Errorf(
			"full definition %s has no transient syntax", definition.ID(),
		)
	}
	file, exists := index.File(definition.ID().File())
	if !exists {
		return Region{}, fmt.Errorf(
			"full definition %s has no transient source file", definition.ID(),
		)
	}
	builder := regionBuilder{
		file:         definition.ID().File(),
		fset:         file.PhysicalFileSet(),
		definitionAt: definitionAt,
		current:      definition.ID(),
		graph:        graph,
		inventory:    inventory,
		work:         work,
		region:       &region,
	}
	boundary, present := graph.ResidentBoundary(definition.ID())
	if !present {
		return Region{}, fmt.Errorf(
			"full definition %s has no execution boundary", definition.ID(),
		)
	}
	entryNodes := index.ExecutionEntryNodes(definition.ID())
	entries := boundary.Entries()
	if len(entryNodes) != len(entries) {
		return Region{}, fmt.Errorf(
			"full definition %s has %d transient entries for %d boundaries",
			definition.ID(), len(entryNodes), len(entries),
		)
	}
	for entryIndex, entry := range entries {
		entryNode := entryNodes[entryIndex]
		entryOccurrence, present := graph.ResidentOccurrence(entry.ID())
		if !present {
			return Region{}, fmt.Errorf(
				"boundary entry %s has no structural occurrence", entry.ID(),
			)
		}
		nested, isDefinition, err := builder.nestedDefinition(entryNode)
		if err != nil {
			return Region{}, err
		}
		if isDefinition {
			region.references = append(
				region.references,
				DefinitionReference{
					parent:  entryOccurrence.Parent(),
					edge:    entryOccurrence.Edge(),
					ordinal: entryOccurrence.Ordinal(),
					child:   nested,
				},
			)
			work.RecordAppends++
			continue
		}
		if err := builder.visit(
			entryNode,
			entryOccurrence.Parent(),
			entryOccurrence.Edge(),
			entryOccurrence.Ordinal(),
		); err != nil {
			return Region{}, err
		}
	}
	return region, nil
}

type regionBuilder struct {
	file         identity.FileID
	fset         *token.FileSet
	definitionAt map[identity.OccurrenceID]identity.DefinitionID
	current      identity.DefinitionID
	graph        *structure.Graph
	inventory    *Inventory
	work         *Work
	region       *Region
}

func (b *regionBuilder) visit(
	node ast.Node,
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
) error {
	occurrence, err := b.occurrence(node, parent, edge, ordinal)
	if err != nil {
		return err
	}
	if err := b.recordOccurrence(occurrence); err != nil {
		return err
	}
	b.region.members = append(b.region.members, occurrence.ID())
	b.work.RecordAppends++
	children, err := structure.Children(node, occurrence.Kind())
	if err != nil {
		return err
	}
	for _, child := range children {
		b.work.CatalogEdges++
		nested, isDefinition, err := b.nestedDefinition(child.Node())
		if err != nil {
			return err
		}
		if isDefinition {
			b.region.references = append(
				b.region.references,
				DefinitionReference{
					parent:  occurrence.ID(),
					edge:    child.Edge(),
					ordinal: child.Ordinal(),
					child:   nested,
				},
			)
			b.work.RecordAppends++
			continue
		}
		if err := b.visit(
			child.Node(),
			occurrence.ID(),
			child.Edge(),
			child.Ordinal(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (b *regionBuilder) nestedDefinition(
	node ast.Node,
) (identity.DefinitionID, bool, error) {
	b.work.BoundaryProbes++
	kind, err := structure.Classify(node)
	if err != nil {
		return identity.DefinitionID{}, false, err
	}
	span := physicalSpan(b.fset, node)
	spanID, err := identity.NewSpanID(
		b.file, span.Start.Offset, span.End.Offset,
	)
	if err != nil {
		return identity.DefinitionID{}, false, err
	}
	occurrenceID, err := identity.NewOccurrenceID(spanID, uint16(kind))
	if err != nil {
		return identity.DefinitionID{}, false, err
	}
	b.work.IdentityProbes++
	definition, present := b.definitionAt[occurrenceID]
	return definition, present && definition != b.current, nil
}

func (b *regionBuilder) occurrence(
	node ast.Node,
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
) (structure.Occurrence, error) {
	kind, err := structure.Classify(node)
	if err != nil {
		return structure.Occurrence{}, err
	}
	if kind.Disposition() != catalog.DispositionActive {
		return structure.Occurrence{}, fmt.Errorf(
			"executable occurrence %s has disposition %s",
			kind, kind.Disposition(),
		)
	}
	span := physicalSpan(b.fset, node)
	spanID, err := identity.NewSpanID(
		b.file, span.Start.Offset, span.End.Offset,
	)
	if err != nil {
		return structure.Occurrence{}, err
	}
	id, err := identity.NewOccurrenceID(spanID, uint16(kind))
	if err != nil {
		return structure.Occurrence{}, err
	}
	lexical, err := structure.TokenEvidence(node)
	if err != nil {
		return structure.Occurrence{}, err
	}
	return structure.NewOccurrence(
		id, kind, parent, edge, ordinal, span,
		displaySpan(b.fset, b.file, node), lexical,
	)
}

func (b *regionBuilder) recordOccurrence(
	occurrence structure.Occurrence,
) error {
	b.work.JoinProbes++
	if structural, present := b.graph.ResidentOccurrence(
		occurrence.ID(),
	); present {
		if structural != occurrence {
			return fmt.Errorf(
				"executable occurrence %s conflicts with structural payload",
				occurrence.ID(),
			)
		}
		return nil
	}
	b.work.IdentityProbes++
	if existing, present := b.inventory.byOccurrence[occurrence.ID()]; present {
		if existing != occurrence {
			return fmt.Errorf(
				"executable occurrence %s has conflicting payloads",
				occurrence.ID(),
			)
		}
		return nil
	}
	b.inventory.byOccurrence[occurrence.ID()] = occurrence
	b.inventory.additionalIDs = append(
		b.inventory.additionalIDs, occurrence.ID(),
	)
	b.work.RecordAppends++
	return nil
}

func physicalSpan(
	fset *token.FileSet,
	node ast.Node,
) structure.Span {
	start := fset.PositionFor(node.Pos(), false)
	end := fset.PositionFor(node.End(), false)
	return structure.Span{
		Start: structure.Position{
			Line: start.Line, Column: start.Column, Offset: start.Offset,
		},
		End: structure.Position{
			Line: end.Line, Column: end.Column, Offset: end.Offset,
		},
	}
}

func displaySpan(
	fset *token.FileSet,
	file identity.FileID,
	node ast.Node,
) structure.DisplaySpan {
	start := fset.Position(node.Pos())
	end := fset.Position(node.End())
	physicalStart := fset.PositionFor(node.Pos(), false)
	physicalEnd := fset.PositionFor(node.End(), false)
	return structure.DisplaySpan{
		Start: executableDisplayPosition(start, physicalStart, file),
		End:   executableDisplayPosition(end, physicalEnd, file),
	}
}

func executableDisplayPosition(
	adjusted token.Position,
	physical token.Position,
	file identity.FileID,
) structure.DisplayPosition {
	filename := adjusted.Filename
	if filename == physical.Filename {
		filename = file.String()
	}
	return structure.DisplayPosition{
		Filename: filename,
		Line:     adjusted.Line,
		Column:   adjusted.Column,
	}
}
