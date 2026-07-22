package identity

import "fmt"

// UnitKind is the closed, total kind of one source implementation unit. Every
// executable or implementation-obligated unit in a source file is exactly one
// of these; there is no default member and "function bodies plus var
// initializers" alone is never the total ledger.
//
// Initializer granularity is defined here, once: one unit per ValueSpec
// (covering every name the spec binds, including grouped and multi-result
// initializers), spanning the complete spec. Producers and verifiers may not
// count initializers differently.
type UnitKind uint8

// Explicit, permanent unit-kind identities. Do not renumber; append only.
const (
	UnitInvalid UnitKind = 0
	// UnitFuncBody: a function or method declaration with a Go body.
	UnitFuncBody UnitKind = 1
	// UnitFuncLitBody: a function literal, censused recursively BEFORE scope
	// selection — the pre-scope ledger is total; construct inventory never
	// creates missing units later.
	UnitFuncLitBody UnitKind = 2
	// UnitVarInitializer: one package-level ValueSpec with initializer
	// expressions.
	UnitVarInitializer UnitKind = 3
	// UnitBodylessDecl: a function/method declaration with no Go body — an
	// implementation obligation satisfied by assembly, linkname, wasm, or an
	// external contract. Its span is the declaration span.
	UnitBodylessDecl UnitKind = 4
	// UnitImplicitExecutable: catalog-owned implicit executable work. Its
	// identity is the owning catalog/typed contract, never a fabricated
	// source span; it does not construct a SourceUnitID.
	UnitImplicitExecutable UnitKind = 5

	numUnitKinds = 6
)

var unitKindNames = [numUnitKinds]string{
	UnitFuncBody: "func-body", UnitFuncLitBody: "funclit-body",
	UnitVarInitializer: "var-initializer", UnitBodylessDecl: "bodyless-decl",
	UnitImplicitExecutable: "implicit-executable",
}

// Valid reports whether k names a unit kind.
func (k UnitKind) Valid() bool { return k > UnitInvalid && k < numUnitKinds }

// String renders k for reports.
func (k UnitKind) String() string {
	if k.Valid() {
		return unitKindNames[k]
	}
	return fmt.Sprintf("identity.UnitKind(%d)", uint8(k))
}

// SourceUnitID is the canonical identity of one source implementation unit:
// its physical span plus its unit kind. Display names never enter the
// identity.
type SourceUnitID struct {
	span SpanID
	kind UnitKind
}

// NewSourceUnitID validates one source-unit identity.
func NewSourceUnitID(span SpanID, kind UnitKind) (SourceUnitID, error) {
	if span.IsZero() {
		return SourceUnitID{}, &Error{Identity: "source-unit", Value: "", Reason: "span identity must not be zero"}
	}
	if !kind.Valid() {
		return SourceUnitID{}, &Error{Identity: "source-unit", Value: span.String(), Reason: "unit kind must be valid"}
	}
	if kind == UnitImplicitExecutable {
		return SourceUnitID{}, &Error{Identity: "source-unit", Value: span.String(),
			Reason: "implicit executable work carries a typed catalog identity, never a fabricated source span"}
	}
	return SourceUnitID{span: span, kind: kind}, nil
}

// IsZero reports whether u is the invalid zero identity.
func (u SourceUnitID) IsZero() bool { return u == SourceUnitID{} }

// Span is the unit's physical span identity.
func (u SourceUnitID) Span() SpanID { return u.span }

// Kind is the unit kind.
func (u SourceUnitID) Kind() UnitKind { return u.kind }

// String is the canonical serialization: file#start-end/<kind>.
func (u SourceUnitID) String() string {
	return u.span.String() + "/" + u.kind.String()
}
