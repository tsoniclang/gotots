package analyze

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// InventoryArtifactVersion is the schema version of the inventory artifact.
const InventoryArtifactVersion = 1

// Position is one source point: 1-based line and column plus 0-based byte
// offset.
type Position struct {
	Line   int
	Column int
	Offset int
}

// Span is the physical half-open range of one occurrence within its file,
// measured with //line directives ignored so identity never depends on display
// adjustments.
type Span struct {
	Start Position
	End   Position
}

// DisplaySpan is the human-facing position after applying //line directives.
// It exists for diagnostics only and never enters an identity.
type DisplaySpan struct {
	Filename string
	Start    Position
	End      Position
}

// Occurrence is one classified construct instance: canonical identity, kind,
// enclosing occurrence and edge, physical and display spans, lexical token
// evidence, resolved semantic variant, and detected implicit operations. It is
// immutable outside this package; the grammatical role is derived from the
// edge (one owner), never stored twice.
type Occurrence struct {
	id       identity.OccurrenceID
	kind     catalog.Kind
	parent   identity.OccurrenceID
	edge     catalog.Edge
	span     Span
	display  DisplaySpan
	token    catalog.TokenKind
	variant  catalog.Variant
	implicit []catalog.ImplicitOp
}

// ID is the canonical occurrence identity.
func (o Occurrence) ID() identity.OccurrenceID { return o.id }

// Kind is the construct kind.
func (o Occurrence) Kind() catalog.Kind { return o.kind }

// Parent is the enclosing occurrence identity, zero for the root file.
func (o Occurrence) Parent() identity.OccurrenceID { return o.parent }

// Edge is the parent edge this occurrence hangs from, invalid for the root.
func (o Occurrence) Edge() catalog.Edge { return o.edge }

// Role is the grammatical role the parent assigned, derived from the edge.
func (o Occurrence) Role() catalog.Role { return o.edge.Role() }

// Span is the physical span (//line-independent).
func (o Occurrence) Span() Span { return o.span }

// Display is the adjusted human-facing span.
func (o Occurrence) Display() DisplaySpan { return o.display }

// Token is the lexical token evidence for token-bearing kinds (operators,
// keyword-selected declarations, literal kinds); zero when the kind carries no
// token.
func (o Occurrence) Token() catalog.TokenKind { return o.token }

// Variant is the resolved semantic variant; VariantNone for kinds without a
// variant axis.
func (o Occurrence) Variant() catalog.Variant { return o.variant }

// Implicit is the detected implicit-operation evidence (inventory-owned
// members only; the semantic model owns the rest).
func (o Occurrence) Implicit() []catalog.ImplicitOp { return o.implicit }

// DirectiveRecord is one inventoried comment directive.
type DirectiveRecord struct {
	kind    catalog.DirectiveKind
	tool    string
	name    string
	args    string
	span    Span
	display DisplaySpan
}

// Kind is the directive catalog member.
func (d DirectiveRecord) Kind() catalog.DirectiveKind { return d.kind }

// Tool is the directive tool as written.
func (d DirectiveRecord) Tool() string { return d.tool }

// Name is the directive name as written.
func (d DirectiveRecord) Name() string { return d.name }

// Args is the directive argument text.
func (d DirectiveRecord) Args() string { return d.args }

// Span is the physical span.
func (d DirectiveRecord) Span() Span { return d.span }

// Display is the adjusted span.
func (d DirectiveRecord) Display() DisplaySpan { return d.display }

// FileInventory is the immutable inventory artifact of one file: every
// occurrence in parent-directed pre-order plus every comment directive. It is
// constructed only through newFileInventory, which enforces the artifact
// invariants.
type FileInventory struct {
	version            int
	path               string
	file               identity.FileID
	effectiveGoVersion string
	rootUnit           identity.SourceUnitID // set for unit-rooted inventories
	occurrences        []Occurrence
	directives         []DirectiveRecord
}

// newUnitInventory validates a unit-rooted inventory of one retained
// full-semantic unit in a mixed file: the root occurrence must be parentless
// and lie within the unit's span; all other invariants match file
// inventories.
func newUnitInventory(path string, file identity.FileID, effectiveGoVersion string, unit identity.SourceUnitID, occurrences []Occurrence) (*FileInventory, error) {
	if len(occurrences) == 0 {
		return nil, newResolutionError(0, file, Span{}, "unit inventory has no occurrences")
	}
	root := occurrences[0]
	if !root.parent.IsZero() || root.edge != catalog.Edge(0) {
		return nil, newResolutionError(root.kind, file, root.span, "unit inventory root must be parentless")
	}
	// The retained root either equals the unit span (initializer specs) or
	// contains it (a FuncDecl containing its body span). Anything else leaks
	// evidence outside the unit.
	containsUnit := root.span.Start.Offset <= unit.Span().Start() && unit.Span().End() <= root.span.End.Offset
	if !containsUnit {
		return nil, newResolutionError(root.kind, file, root.span, "unit inventory root does not cover its unit span")
	}
	inv, err := validateInventoryBody(path, file, effectiveGoVersion, occurrences, nil)
	if err != nil {
		return nil, err
	}
	inv.rootUnit = unit
	return inv, nil
}

// newFileInventory validates the artifact invariants: a File root with no
// parent/edge, unique canonical identities, parents preceding children,
// edge parents matching, and total variant resolution.
func newFileInventory(path string, file identity.FileID, effectiveGoVersion string, occurrences []Occurrence, directives []DirectiveRecord) (*FileInventory, error) {
	if file.IsZero() {
		return nil, &identity.Error{Identity: "file", Value: path, Reason: "inventory requires a non-zero file identity"}
	}
	if len(occurrences) == 0 {
		return nil, newResolutionError(catalog.KindFile, file, Span{}, "inventory has no occurrences")
	}
	root := occurrences[0]
	if root.kind != catalog.KindFile || !root.parent.IsZero() || root.edge != catalog.Edge(0) {
		return nil, newResolutionError(root.kind, file, root.span, "artifact root must be a parentless File occurrence")
	}
	return validateInventoryBody(path, file, effectiveGoVersion, occurrences, directives)
}

// validateInventoryBody enforces the shared inventory invariants.
func validateInventoryBody(path string, file identity.FileID, effectiveGoVersion string, occurrences []Occurrence, directives []DirectiveRecord) (*FileInventory, error) {
	seen := make(map[identity.OccurrenceID]int, len(occurrences))
	kindsByID := make(map[identity.OccurrenceID]catalog.Kind, len(occurrences))
	for i, occurrence := range occurrences {
		if occurrence.id.IsZero() {
			return nil, newResolutionError(occurrence.kind, file, occurrence.span, "occurrence has zero identity")
		}
		if prior, dup := seen[occurrence.id]; dup {
			return nil, newResolutionError(occurrence.kind, file, occurrence.span,
				"duplicate occurrence identity (also at preorder "+itoa(prior)+")")
		}
		seen[occurrence.id] = i
		kindsByID[occurrence.id] = occurrence.kind
		if i == 0 {
			continue
		}
		parentIdx, ok := seen[occurrence.parent]
		if !ok || parentIdx >= i {
			return nil, newResolutionError(occurrence.kind, file, occurrence.span, "occurrence parent does not precede it")
		}
		if !occurrence.edge.Valid() {
			return nil, newResolutionError(occurrence.kind, file, occurrence.span, "non-root occurrence lacks a parent edge")
		}
		if occurrence.edge.Parent() != kindsByID[occurrence.parent] {
			return nil, newResolutionError(occurrence.kind, file, occurrence.span,
				"edge "+occurrence.edge.String()+" does not belong to parent kind "+kindsByID[occurrence.parent].String())
		}
	}
	for _, occurrence := range occurrences {
		bearing := catalog.VariantBearing(occurrence.kind)
		if bearing && !occurrence.variant.Valid() {
			return nil, newResolutionError(occurrence.kind, file, occurrence.span, "variant-bearing occurrence unresolved")
		}
		if !bearing && occurrence.variant != catalog.VariantNone {
			return nil, newResolutionError(occurrence.kind, file, occurrence.span, "non-variant-bearing occurrence carries a variant")
		}
	}
	return &FileInventory{
		version: InventoryArtifactVersion, path: path, file: file,
		effectiveGoVersion: effectiveGoVersion,
		occurrences:        occurrences, directives: directives,
	}, nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := []byte{}
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	return string(digits)
}

// Version is the artifact schema version.
func (inv *FileInventory) Version() int { return inv.version }

// Path is the display-only OS path the file was loaded from.
func (inv *FileInventory) Path() string { return inv.path }

// File is the canonical file identity.
func (inv *FileInventory) File() identity.FileID { return inv.file }

// EffectiveGoVersion is the file's effective language version from typed
// toolchain evidence; it governs construct admission for this file's
// occurrences. There is no workspace-wide version.
func (inv *FileInventory) EffectiveGoVersion() string { return inv.effectiveGoVersion }

// RootUnit is the retained unit identity for unit-rooted inventories of mixed
// files; zero for whole-file inventories.
func (inv *FileInventory) RootUnit() identity.SourceUnitID { return inv.rootUnit }

// Occurrences is the pre-ordered authoritative occurrence list. Callers must
// not mutate it.
func (inv *FileInventory) Occurrences() []Occurrence { return inv.occurrences }

// Directives is the inventoried comment-directive list. Callers must not
// mutate it.
func (inv *FileInventory) Directives() []DirectiveRecord { return inv.directives }

// KindCount is one projected per-kind count for the report surface.
type KindCount struct {
	Kind  catalog.Kind
	Count int
}

// CountsByKind projects the authoritative occurrences into per-kind counts,
// sorted by kind name. Counts are a projection, never the source of truth.
func (inv *FileInventory) CountsByKind() []KindCount {
	counts := map[catalog.Kind]int{}
	for _, occurrence := range inv.occurrences {
		counts[occurrence.kind]++
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

// PackageInventory is the immutable inventory of one package.
type PackageInventory struct {
	id    identity.PackageID
	files []*FileInventory
}

// ID is the package identity.
func (p *PackageInventory) ID() identity.PackageID { return p.id }

// Files are the per-file inventories in deterministic order.
func (p *PackageInventory) Files() []*FileInventory { return p.files }

// WorkspaceInventory is the immutable whole-workspace inventory artifact.
type WorkspaceInventory struct {
	version  int
	packages []*PackageInventory
}

// Version is the artifact schema version.
func (w *WorkspaceInventory) Version() int { return w.version }

// Packages are the per-package inventories in deterministic order.
func (w *WorkspaceInventory) Packages() []*PackageInventory { return w.packages }

// Denominators are the exact whole-workspace counts every report names.
type Denominators struct {
	Packages       int
	Files          int
	Occurrences    int
	Directives     int
	VariantBearing int
	ImplicitOps    int
}

// Denominators computes the exact denominators of the inventory.
func (w *WorkspaceInventory) Denominators() Denominators {
	var d Denominators
	d.Packages = len(w.packages)
	for _, pkg := range w.packages {
		d.Files += len(pkg.files)
		for _, file := range pkg.files {
			d.Occurrences += len(file.occurrences)
			d.Directives += len(file.directives)
			for _, occurrence := range file.occurrences {
				if catalog.VariantBearing(occurrence.kind) {
					d.VariantBearing++
				}
				d.ImplicitOps += len(occurrence.implicit)
			}
		}
	}
	return d
}
