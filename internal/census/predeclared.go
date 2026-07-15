package census

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// reviewedPredeclared is the reviewed contract with the Go predeclared
// universe. The pinned toolchain — not this list — is authoritative: if a
// future toolchain exposes a predeclared object absent from this list,
// the census fails until the object is reviewed and its semantics are
// classified.
var reviewedPredeclared = map[string]bool{
	// types
	"bool": true, "byte": true, "comparable": true,
	"complex64": true, "complex128": true, "error": true,
	"float32": true, "float64": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"rune": true, "string": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "any": true,
	// constants and zero value
	"true": true, "false": true, "iota": true, "nil": true,
	// functions
	"append": true, "cap": true, "clear": true, "close": true,
	"complex": true, "copy": true, "delete": true, "imag": true,
	"len": true, "make": true, "max": true, "min": true, "new": true,
	"panic": true, "print": true, "println": true, "real": true,
	"recover": true,
}

// checkPredeclaredUniverse compares the pinned toolchain's actual
// predeclared universe against the reviewed contract. It returns the
// complete sorted universe as evidence and fails closed on any object
// this census has not been reviewed against.
func checkPredeclaredUniverse() ([]string, error) {
	names := types.Universe.Names()
	sort.Strings(names)
	var unreviewed []string
	for _, name := range names {
		if !reviewedPredeclared[name] {
			unreviewed = append(unreviewed, name)
		}
	}
	if len(unreviewed) > 0 {
		return nil, fmt.Errorf("GOTOTS_UNSUPPORTED_PREDECLARED_OBJECT: the toolchain's predeclared universe contains unreviewed objects: %s",
			strings.Join(unreviewed, ", "))
	}
	return names, nil
}
