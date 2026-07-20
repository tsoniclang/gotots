// Package implid is the ONE owner of the ADR-0010 implementation
// identity grammar:
//
//	ImplementationID = SourceDeclarationID "/" specializationKey
//
// Every producer constructs identities here and every consumer parses
// them here. Manual string concatenation or splitting of an
// implementation identity anywhere else is a defect; the grammar has
// exactly one implementation.
package implid

import (
	"fmt"
	"strings"
)

// ID is a parsed implementation identity.
type ID struct {
	// Source is the canonical source-declaration identity (contains
	// "::" and may contain "/" inside its package path).
	Source string
	// Key is the specialization key: "default", a binding-family key
	// such as "map-key-encoded" or "pointer-cell", or a typed exception
	// key. Never empty; never contains "/" or ":".
	Key string
}

// New validates and constructs an identity. A source without "::" or a
// key containing separator runes is malformed and fails closed.
func New(source, key string) (ID, error) {
	if !strings.Contains(source, "::") {
		return ID{}, fmt.Errorf("implementation source %q is not a canonical declaration identity", source)
	}
	if key == "" || strings.ContainsAny(key, "/:") {
		return ID{}, fmt.Errorf("implementation key %q is malformed", key)
	}
	return ID{Source: source, Key: key}, nil
}

// MustNew constructs an identity whose inputs are construction
// invariants of the caller (the emitter's IDs are builder-validated);
// violation is a compiler defect and panics, never a silent fallback.
func MustNew(source, key string) ID {
	id, err := New(source, key)
	if err != nil {
		panic("implid: " + err.Error())
	}
	return id
}

// String spells the canonical form.
func (id ID) String() string { return id.Source + "/" + id.Key }

// Parse recovers an identity from its canonical spelling: the key is
// the segment after the LAST "/", which must fall after the last "::"
// (package paths before "::" contain "/"; declaration names never do).
func Parse(spelling string) (ID, error) {
	slash := strings.LastIndex(spelling, "/")
	if slash < 0 {
		return ID{}, fmt.Errorf("implementation id %q has no specialization key", spelling)
	}
	colons := strings.LastIndex(spelling, "::")
	if colons < 0 || slash < colons {
		return ID{}, fmt.Errorf("implementation id %q is not source::.../key shaped", spelling)
	}
	return New(spelling[:slash], spelling[slash+1:])
}
