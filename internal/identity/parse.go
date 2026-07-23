package identity

import (
	"strconv"
	"strings"
)

// Parsing reconstructs identities from their canonical serializations through
// the same validating constructors that produced them. A parse succeeds only
// when re-rendering the result reproduces the input exactly — noncanonical
// spellings never enter the identity space.

// ParseOwner reconstructs an owner from its canonical serialization.
func ParseOwner(s string) (Owner, error) {
	switch {
	case s == "std":
		return StandardLibraryOwner(), nil
	case s == "toolchain":
		return ToolchainOwner(), nil
	case s == "lang":
		return LanguagePseudoOwner(), nil
	case strings.HasPrefix(s, "mod="):
		spec := s[len("mod="):]
		path, version := spec, ""
		if at := strings.Index(spec, "@"); at >= 0 {
			path, version = spec[:at], spec[at+1:]
		}
		module, err := NewModuleID(path, version)
		if err != nil {
			return Owner{}, err
		}
		return NewModuleOwner(module)
	}
	return Owner{}, &Error{Identity: "owner", Value: s, Reason: "not a canonical owner serialization"}
}

// ParsePackageID reconstructs a package identity from owner::importPath.
func ParsePackageID(s string) (PackageID, error) {
	sep := strings.Index(s, "::")
	if sep < 0 {
		return PackageID{}, &Error{Identity: "package", Value: s, Reason: "not a canonical package serialization"}
	}
	owner, err := ParseOwner(s[:sep])
	if err != nil {
		return PackageID{}, err
	}
	pkg, err := NewPackageID(owner, s[sep+2:])
	if err != nil {
		return PackageID{}, err
	}
	if pkg.String() != s {
		return PackageID{}, &Error{Identity: "package", Value: s, Reason: "serialization is not canonical"}
	}
	return pkg, nil
}

// ParseFileID reconstructs a file identity from owner::rel.
func ParseFileID(s string) (FileID, error) {
	sep := strings.Index(s, "::")
	if sep < 0 {
		return FileID{}, &Error{Identity: "file", Value: s, Reason: "not a canonical file serialization"}
	}
	owner, err := ParseOwner(s[:sep])
	if err != nil {
		return FileID{}, err
	}
	file, err := NewFileID(owner, s[sep+2:])
	if err != nil {
		return FileID{}, err
	}
	if file.String() != s {
		return FileID{}, &Error{Identity: "file", Value: s, Reason: "serialization is not canonical"}
	}
	return file, nil
}

// ParseSpanID reconstructs a span identity from owner::rel#start-end.
func ParseSpanID(s string) (SpanID, error) {
	fail := func(reason string) (SpanID, error) {
		return SpanID{}, &Error{Identity: "span", Value: s, Reason: reason}
	}
	hash := strings.LastIndex(s, "#")
	if hash < 0 {
		return fail("not a canonical span serialization")
	}
	file, err := ParseFileID(s[:hash])
	if err != nil {
		return SpanID{}, err
	}
	dash := strings.Index(s[hash+1:], "-")
	if dash < 0 {
		return fail("malformed span range")
	}
	rangePart := s[hash+1:]
	start, err := strconv.Atoi(rangePart[:dash])
	if err != nil {
		return fail("malformed span start")
	}
	end, err := strconv.Atoi(rangePart[dash+1:])
	if err != nil {
		return fail("malformed span end")
	}
	span, err := NewSpanID(file, start, end)
	if err != nil {
		return SpanID{}, err
	}
	if span.String() != s {
		return fail("serialization is not canonical")
	}
	return span, nil
}

// ParseOccurrenceID reconstructs an occurrence identity from
// owner::rel#start-end/K<kindID>.
func ParseOccurrenceID(s string) (OccurrenceID, error) {
	fail := func(reason string) (OccurrenceID, error) {
		return OccurrenceID{}, &Error{Identity: "occurrence", Value: s, Reason: reason}
	}
	slashK := strings.LastIndex(s, "/K")
	if slashK < 0 {
		return fail("not a canonical occurrence serialization")
	}
	span, err := ParseSpanID(s[:slashK])
	if err != nil {
		return OccurrenceID{}, err
	}
	kindID, err := strconv.ParseUint(s[slashK+2:], 10, 16)
	if err != nil {
		return fail("malformed kind identity")
	}
	occ, err := NewOccurrenceID(span, uint16(kindID))
	if err != nil {
		return OccurrenceID{}, err
	}
	if occ.String() != s {
		return fail("serialization is not canonical")
	}
	return occ, nil
}

// ParseSourceUnitID reconstructs a source-unit identity from
// owner::rel#start-end/kind.
func ParseSourceUnitID(s string) (SourceUnitID, error) {
	fail := func(reason string) (SourceUnitID, error) {
		return SourceUnitID{}, &Error{Identity: "source-unit", Value: s, Reason: reason}
	}
	hash := strings.Index(s, "#")
	if hash < 0 {
		return fail("not a canonical source-unit serialization")
	}
	filePart, rest := s[:hash], s[hash+1:]
	slash := strings.LastIndex(rest, "/")
	if slash < 0 {
		return fail("missing unit kind")
	}
	spanPart, kindName := rest[:slash], rest[slash+1:]
	kind := UnitInvalid
	for k := UnitKind(1); k.Valid(); k++ {
		if k.String() == kindName {
			kind = k
		}
	}
	if kind == UnitInvalid {
		return fail("unknown unit kind " + kindName)
	}
	dash := strings.Index(spanPart, "-")
	if dash < 0 {
		return fail("malformed span")
	}
	start, err := strconv.Atoi(spanPart[:dash])
	if err != nil {
		return fail("malformed span start")
	}
	end, err := strconv.Atoi(spanPart[dash+1:])
	if err != nil {
		return fail("malformed span end")
	}
	sep := strings.Index(filePart, "::")
	if sep < 0 {
		return fail("missing file identity")
	}
	owner, err := ParseOwner(filePart[:sep])
	if err != nil {
		return SourceUnitID{}, err
	}
	file, err := NewFileID(owner, filePart[sep+2:])
	if err != nil {
		return SourceUnitID{}, err
	}
	span, err := NewSpanID(file, start, end)
	if err != nil {
		return SourceUnitID{}, err
	}
	unit, err := NewSourceUnitID(span, kind)
	if err != nil {
		return SourceUnitID{}, err
	}
	if unit.String() != s {
		return fail("serialization is not canonical")
	}
	return unit, nil
}

// implicitMarker separates the owning package from the implicit operation in
// the canonical implicit-unit serialization.
const implicitMarker = "#implicit/"

// ParseImplicitUnitID reconstructs an implicit-unit identity from
// package#implicit/op.
func ParseImplicitUnitID(s string) (ImplicitUnitID, error) {
	fail := func(reason string) (ImplicitUnitID, error) {
		return ImplicitUnitID{}, &Error{Identity: "implicit-unit", Value: s, Reason: reason}
	}
	marker := strings.Index(s, implicitMarker)
	if marker < 0 {
		return fail("not a canonical implicit-unit serialization")
	}
	pkg, err := ParsePackageID(s[:marker])
	if err != nil {
		return ImplicitUnitID{}, err
	}
	opName := s[marker+len(implicitMarker):]
	op := ImplicitOpInvalid
	for o := ImplicitUnitOp(1); o.Valid(); o++ {
		if o.String() == opName {
			op = o
		}
	}
	if op == ImplicitOpInvalid {
		return fail("unknown implicit operation " + opName)
	}
	id, err := NewImplicitUnitID(pkg, op)
	if err != nil {
		return ImplicitUnitID{}, err
	}
	if id.String() != s {
		return fail("serialization is not canonical")
	}
	return id, nil
}

// UnitRef is the closed discriminated reference to one implementation unit:
// exactly one of a source-spanned unit or an implicit unit. It exists so
// consumers that address "a unit" (contract rules, ledgers) carry a typed
// identity, never a raw string.
type UnitRef struct {
	source   SourceUnitID
	implicit ImplicitUnitID
}

// NewSourceUnitRef references one source-spanned unit.
func NewSourceUnitRef(unit SourceUnitID) (UnitRef, error) {
	if unit.IsZero() {
		return UnitRef{}, &Error{Identity: "unit-ref", Value: "", Reason: "source unit must not be zero"}
	}
	return UnitRef{source: unit}, nil
}

// NewImplicitUnitRef references one implicit unit.
func NewImplicitUnitRef(unit ImplicitUnitID) (UnitRef, error) {
	if unit.IsZero() {
		return UnitRef{}, &Error{Identity: "unit-ref", Value: "", Reason: "implicit unit must not be zero"}
	}
	return UnitRef{implicit: unit}, nil
}

// ParseUnitRef reconstructs a unit reference from either canonical unit
// serialization, discriminating on the implicit marker.
func ParseUnitRef(s string) (UnitRef, error) {
	if strings.Contains(s, implicitMarker) {
		implicit, err := ParseImplicitUnitID(s)
		if err != nil {
			return UnitRef{}, err
		}
		return NewImplicitUnitRef(implicit)
	}
	source, err := ParseSourceUnitID(s)
	if err != nil {
		return UnitRef{}, err
	}
	return NewSourceUnitRef(source)
}

// IsZero reports whether r references nothing.
func (r UnitRef) IsZero() bool { return r == UnitRef{} }

// Source is the referenced source unit; zero when the reference is implicit.
func (r UnitRef) Source() SourceUnitID { return r.source }

// Implicit is the referenced implicit unit; zero when the reference is
// source-spanned.
func (r UnitRef) Implicit() ImplicitUnitID { return r.implicit }

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
