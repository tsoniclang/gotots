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
// declaration structure, or one implementation unit's body.
type RegionOwner struct {
	file identity.FileID // set for the file declaration region
	unit UnitRef         // set for a body region
}

// FileDeclarationOwner is the owner of a file's declaration region.
func FileDeclarationOwner(file identity.FileID) RegionOwner { return RegionOwner{file: file} }

// UnitOwner is the owner of one unit's body region.
func UnitOwner(unit UnitRef) RegionOwner { return RegionOwner{unit: unit} }

// IsFileDeclaration reports whether the owner is a file declaration region.
func (o RegionOwner) IsFileDeclaration() bool { return o.unit.IsZero() }

// File is the owning file identity (declaration regions only).
func (o RegionOwner) File() identity.FileID { return o.file }

// Unit is the owning unit reference (body regions only).
func (o RegionOwner) Unit() UnitRef { return o.unit }

// String is a stable serialization of the owner for joins and reports.
func (o RegionOwner) String() string {
	if o.IsFileDeclaration() {
		return "decl:" + o.file.String()
	}
	return "unit:" + o.unit.String()
}

// ImplementationDefinition is one implementation unit's definition: its typed
// identity, kind, and evidence depth. A full-semantic unit owns a retained
// body region; a non-full unit is a contract with zero body occurrences.
type ImplementationDefinition struct {
	unit  UnitRef
	kind  identity.UnitKind
	depth source.EvidenceDepth
	full  bool
}

// Unit is the definition's typed unit reference.
func (d ImplementationDefinition) Unit() UnitRef { return d.unit }

// Kind is the unit kind.
func (d ImplementationDefinition) Kind() identity.UnitKind { return d.kind }

// Depth is the selected evidence depth.
func (d ImplementationDefinition) Depth() source.EvidenceDepth { return d.depth }

// Full reports whether the definition owns a retained body region.
func (d ImplementationDefinition) Full() bool { return d.full }

// ImplementationRef is a nested implementation reference recorded at the exact
// grammatical edge where a parent region contains a child unit. It preserves
// the parent's operation without retaining a raw parent-to-child body pointer.
type ImplementationRef struct {
	parent    RegionOwner
	parentOcc identity.OccurrenceID
	edge      catalog.Edge
	child     UnitRef
	anchor    identity.SpanID
	ordinal   int
}

// Parent is the enclosing region owner.
func (r ImplementationRef) Parent() RegionOwner { return r.parent }

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
