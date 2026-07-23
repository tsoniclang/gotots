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
		owner, err := NewModuleOwner(module)
		if err != nil {
			return Owner{}, err
		}
		if owner.String() != s {
			return Owner{}, &Error{
				Identity: "owner",
				Value:    s,
				Reason:   "serialization is not canonical",
			}
		}
		return owner, nil
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

// ParseDefinitionID reconstructs either a source construct-root definition or
// a typed implicit definition from its canonical serialization.
func ParseDefinitionID(s string) (DefinitionID, error) {
	fail := func(reason string) (DefinitionID, error) {
		return DefinitionID{}, &Error{Identity: "definition", Value: s, Reason: reason}
	}
	if marker := strings.LastIndex(s, "#definition/"); marker >= 0 {
		pkg, err := ParsePackageID(s[:marker])
		if err != nil {
			return DefinitionID{}, err
		}
		name := s[marker+len("#definition/"):]
		op := ImplicitDefinitionInvalid
		for candidate := ImplicitDefinitionOp(1); candidate.Valid(); candidate++ {
			if candidate.String() == name {
				op = candidate
				break
			}
		}
		if !op.Valid() {
			return fail("unknown implicit definition operation " + name)
		}
		id, err := NewImplicitDefinitionID(pkg, op)
		if err != nil {
			return DefinitionID{}, err
		}
		if id.String() != s {
			return fail("serialization is not canonical")
		}
		return id, nil
	}
	if marker := strings.LastIndex(s, "#synthetic/"); marker >= 0 {
		pkg, err := ParsePackageID(s[:marker])
		if err != nil {
			return DefinitionID{}, err
		}
		payload := s[marker+len("#synthetic/"):]
		slash := strings.Index(payload, "/")
		if slash < 0 {
			return fail("synthetic definition lacks role/name separation")
		}
		roleName, name := payload[:slash], payload[slash+1:]
		role := SyntheticDefinitionInvalid
		for candidate := SyntheticDefinitionRole(1); candidate.Valid(); candidate++ {
			if candidate.String() == roleName {
				role = candidate
				break
			}
		}
		id, err := NewSyntheticDefinitionID(pkg, role, name)
		if err != nil {
			return DefinitionID{}, err
		}
		if id.String() != s {
			return fail("serialization is not canonical")
		}
		return id, nil
	}
	marker := strings.LastIndex(s, "/D")
	if marker < 0 {
		return fail("not a canonical definition serialization")
	}
	root, err := ParseOccurrenceID(s[:marker])
	if err != nil {
		return DefinitionID{}, err
	}
	kindValue, err := strconv.ParseUint(s[marker+2:], 10, 8)
	if err != nil {
		return fail("malformed definition kind")
	}
	id, err := NewSourceDefinitionID(root, DefinitionKind(kindValue))
	if err != nil {
		return DefinitionID{}, err
	}
	if id.String() != s {
		return fail("serialization is not canonical")
	}
	return id, nil
}
