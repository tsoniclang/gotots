package analyze

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// UnitRef is a typed reference to exactly one implementation unit — a
// source-spanned unit or an implicit unit. It never carries a raw node.
type UnitRef struct {
	source   identity.SourceUnitID
	implicit identity.ImplicitUnitID
}

// SourceUnitRef references a source-spanned unit.
func SourceUnitRef(id identity.SourceUnitID) UnitRef { return UnitRef{source: id} }

// ImplicitUnitRef references an implicit unit.
func ImplicitUnitRef(id identity.ImplicitUnitID) UnitRef { return UnitRef{implicit: id} }

// IsZero reports whether the reference names no unit.
func (r UnitRef) IsZero() bool { return r.source.IsZero() && r.implicit.IsZero() }

// Source is the referenced source unit; zero when implicit.
func (r UnitRef) Source() identity.SourceUnitID { return r.source }

// Implicit is the referenced implicit unit; zero when source-spanned.
func (r UnitRef) Implicit() identity.ImplicitUnitID { return r.implicit }

// Kind is the unit kind (implicit-executable for implicit units).
func (r UnitRef) Kind() identity.UnitKind {
	if !r.source.IsZero() {
		return r.source.Kind()
	}
	return identity.UnitImplicitExecutable
}

// String is the referenced unit's canonical serialization.
func (r UnitRef) String() string {
	if !r.source.IsZero() {
		return r.source.String()
	}
	if !r.implicit.IsZero() {
		return r.implicit.String()
	}
	return ""
}

// RegionOwner identifies what a region belongs to: a file's top-level
// declaration structure, one implementation unit's body, or a package's
// initialization (the owning catalog operation of an implicit unit).
type RegionOwner struct {
	file identity.FileID    // set for the file declaration region
	unit UnitRef            // set for a body region
	pkg  identity.PackageID // set for a package-initialization owner
}

// FileDeclarationOwner is the owner of a file's declaration region.
func FileDeclarationOwner(file identity.FileID) RegionOwner { return RegionOwner{file: file} }

// UnitOwner is the owner of one unit's body region.
func UnitOwner(unit UnitRef) RegionOwner { return RegionOwner{unit: unit} }

// PackageInitializationOwner is the owning catalog operation of a package's
// implicit executable (package-initialization) unit.
func PackageInitializationOwner(pkg identity.PackageID) RegionOwner {
	return RegionOwner{pkg: pkg}
}

// IsFileDeclaration reports whether the owner is a file declaration region.
func (o RegionOwner) IsFileDeclaration() bool { return !o.file.IsZero() }

// IsPackageInitialization reports whether the owner is a package-initialization
// operation.
func (o RegionOwner) IsPackageInitialization() bool { return !o.pkg.IsZero() }

// File is the owning file identity (declaration regions only).
func (o RegionOwner) File() identity.FileID { return o.file }

// Unit is the owning unit reference (body regions only).
func (o RegionOwner) Unit() UnitRef { return o.unit }

// Package is the owning package identity (package-initialization owner only).
func (o RegionOwner) Package() identity.PackageID { return o.pkg }

// String is a stable serialization of the owner for joins and reports.
func (o RegionOwner) String() string {
	switch {
	case o.IsFileDeclaration():
		return "decl:" + o.file.String()
	case o.IsPackageInitialization():
		return "pkginit:" + o.pkg.String()
	default:
		return "unit:" + o.unit.String()
	}
}

// ImplementationDefinition is one implementation unit's definition: its typed
// identity, kind, declaration contract, and evidence depth. A full-semantic
// unit owns a retained body region; a non-full unit is a contract with zero
// body occurrences. The contract is the typed obligation the unit's parent
// retains at its edge even when the body is excised.
type ImplementationDefinition struct {
	unit     UnitRef
	kind     identity.UnitKind
	contract Contract
	depth    source.EvidenceDepth
	full     bool
}

// Unit is the definition's typed unit reference.
func (d ImplementationDefinition) Unit() UnitRef { return d.unit }

// Kind is the unit kind.
func (d ImplementationDefinition) Kind() identity.UnitKind { return d.kind }

// Contract is the unit's declaration contract.
func (d ImplementationDefinition) Contract() Contract { return d.contract }

// Depth is the selected evidence depth.
func (d ImplementationDefinition) Depth() source.EvidenceDepth { return d.depth }

// Full reports whether the definition owns a retained body region.
func (d ImplementationDefinition) Full() bool { return d.full }

// ImplementationRef is a nested implementation reference recorded at the exact
// grammatical edge where a parent region contains a child unit. It preserves
// the parent's operation and the child's declaration contract without retaining
// a raw parent-to-child body pointer, so an excised non-full child never
// removes the contract from its parent.
type ImplementationRef struct {
	parent    RegionOwner
	parentOcc identity.OccurrenceID
	edge      catalog.Edge
	child     UnitRef
	contract  Contract
	anchor    identity.SpanID
	ordinal   int
}

// Parent is the enclosing region owner.
func (r ImplementationRef) Parent() RegionOwner { return r.parent }

// Contract is the child's declaration contract the parent retains at this edge.
func (r ImplementationRef) Contract() Contract { return r.contract }

// ParentOccurrence is the enclosing occurrence the reference hangs from.
func (r ImplementationRef) ParentOccurrence() identity.OccurrenceID { return r.parentOcc }

// Edge is the grammatical edge the child hangs from.
func (r ImplementationRef) Edge() catalog.Edge { return r.edge }

// Role is the parent-assigned grammatical role, derived from the edge.
func (r ImplementationRef) Role() catalog.Role { return r.edge.Role() }

// Child is the referenced child unit.
func (r ImplementationRef) Child() UnitRef { return r.child }

// Anchor is the child root's physical span identity (the source anchor).
func (r ImplementationRef) Anchor() identity.SpanID { return r.anchor }

// Ordinal is the reference's source order within its parent region.
func (r ImplementationRef) Ordinal() int { return r.ordinal }

// IsImplicit reports whether this reference owns an implicit unit — a
// package-initialization reference with no grammatical edge, occurrence, or
// anchor, since an implicit unit has no source span.
func (r ImplementationRef) IsImplicit() bool { return !r.child.Implicit().IsZero() }

// NewImplicitReference builds the owning reference of a package's implicit
// executable unit: the package-initialization operation retains the implicit
// unit with the catalog-owner contract.
func NewImplicitReference(pkg identity.PackageID, child identity.ImplicitUnitID) ImplementationRef {
	return ImplementationRef{
		parent:   PackageInitializationOwner(pkg),
		child:    ImplicitUnitRef(child),
		contract: ContractCatalogOwner,
	}
}
