// Package structure owns the immutable, target-independent Stage-1 definition
// graph and the canonical structural occurrence store.
package structure

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// ArtifactVersion is the structural artifact schema version.
const ArtifactVersion = 3

// Position is one source point: 1-based line/column and 0-based byte offset.
type Position struct {
	Line   int
	Column int
	Offset int
}

// Span is a physical, half-open source range.
type Span struct {
	Start Position
	End   Position
}

// DisplayPosition is one line-directive-adjusted diagnostic point. Filename is
// a canonical FileID for an unadjusted source point and the exact directive
// filename otherwise; acquisition paths never enter finalized artifacts.
type DisplayPosition struct {
	Filename string
	Line     int
	Column   int
}

// DisplaySpan is diagnostic-only line-directive-adjusted source evidence.
type DisplaySpan struct {
	Start DisplayPosition
	End   DisplayPosition
}

// Occurrence is the one canonical immutable payload for a retained
// grammatical occurrence. Region membership is stored separately.
type Occurrence struct {
	id      identity.OccurrenceID
	kind    catalog.Kind
	parent  identity.OccurrenceID
	edge    catalog.Edge
	ordinal int
	span    Span
	display DisplaySpan
	token   catalog.TokenKind
}

// NewOccurrence validates one canonical grammatical occurrence payload.
func NewOccurrence(
	id identity.OccurrenceID,
	kind catalog.Kind,
	parent identity.OccurrenceID,
	edge catalog.Edge,
	ordinal int,
	span Span,
	display DisplaySpan,
	token catalog.TokenKind,
) (Occurrence, error) {
	if id.IsZero() ||
		!kind.Valid() ||
		id.KindID() != uint16(kind) ||
		id.Span().Start() != span.Start.Offset ||
		id.Span().End() != span.End.Offset ||
		span.Start.Line <= 0 ||
		span.Start.Column <= 0 ||
		span.Start.Offset < 0 ||
		span.End.Line <= 0 ||
		span.End.Column <= 0 ||
		span.End.Offset < span.Start.Offset ||
		display.Start.Filename == "" ||
		display.Start.Line <= 0 ||
		display.Start.Column <= 0 ||
		display.End.Filename == "" ||
		display.End.Line <= 0 ||
		display.End.Column <= 0 ||
		ordinal < 0 ||
		(parent.IsZero() && edge != catalog.EdgeInvalid) ||
		(!parent.IsZero() && !edge.Valid()) ||
		(edge.Valid() && !edge.IsList() && ordinal != 0) ||
		(token != catalog.TokenInvalid && !token.Valid()) ||
		kind.CarriesToken() != token.Valid() {
		return Occurrence{}, fmt.Errorf(
			"occurrence %s has invalid canonical payload", id,
		)
	}
	return Occurrence{
		id:      id,
		kind:    kind,
		parent:  parent,
		edge:    edge,
		ordinal: ordinal,
		span:    span,
		display: display,
		token:   token,
	}, nil
}

func (o Occurrence) ID() identity.OccurrenceID     { return o.id }
func (o Occurrence) Kind() catalog.Kind            { return o.kind }
func (o Occurrence) Parent() identity.OccurrenceID { return o.parent }
func (o Occurrence) Edge() catalog.Edge            { return o.edge }
func (o Occurrence) Role() catalog.Role            { return o.edge.Role() }
func (o Occurrence) Ordinal() int                  { return o.ordinal }
func (o Occurrence) Span() Span                    { return o.span }
func (o Occurrence) Display() DisplaySpan          { return o.display }
func (o Occurrence) Token() catalog.TokenKind      { return o.token }

// Directive is one closed-catalog comment directive.
type Directive struct {
	kind    catalog.DirectiveKind
	tool    string
	name    string
	args    string
	span    Span
	display DisplaySpan
}

func (d Directive) Kind() catalog.DirectiveKind { return d.kind }
func (d Directive) Tool() string                { return d.tool }
func (d Directive) Name() string                { return d.name }
func (d Directive) Args() string                { return d.args }
func (d Directive) Span() Span                  { return d.span }
func (d Directive) Display() DisplaySpan        { return d.display }

// OwnerRegionKind is the closed source/synthetic owner-region domain.
type OwnerRegionKind uint8

const (
	OwnerRegionInvalid OwnerRegionKind = iota
	OwnerRegionSourceFile
	OwnerRegionSynthetic
)

// SyntheticOwnerKind is the closed semantic owner class for structures with
// no source-file identity.
type SyntheticOwnerKind uint8

const (
	SyntheticOwnerInvalid SyntheticOwnerKind = iota
	SyntheticOwnerPackageInitialization
	SyntheticOwnerCgoGenerated
)

func (k SyntheticOwnerKind) Valid() bool {
	return k == SyntheticOwnerPackageInitialization ||
		k == SyntheticOwnerCgoGenerated
}

func (k SyntheticOwnerKind) String() string {
	switch k {
	case SyntheticOwnerPackageInitialization:
		return "package-initialization"
	case SyntheticOwnerCgoGenerated:
		return "cgo-generated"
	default:
		return fmt.Sprintf("structure.SyntheticOwnerKind(%d)", uint8(k))
	}
}

// OwnerRegionID canonically identifies one structural owner region.
type OwnerRegionID struct {
	kind      OwnerRegionKind
	file      identity.FileID
	pkg       identity.PackageID
	synthetic SyntheticOwnerKind
}

func SourceFileOwner(file identity.FileID) (OwnerRegionID, error) {
	if file.IsZero() {
		return OwnerRegionID{}, fmt.Errorf("source owner requires a file identity")
	}
	return OwnerRegionID{kind: OwnerRegionSourceFile, file: file}, nil
}

func SyntheticOwner(
	pkg identity.PackageID,
	kind SyntheticOwnerKind,
) (OwnerRegionID, error) {
	if pkg.IsZero() || !kind.Valid() {
		return OwnerRegionID{}, fmt.Errorf(
			"synthetic owner requires package and closed owner kind",
		)
	}
	return OwnerRegionID{
		kind: OwnerRegionSynthetic, pkg: pkg, synthetic: kind,
	}, nil
}

func (o OwnerRegionID) IsZero() bool                { return o.kind == OwnerRegionInvalid }
func (o OwnerRegionID) Kind() OwnerRegionKind       { return o.kind }
func (o OwnerRegionID) File() identity.FileID       { return o.file }
func (o OwnerRegionID) Package() identity.PackageID { return o.pkg }
func (o OwnerRegionID) SyntheticKind() SyntheticOwnerKind {
	return o.synthetic
}
func (o OwnerRegionID) String() string {
	switch o.kind {
	case OwnerRegionSourceFile:
		return "source:" + o.file.String()
	case OwnerRegionSynthetic:
		return "synthetic:" + o.pkg.String() + "/" + o.synthetic.String()
	default:
		return ""
	}
}

// OwnerRegion references ordinary source occurrences outside definitions.
type OwnerRegion struct {
	id         OwnerRegionID
	members    []identity.OccurrenceID
	directives []Directive
}

func (r OwnerRegion) ID() OwnerRegionID { return r.id }
func (r OwnerRegion) Members() []identity.OccurrenceID {
	return append([]identity.OccurrenceID(nil), r.members...)
}
func (r OwnerRegion) Directives() []Directive {
	return append([]Directive(nil), r.directives...)
}

// ContainmentGraph records only canonical occurrence identities used by one
// normalized sparse containment forest. Occurrence payload is never copied.
type ContainmentGraph struct {
	owner   OwnerRegionID
	anchors []identity.OccurrenceID
}

func (g ContainmentGraph) Owner() OwnerRegionID { return g.owner }
func (g ContainmentGraph) Anchors() []identity.OccurrenceID {
	return append([]identity.OccurrenceID(nil), g.anchors...)
}

// DefinitionSiteKind is the closed source/synthetic site domain.
type DefinitionSiteKind uint8

const (
	DefinitionSiteInvalid DefinitionSiteKind = iota
	DefinitionSiteSource
	DefinitionSiteSynthetic
)

// DefinitionSite is the unique rooted containment site of one definition.
type DefinitionSite struct {
	kind             DefinitionSiteKind
	definition       identity.DefinitionID
	owner            OwnerRegionID
	parentDefinition identity.DefinitionID
	terminal         identity.OccurrenceID
}

func (s DefinitionSite) Kind() DefinitionSiteKind          { return s.kind }
func (s DefinitionSite) Definition() identity.DefinitionID { return s.definition }
func (s DefinitionSite) Owner() OwnerRegionID              { return s.owner }
func (s DefinitionSite) ParentDefinition() identity.DefinitionID {
	return s.parentDefinition
}
func (s DefinitionSite) Terminal() identity.OccurrenceID { return s.terminal }

// HeaderRegion references the complete non-executable occurrence graph of one
// definition. The canonical occurrence store owns every occurrence payload.
type HeaderRegion struct {
	id      identity.HeaderRegionID
	digest  string
	members []identity.OccurrenceID
}

func (h HeaderRegion) ID() identity.HeaderRegionID { return h.id }
func (h HeaderRegion) Digest() string              { return h.digest }
func (h HeaderRegion) Members() []identity.OccurrenceID {
	return append([]identity.OccurrenceID(nil), h.members...)
}

// ExecutionBoundaryKind is the closed boundary representation.
type ExecutionBoundaryKind uint8

const (
	BoundaryInvalid ExecutionBoundaryKind = iota
	BoundaryBlock
	BoundaryInitializers
	BoundaryBodyless
	BoundaryImplicit
)

func (k ExecutionBoundaryKind) Valid() bool {
	return k > BoundaryInvalid && k <= BoundaryImplicit
}

// ExecutionEntry references one canonical source occurrence and owns only its
// independent content hash.
type ExecutionEntry struct {
	id   identity.OccurrenceID
	hash string
}

func (e ExecutionEntry) ID() identity.OccurrenceID { return e.id }
func (e ExecutionEntry) Hash() string              { return e.hash }

// ExecutionBoundary is the exact executable-entry boundary of one definition.
type ExecutionBoundary struct {
	id             identity.ExecutionBoundaryID
	kind           ExecutionBoundaryKind
	entries        []ExecutionEntry
	combinedDigest string
	implicit       identity.ImplicitDefinitionOp
	synthetic      identity.SyntheticDefinitionRole
}

func (b ExecutionBoundary) ID() identity.ExecutionBoundaryID { return b.id }
func (b ExecutionBoundary) Kind() ExecutionBoundaryKind      { return b.kind }
func (b ExecutionBoundary) Entries() []ExecutionEntry {
	return append([]ExecutionEntry(nil), b.entries...)
}
func (b ExecutionBoundary) CombinedDigest() string { return b.combinedDigest }
func (b ExecutionBoundary) ImplicitOp() identity.ImplicitDefinitionOp {
	return b.implicit
}
func (b ExecutionBoundary) SyntheticRole() identity.SyntheticDefinitionRole {
	return b.synthetic
}

// CheckedOriginMatch records the closed toolchain-origin proof used to join a
// cgo checked definition to its original definition.
type CheckedOriginMatch uint8

const (
	CheckedOriginInvalid CheckedOriginMatch = iota
	CheckedOriginExact
	CheckedOriginUniqueLine
)

func (m CheckedOriginMatch) Valid() bool {
	return m == CheckedOriginExact || m == CheckedOriginUniqueLine
}

// CheckedDefinitionMapping records one exact cgo checked-view counterpart.
// It stores the canonical original origin and a content digest of the checked
// syntax; no generated temporary path or disguised source coordinate survives.
type CheckedDefinitionMapping struct {
	definition    identity.DefinitionID
	originLine    int
	originColumn  int
	originMatch   CheckedOriginMatch
	checkedDigest string
}

func (m CheckedDefinitionMapping) Definition() identity.DefinitionID {
	return m.definition
}
func (m CheckedDefinitionMapping) OriginLine() int {
	return m.originLine
}
func (m CheckedDefinitionMapping) OriginColumn() int {
	return m.originColumn
}
func (m CheckedDefinitionMapping) OriginMatch() CheckedOriginMatch {
	return m.originMatch
}
func (m CheckedDefinitionMapping) CheckedDigest() string {
	return m.checkedDigest
}

// ImplementationDefinition is the depth-independent structural definition.
type ImplementationDefinition struct {
	id       identity.DefinitionID
	owner    OwnerRegionID
	header   identity.HeaderRegionID
	boundary identity.ExecutionBoundaryID
	name     string
}

func (d ImplementationDefinition) ID() identity.DefinitionID       { return d.id }
func (d ImplementationDefinition) Kind() identity.DefinitionKind   { return d.id.Kind() }
func (d ImplementationDefinition) Owner() OwnerRegionID            { return d.owner }
func (d ImplementationDefinition) Header() identity.HeaderRegionID { return d.header }
func (d ImplementationDefinition) Boundary() identity.ExecutionBoundaryID {
	return d.boundary
}
func (d ImplementationDefinition) Name() string { return d.name }

// FileGraph is one complete source-file structural graph.
type FileGraph struct {
	owner       OwnerRegion
	occurrences []Occurrence
	containment ContainmentGraph
	definitions []ImplementationDefinition
	sites       []DefinitionSite
	headers     []HeaderRegion
	boundaries  []ExecutionBoundary
	mappings    []CheckedDefinitionMapping
}

func (g FileGraph) Owner() OwnerRegion { return g.owner }
func (g FileGraph) Occurrences() []Occurrence {
	return append([]Occurrence(nil), g.occurrences...)
}
func (g FileGraph) OccurrenceRefs() []OccurrenceRef {
	out := make([]OccurrenceRef, 0, len(g.occurrences))
	for index := range g.occurrences {
		out = append(out, OccurrenceRef{
			occurrence: &g.occurrences[index],
		})
	}
	return out
}
func (g FileGraph) Containment() ContainmentGraph { return g.containment }
func (g FileGraph) Definitions() []ImplementationDefinition {
	return append([]ImplementationDefinition(nil), g.definitions...)
}
func (g FileGraph) Sites() []DefinitionSite {
	return append([]DefinitionSite(nil), g.sites...)
}
func (g FileGraph) Headers() []HeaderRegion {
	return append([]HeaderRegion(nil), g.headers...)
}
func (g FileGraph) Boundaries() []ExecutionBoundary {
	return append([]ExecutionBoundary(nil), g.boundaries...)
}
func (g FileGraph) CheckedMappings() []CheckedDefinitionMapping {
	return append([]CheckedDefinitionMapping(nil), g.mappings...)
}

// PackageGraph is one package's source and synthetic owner graphs.
type PackageGraph struct {
	id               identity.PackageID
	files            []FileGraph
	synthetic        []OwnerRegion
	ownedDefinitions []ImplementationDefinition
	ownedSites       []DefinitionSite
	ownedHeaders     []HeaderRegion
	ownedBoundaries  []ExecutionBoundary
}

func (g PackageGraph) ID() identity.PackageID { return g.id }
func (g PackageGraph) Files() []FileGraph     { return append([]FileGraph(nil), g.files...) }
func (g PackageGraph) SyntheticOwners() []OwnerRegion {
	return append([]OwnerRegion(nil), g.synthetic...)
}
func (g PackageGraph) Definitions() []ImplementationDefinition {
	out := append([]ImplementationDefinition(nil), g.ownedDefinitions...)
	for _, file := range g.files {
		out = append(out, file.definitions...)
	}
	return out
}
func (g PackageGraph) Sites() []DefinitionSite {
	out := append([]DefinitionSite(nil), g.ownedSites...)
	for _, file := range g.files {
		out = append(out, file.sites...)
	}
	return out
}
func (g PackageGraph) Headers() []HeaderRegion {
	out := append([]HeaderRegion(nil), g.ownedHeaders...)
	for _, file := range g.files {
		out = append(out, file.headers...)
	}
	return out
}
func (g PackageGraph) Boundaries() []ExecutionBoundary {
	out := append([]ExecutionBoundary(nil), g.ownedBoundaries...)
	for _, file := range g.files {
		out = append(out, file.boundaries...)
	}
	return out
}
func (g PackageGraph) CheckedMappings() []CheckedDefinitionMapping {
	var out []CheckedDefinitionMapping
	for _, file := range g.files {
		out = append(out, file.mappings...)
	}
	return out
}

// Work records every scalable construction operation class.
type Work struct {
	CatalogEdges    int
	BoundaryProbes  int
	RecordAppends   int
	IdentityProbes  int
	JoinProbes      int
	SortComparisons int
}

type certifiedFileProjection struct {
	id         identity.FileID
	byteDigest string
}

type packageProjection struct {
	id                 identity.PackageID
	certifiedFiles     []certifiedFileProjection
	certifiedSynthetic bool
}

// Graph is the immutable logical whole-universe structural artifact. Locally
// extracted package records are resident. Certified records remain in their
// content-addressed provider artifact and are projected one package at a time.
type Graph struct {
	version           int
	packages          []PackageGraph
	projections       []packageProjection
	provider          *ProviderArtifact
	definitions       []DefinitionCensusRecord
	definitionByID    map[identity.DefinitionID]*DefinitionCensusRecord
	headerOccurrences int
	boundaryEntries   int
	occurrenceIDs     []identity.OccurrenceID
	byOccurrence      map[identity.OccurrenceID]*Occurrence
	definitionIDs     []identity.DefinitionID
	byDefinition      map[identity.DefinitionID]*ImplementationDefinition
	byBoundary        map[identity.DefinitionID]*ExecutionBoundary
	work              Work
}

func (g *Graph) Version() int { return g.version }
func (g *Graph) Work() Work   { return g.work }
func (g *Graph) residentOccurrences() []Occurrence {
	out := make([]Occurrence, 0, len(g.occurrenceIDs))
	for _, id := range g.occurrenceIDs {
		out = append(out, *g.byOccurrence[id])
	}
	return out
}
func (g *Graph) residentOccurrence(
	id identity.OccurrenceID,
) (Occurrence, bool) {
	occurrence, ok := g.byOccurrence[id]
	if !ok {
		return Occurrence{}, false
	}
	return *occurrence, true
}
func (g *Graph) residentDefinitions() []ImplementationDefinition {
	out := make([]ImplementationDefinition, 0, len(g.definitionIDs))
	for _, id := range g.definitionIDs {
		out = append(out, *g.byDefinition[id])
	}
	return out
}
func (g *Graph) residentDefinition(
	id identity.DefinitionID,
) (ImplementationDefinition, bool) {
	definition, ok := g.byDefinition[id]
	if !ok {
		return ImplementationDefinition{}, false
	}
	return *definition, true
}
func (g *Graph) residentBoundary(
	id identity.DefinitionID,
) (ExecutionBoundary, bool) {
	boundary, ok := g.byBoundary[id]
	if !ok {
		return ExecutionBoundary{}, false
	}
	return *boundary, true
}

func sortPackageGraphs(packages []PackageGraph, work *Work) {
	sort.Slice(packages, func(i, j int) bool {
		work.SortComparisons++
		return packages[i].id.String() < packages[j].id.String()
	})
}
