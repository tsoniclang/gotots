package analyze

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// Position is one source point: 1-based line and column plus 0-based byte
// offset, all derived from the toolchain file set.
type Position struct {
	Line   int
	Column int
	Offset int
}

// Span is the half-open source range of one occurrence.
type Span struct {
	Filename string
	Start    Position
	End      Position
}

// OccurrenceID is the canonical identity of one construct occurrence, derived
// from its span and kind so it is independent of traversal order and stable
// across runs.
type OccurrenceID string

// Occurrence is one classified construct instance in a source file: its
// canonical identity, catalog kind, span, and the identity of its enclosing
// occurrence (empty for the root file). This per-occurrence record is the
// authoritative inventory; counts are a projection of it.
type Occurrence struct {
	ID     OccurrenceID
	Kind   catalog.Kind
	Parent OccurrenceID
	Span   Span
}

// Inventory is the authoritative construct inventory of one file: every
// occurrence in parent-directed pre-order.
type Inventory struct {
	Path        string
	Occurrences []Occurrence
}

// KindCount is one projected count for the report surface.
type KindCount struct {
	Kind  catalog.Kind
	Count int
}

// CountsByKind projects the authoritative occurrences into per-kind counts,
// sorted by kind name. Counts are derived here; they are never the source of
// truth.
func (inv Inventory) CountsByKind() []KindCount {
	counts := map[catalog.Kind]int{}
	for _, occurrence := range inv.Occurrences {
		counts[occurrence.Kind]++
	}
	projected := make([]KindCount, 0, len(counts))
	for kind, count := range counts {
		projected = append(projected, KindCount{Kind: kind, Count: count})
	}
	sort.Slice(projected, func(i, j int) bool {
		return projected[i].Kind.Name() < projected[j].Kind.Name()
	})
	return projected
}
