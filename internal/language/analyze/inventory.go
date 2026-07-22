package analyze

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// Position is one source point: 1-based line and column plus 0-based byte
// offset.
type Position struct {
	Line   int
	Column int
	Offset int
}

// Span is the physical half-open range of one occurrence within its file,
// measured with //line directives ignored so identity never depends on display
// adjustments. The file it ranges over is the inventory's canonical FileID.
type Span struct {
	Start Position
	End   Position
}

// DisplaySpan is the human-facing position of one occurrence after applying
// //line directives. It exists for diagnostics only and never enters an
// identity.
type DisplaySpan struct {
	Filename string
	Start    Position
	End      Position
}

// Occurrence is one classified construct instance in a source file: its
// canonical identity, catalog kind, the identity of its enclosing occurrence
// (empty for the root file), its physical span, and its display position.
// This per-occurrence record is the authoritative inventory; counts are a
// projection of it.
type Occurrence struct {
	ID      identity.OccurrenceID
	Kind    catalog.Kind
	Parent  identity.OccurrenceID
	Span    Span
	Display DisplaySpan
}

// Inventory is the authoritative construct inventory of one file: every
// occurrence in parent-directed pre-order. File is the canonical
// machine-independent identity; Path is the display-only OS path it was loaded
// from.
type Inventory struct {
	Path        string
	File        identity.FileID
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
