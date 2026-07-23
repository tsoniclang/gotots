package identity

import "fmt"

// DefinitionKind is the closed, pinned class of one independently selected
// implementation definition. It names implementation ownership, not an
// executable entry such as a function body block or initializer expression.
type DefinitionKind uint8

const (
	DefinitionInvalid            DefinitionKind = 0
	DefinitionFuncDecl           DefinitionKind = 1
	DefinitionFuncLit            DefinitionKind = 2
	DefinitionPackageInitializer DefinitionKind = 3
	DefinitionBodylessDecl       DefinitionKind = 4
	DefinitionImplicit           DefinitionKind = 5

	definitionKindCount = 5
)

var definitionKindNames = [definitionKindCount + 1]string{
	DefinitionFuncDecl:           "func-decl",
	DefinitionFuncLit:            "func-lit",
	DefinitionPackageInitializer: "package-initializer",
	DefinitionBodylessDecl:       "bodyless-decl",
	DefinitionImplicit:           "implicit",
}

// Valid reports whether k is a defined implementation-definition class.
func (k DefinitionKind) Valid() bool {
	return k > DefinitionInvalid && k <= definitionKindCount
}

// Source reports whether k must be anchored to a source construct root.
func (k DefinitionKind) Source() bool {
	return k.Valid() && k != DefinitionImplicit
}

// String renders the pinned canonical spelling.
func (k DefinitionKind) String() string {
	if k.Valid() {
		return definitionKindNames[k]
	}
	return fmt.Sprintf("identity.DefinitionKind(%d)", uint8(k))
}

// ImplicitDefinitionOp is the closed semantic owner vocabulary for unspelled
// implementation definitions.
type ImplicitDefinitionOp uint8

const (
	ImplicitDefinitionInvalid     ImplicitDefinitionOp = 0
	ImplicitDefinitionPackageInit ImplicitDefinitionOp = 1

	implicitDefinitionOpCount = 1
)

var implicitDefinitionOpNames = [implicitDefinitionOpCount + 1]string{
	ImplicitDefinitionPackageInit: "package-init",
}

// Valid reports whether o is a defined implicit implementation operation.
func (o ImplicitDefinitionOp) Valid() bool {
	return o > ImplicitDefinitionInvalid && o <= implicitDefinitionOpCount
}

// String renders the pinned canonical spelling.
func (o ImplicitDefinitionOp) String() string {
	if o.Valid() {
		return implicitDefinitionOpNames[o]
	}
	return fmt.Sprintf("identity.ImplicitDefinitionOp(%d)", uint8(o))
}

// DefinitionID identifies one implementation definition. A source definition
// is anchored to the occurrence identity of its complete construct root. An
// implicit definition is anchored to its semantic package owner and pinned
// operation. The two forms are disjoint.
type DefinitionID struct {
	sourceKind DefinitionKind
	root       OccurrenceID
	pkg        PackageID
	implicit   ImplicitDefinitionOp
	synthetic  SyntheticDefinitionRole
	name       string
}

// NewSourceDefinitionID validates a construct-root definition identity.
func NewSourceDefinitionID(root OccurrenceID, kind DefinitionKind) (DefinitionID, error) {
	if root.IsZero() {
		return DefinitionID{}, &Error{Identity: "definition", Reason: "source definition requires a construct-root occurrence"}
	}
	if !kind.Source() {
		return DefinitionID{}, &Error{Identity: "definition", Value: root.String(), Reason: "source definition requires a source definition kind"}
	}
	return DefinitionID{sourceKind: kind, root: root}, nil
}

// NewImplicitDefinitionID validates an unspelled implementation identity.
func NewImplicitDefinitionID(pkg PackageID, op ImplicitDefinitionOp) (DefinitionID, error) {
	if pkg.IsZero() {
		return DefinitionID{}, &Error{Identity: "definition", Reason: "implicit definition requires a package identity"}
	}
	if !op.Valid() {
		return DefinitionID{}, &Error{Identity: "definition", Value: pkg.String(), Reason: "implicit definition requires a valid operation"}
	}
	return DefinitionID{sourceKind: DefinitionImplicit, pkg: pkg, implicit: op}, nil
}

// SyntheticDefinitionRole is the closed identity role of one tool-generated
// definition that has no stable source-file identity.
type SyntheticDefinitionRole uint8

const (
	SyntheticDefinitionInvalid SyntheticDefinitionRole = iota
	SyntheticDefinitionAdapter
	SyntheticDefinitionType
	SyntheticDefinitionData
)

func (r SyntheticDefinitionRole) Valid() bool {
	return r >= SyntheticDefinitionAdapter && r <= SyntheticDefinitionData
}

func (r SyntheticDefinitionRole) String() string {
	switch r {
	case SyntheticDefinitionAdapter:
		return "adapter"
	case SyntheticDefinitionType:
		return "type"
	case SyntheticDefinitionData:
		return "data"
	default:
		return fmt.Sprintf("identity.SyntheticDefinitionRole(%d)", uint8(r))
	}
}

// NewSyntheticDefinitionID validates a package-owned generated definition.
// Its identity is semantic package+role+declared-name evidence, never a
// temporary checked-view path.
func NewSyntheticDefinitionID(
	pkg PackageID,
	role SyntheticDefinitionRole,
	name string,
) (DefinitionID, error) {
	if pkg.IsZero() || !role.Valid() || name == "" || hasReserved(name) {
		return DefinitionID{}, &Error{
			Identity: "definition", Value: name,
			Reason: "synthetic definition requires package, closed role, and canonical name",
		}
	}
	return DefinitionID{
		sourceKind: DefinitionImplicit, pkg: pkg, synthetic: role, name: name,
	}, nil
}

// IsZero reports whether d is the invalid zero identity.
func (d DefinitionID) IsZero() bool { return d == DefinitionID{} }

// Kind is the definition's pinned class.
func (d DefinitionID) Kind() DefinitionKind { return d.sourceKind }

// Root is the source construct-root occurrence; it is zero for implicit
// definitions.
func (d DefinitionID) Root() OccurrenceID { return d.root }

// Package is the semantic owner of an implicit definition; it is zero for
// source definitions.
func (d DefinitionID) Package() PackageID { return d.pkg }

// ImplicitOp is the semantic operation of an implicit definition.
func (d DefinitionID) ImplicitOp() ImplicitDefinitionOp { return d.implicit }

// SyntheticRole is the closed generated-definition role.
func (d DefinitionID) SyntheticRole() SyntheticDefinitionRole { return d.synthetic }

// SyntheticName is the generated definition's declared package-scope name.
func (d DefinitionID) SyntheticName() string { return d.name }

// File is the owning source file; it is zero for implicit definitions.
func (d DefinitionID) File() FileID {
	if d.root.IsZero() {
		return FileID{}
	}
	return d.root.Span().File()
}

// String is the canonical serialization.
func (d DefinitionID) String() string {
	if !d.root.IsZero() {
		return d.root.String() + "/D" + fmt.Sprint(uint8(d.sourceKind))
	}
	if !d.pkg.IsZero() && d.implicit.Valid() {
		return d.pkg.String() + "#definition/" + d.implicit.String()
	}
	if !d.pkg.IsZero() && d.synthetic.Valid() && d.name != "" {
		return d.pkg.String() + "#synthetic/" + d.synthetic.String() + "/" + d.name
	}
	return ""
}

// HeaderRegionID identifies the one header region owned by a definition.
type HeaderRegionID struct{ definition DefinitionID }

// NewHeaderRegionID validates a header-region identity.
func NewHeaderRegionID(definition DefinitionID) (HeaderRegionID, error) {
	if definition.IsZero() {
		return HeaderRegionID{}, &Error{Identity: "header-region", Reason: "definition identity must not be zero"}
	}
	return HeaderRegionID{definition: definition}, nil
}

// IsZero reports whether h is invalid.
func (h HeaderRegionID) IsZero() bool { return h == HeaderRegionID{} }

// Definition is the owning definition.
func (h HeaderRegionID) Definition() DefinitionID { return h.definition }

// String is the canonical serialization.
func (h HeaderRegionID) String() string {
	if h.IsZero() {
		return ""
	}
	return h.definition.String() + "#header"
}

// ExecutionBoundaryID identifies the one execution boundary owned by a
// definition.
type ExecutionBoundaryID struct{ definition DefinitionID }

// NewExecutionBoundaryID validates an execution-boundary identity.
func NewExecutionBoundaryID(definition DefinitionID) (ExecutionBoundaryID, error) {
	if definition.IsZero() {
		return ExecutionBoundaryID{}, &Error{Identity: "execution-boundary", Reason: "definition identity must not be zero"}
	}
	return ExecutionBoundaryID{definition: definition}, nil
}

// IsZero reports whether b is invalid.
func (b ExecutionBoundaryID) IsZero() bool { return b == ExecutionBoundaryID{} }

// Definition is the owning definition.
func (b ExecutionBoundaryID) Definition() DefinitionID { return b.definition }

// String is the canonical serialization.
func (b ExecutionBoundaryID) String() string {
	if b.IsZero() {
		return ""
	}
	return b.definition.String() + "#execution"
}

// ExecutableRegionID identifies the optional executable region of one
// full-semantic definition.
type ExecutableRegionID struct{ definition DefinitionID }

// NewExecutableRegionID validates an executable-region identity.
func NewExecutableRegionID(definition DefinitionID) (ExecutableRegionID, error) {
	if definition.IsZero() {
		return ExecutableRegionID{}, &Error{Identity: "executable-region", Reason: "definition identity must not be zero"}
	}
	return ExecutableRegionID{definition: definition}, nil
}

// IsZero reports whether r is invalid.
func (r ExecutableRegionID) IsZero() bool { return r == ExecutableRegionID{} }

// Definition is the owning definition.
func (r ExecutableRegionID) Definition() DefinitionID { return r.definition }

// String is the canonical serialization.
func (r ExecutableRegionID) String() string {
	if r.IsZero() {
		return ""
	}
	return r.definition.String() + "#executable"
}
