package source

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// EvidenceDepth is the closed retained-evidence class of one implementation
// unit. Selection is owned by the analysis-scope phase (internal/scope)
// consuming the provider contract; source only records and enforces the
// selected depth. There is no default member.
type EvidenceDepth uint8

const (
	DepthInvalid EvidenceDepth = iota
	// DepthFullSemantic: declarations, checked syntax, contextual
	// occurrences, and boundary records are retained.
	DepthFullSemantic
	// DepthDeclarationContract: complete declaration/type contracts plus
	// unit identity/span/hash boundaries; zero interior occurrences.
	DepthDeclarationContract
	// DepthExternalBoundary: source boundary, checked-view mapping where
	// available, and a typed unresolved obligation.
	DepthExternalBoundary
	// DepthIntrinsic: typed language/toolchain contract with no ordinary
	// source body.
	DepthIntrinsic

	numEvidenceDepths
)

var evidenceDepthNames = [numEvidenceDepths]string{
	DepthFullSemantic: "full-semantic", DepthDeclarationContract: "declaration-contract",
	DepthExternalBoundary: "external-boundary", DepthIntrinsic: "intrinsic",
}

// Valid reports whether d names an evidence depth.
func (d EvidenceDepth) Valid() bool { return d > DepthInvalid && d < numEvidenceDepths }

// String renders d for reports.
func (d EvidenceDepth) String() string {
	if d.Valid() {
		return evidenceDepthNames[d]
	}
	return fmt.Sprintf("source.EvidenceDepth(%d)", uint8(d))
}

// SourceSpanHash is the SHA-256 of one unit's selected source-span bytes
// (overlay-aware). It is selected-input evidence: distinct from canonical
// identity and from the later generated/manual post-format body hashes.
type SourceSpanHash [32]byte

// IsZero reports whether the hash is unset.
func (h SourceSpanHash) IsZero() bool { return h == SourceSpanHash{} }

// String renders the hash in hex for reports.
func (h SourceSpanHash) String() string { return fmt.Sprintf("%x", h[:]) }

// SourceUnit is one censused implementation unit of a source file: canonical
// identity, display name, physical span, selected-input hash, per-unit cgo
// evidence, and — after scope selection — its evidence depth. The census is
// total over the closed identity.UnitKind vocabulary — function/method
// bodies, recursively censused function literals, package-level initializers
// (one per ValueSpec), and bodyless declarations with implementation
// obligations — BEFORE scope selection; nothing creates missing units later.
type SourceUnit struct {
	id         identity.SourceUnitID
	name       string // display only, never identity
	span       Span
	hash       SourceSpanHash
	cDependent bool // the unit's source references the cgo "C" pseudo-package
	depth      EvidenceDepth
}

// ID is the canonical unit identity.
func (u SourceUnit) ID() identity.SourceUnitID { return u.id }

// Kind is the unit kind.
func (u SourceUnit) Kind() identity.UnitKind { return u.id.Kind() }

// Name is the display name (never identity).
func (u SourceUnit) Name() string { return u.name }

// Span is the unit's physical span.
func (u SourceUnit) Span() Span { return u.span }

// Hash is the unit's selected-input source-span hash.
func (u SourceUnit) Hash() SourceSpanHash { return u.hash }

// CDependent reports whether the unit's source references the cgo "C"
// pseudo-package (per-unit evidence for scope selection, never a policy).
func (u SourceUnit) CDependent() bool { return u.cDependent }

// Depth is the scope-selected evidence depth (valid only on finalized
// artifacts).
func (u SourceUnit) Depth() EvidenceDepth { return u.depth }

// Position is one source point: 1-based line and column plus 0-based byte
// offset.
type Position struct {
	Line   int
	Column int
	Offset int
}

// Span is a physical half-open source range (//line-independent).
type Span struct {
	Start Position
	End   Position
}

// SyntheticRole is the closed role class of a cgo-synthetic checked
// declaration, derived from the declaration's own kind — never from a file
// path or name spelling.
type SyntheticRole uint8

const (
	SyntheticRoleInvalid SyntheticRole = iota
	// SyntheticAdapter: a synthetic function/call adapter.
	SyntheticAdapter
	// SyntheticTypeDecl: a synthetic type declaration.
	SyntheticTypeDecl
	// SyntheticData: a synthetic variable/constant declaration.
	SyntheticData

	numSyntheticRoles
)

var syntheticRoleNames = [numSyntheticRoles]string{
	SyntheticAdapter: "adapter", SyntheticTypeDecl: "type", SyntheticData: "data",
}

// Valid reports whether r names a synthetic role.
func (r SyntheticRole) Valid() bool { return r > SyntheticRoleInvalid && r < numSyntheticRoles }

// String renders r for reports.
func (r SyntheticRole) String() string {
	if r.Valid() {
		return syntheticRoleNames[r]
	}
	return fmt.Sprintf("source.SyntheticRole(%d)", uint8(r))
}

// SyntheticUnit is one package-synthetic checked declaration produced by cgo
// preprocessing. Its constructor-validated identity is the owning package
// plus the declared scope name plus the closed role — never a temporary path
// or display spelling. Synthetic units are typed external/intrinsic evidence;
// they are never ignored.
type SyntheticUnit struct {
	pkg  identity.PackageID
	name string
	role SyntheticRole
}

// NewSyntheticUnit validates one synthetic-unit identity.
func NewSyntheticUnit(pkg identity.PackageID, name string, role SyntheticRole) (SyntheticUnit, error) {
	if pkg.IsZero() {
		return SyntheticUnit{}, &LoadError{Reason: "synthetic unit requires a package identity"}
	}
	if name == "" {
		return SyntheticUnit{}, &LoadError{Reason: "synthetic unit requires its declared scope name"}
	}
	if !role.Valid() {
		return SyntheticUnit{}, &LoadError{Reason: "synthetic unit requires a valid role"}
	}
	return SyntheticUnit{pkg: pkg, name: name, role: role}, nil
}

// Pkg is the owning package identity.
func (s SyntheticUnit) Pkg() identity.PackageID { return s.pkg }

// Name is the declared name in the checked package scope.
func (s SyntheticUnit) Name() string { return s.name }

// Role is the closed synthetic role class.
func (s SyntheticUnit) Role() SyntheticRole { return s.role }

// CheckedUnitMapping joins one source-derived checked declaration back to its
// original source unit through toolchain line-directive evidence. Checked
// paths are transient acquisition data and never identity.
type CheckedUnitMapping struct {
	source  identity.SourceUnitID
	checked Span // span within the checked view, display evidence only
}

// Source is the original unit identity.
func (m CheckedUnitMapping) Source() identity.SourceUnitID { return m.source }

// Checked is the checked-view span (display evidence).
func (m CheckedUnitMapping) Checked() Span { return m.checked }
