package identity

import "fmt"

// ImplicitUnitOp is the closed, pinned operation vocabulary of unspelled
// implicit executable units. An implicit unit is executable work the language
// defines without any source spelling; its identity is its semantic owner plus
// the operation, never a source span.
type ImplicitUnitOp uint8

// Explicit, permanent implicit-operation identities. Do not renumber; append
// only.
const (
	ImplicitOpInvalid ImplicitUnitOp = 0
	// ImplicitOpPackageInit: the package's implicit initialization unit — the
	// language-defined driver that runs package-level variable initializers
	// and init functions in specification order.
	ImplicitOpPackageInit ImplicitUnitOp = 1

	numImplicitUnitOps = 2
)

var implicitUnitOpNames = [numImplicitUnitOps]string{
	ImplicitOpPackageInit: "package-init",
}

// Valid reports whether o names an implicit operation.
func (o ImplicitUnitOp) Valid() bool { return o > ImplicitOpInvalid && o < numImplicitUnitOps }

// String renders o for reports.
func (o ImplicitUnitOp) String() string {
	if o.Valid() {
		return implicitUnitOpNames[o]
	}
	return fmt.Sprintf("identity.ImplicitUnitOp(%d)", uint8(o))
}

// ImplicitUnitID is the canonical identity of one unspelled implicit
// executable unit: its semantic owner (the package) plus the pinned catalog
// operation. It never carries a source span, an offset, or a display name.
type ImplicitUnitID struct {
	pkg PackageID
	op  ImplicitUnitOp
}

// NewImplicitUnitID validates one implicit-unit identity.
func NewImplicitUnitID(pkg PackageID, op ImplicitUnitOp) (ImplicitUnitID, error) {
	if pkg.IsZero() {
		return ImplicitUnitID{}, &Error{Identity: "implicit-unit", Value: "", Reason: "owning package identity must not be zero"}
	}
	if !op.Valid() {
		return ImplicitUnitID{}, &Error{Identity: "implicit-unit", Value: pkg.String(), Reason: "implicit operation must be valid"}
	}
	return ImplicitUnitID{pkg: pkg, op: op}, nil
}

// IsZero reports whether i is the invalid zero identity.
func (i ImplicitUnitID) IsZero() bool { return i == ImplicitUnitID{} }

// Pkg is the owning package identity.
func (i ImplicitUnitID) Pkg() PackageID { return i.pkg }

// Op is the pinned implicit operation.
func (i ImplicitUnitID) Op() ImplicitUnitOp { return i.op }

// String is the canonical serialization: package#implicit/<op>.
func (i ImplicitUnitID) String() string {
	return i.pkg.String() + "#implicit/" + i.op.String()
}
