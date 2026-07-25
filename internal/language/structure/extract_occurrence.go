package structure

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func (b *fileBuilder) recordOccurrence(
	occurrence Occurrence,
	node ast.Node,
) error {
	b.work.IdentityProbes++
	key := fileOccurrenceKey{
		start: occurrence.id.Span().Start(),
		end:   occurrence.id.Span().End(),
		kind:  occurrence.id.KindID(),
	}
	if index := b.occurrenceIndex[key]; index.valid() {
		existing := b.occurrenceBuilder.occurrence(index)
		if existing != occurrence {
			return fmt.Errorf(
				"occurrence %s has conflicting canonical payloads",
				occurrence.id,
			)
		}
		if int(index) > len(b.occurrenceNodes) ||
			b.occurrenceNodes[index-1] != node {
			return fmt.Errorf(
				"occurrence %s has conflicting transient nodes",
				occurrence.id,
			)
		}
		return nil
	}
	index, err := b.occurrenceBuilder.Append(occurrence)
	if err != nil {
		return err
	}
	if int(index) != len(b.occurrenceNodes)+1 {
		return fmt.Errorf(
			"occurrence %s lost canonical node alignment",
			occurrence.id,
		)
	}
	b.occurrenceNodes = append(b.occurrenceNodes, node)
	b.occurrenceIndex[key] = index
	b.work.RecordAppends++
	return nil
}

func (b *fileBuilder) occurrence(
	id identity.OccurrenceID,
) (Occurrence, bool) {
	if b == nil || id.IsZero() ||
		id.Span().File() != b.file.ID() {
		return Occurrence{}, false
	}
	key := fileOccurrenceKey{
		start: id.Span().Start(),
		end:   id.Span().End(),
		kind:  id.KindID(),
	}
	index := b.occurrenceIndex[key]
	if !index.valid() {
		return Occurrence{}, false
	}
	occurrence := b.occurrenceBuilder.occurrence(index)
	return occurrence, occurrence.ID() == id
}

func estimatedSourceOccurrenceCapacity(byteCount int) int {
	const minimum = 64
	if byteCount <= minimum*32 {
		return minimum
	}
	return byteCount / 32
}

func (b *fileBuilder) makeOccurrence(
	node ast.Node,
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
) (Occurrence, error) {
	kind, err := Classify(node)
	if err != nil {
		return Occurrence{}, err
	}
	if kind.Disposition() != catalog.DispositionActive {
		return Occurrence{}, b.defect(
			node,
			edge,
			"construct disposition is "+kind.Disposition().String(),
		)
	}
	span := b.physicalSpan(node)
	spanID, err := identity.NewSpanID(
		b.file.ID(), span.Start.Offset, span.End.Offset,
	)
	if err != nil {
		return Occurrence{}, err
	}
	id, err := identity.NewOccurrenceID(spanID, uint16(kind))
	if err != nil {
		return Occurrence{}, err
	}
	lexical, err := tokenEvidence(node)
	if err != nil {
		return Occurrence{}, b.defect(node, edge, err.Error())
	}
	occurrence, err := NewOccurrence(
		id,
		kind,
		parent,
		edge,
		ordinal,
		span,
		b.displaySpan(node),
		lexical,
	)
	if err != nil {
		return Occurrence{}, err
	}
	return occurrence, nil
}

func (b *fileBuilder) physicalSpan(node ast.Node) Span {
	start := b.fset.PositionFor(node.Pos(), false)
	end := b.fset.PositionFor(node.End(), false)
	return Span{
		Start: Position{
			Line: start.Line, Column: start.Column, Offset: start.Offset,
		},
		End: Position{
			Line: end.Line, Column: end.Column, Offset: end.Offset,
		},
	}
}

func (b *fileBuilder) displaySpan(node ast.Node) DisplaySpan {
	start := b.fset.Position(node.Pos())
	end := b.fset.Position(node.End())
	physicalStart := b.fset.PositionFor(node.Pos(), false)
	physicalEnd := b.fset.PositionFor(node.End(), false)
	return DisplaySpan{
		Start: displayPosition(start, physicalStart, b.displayFile),
		End:   displayPosition(end, physicalEnd, b.displayFile),
	}
}

func displayPosition(
	adjusted token.Position,
	physical token.Position,
	physicalFile string,
) DisplayPosition {
	filename := adjusted.Filename
	if filename == physical.Filename {
		filename = physicalFile
	}
	return DisplayPosition{
		Filename: filename,
		Line:     adjusted.Line,
		Column:   adjusted.Column,
	}
}
