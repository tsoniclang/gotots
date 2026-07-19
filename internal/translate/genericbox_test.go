// An instantiated OWNED generic type is a CONCRETE dynamic type: the
// closed instantiation evidence enumerates it as a composite-branded
// union member with an inline vtable over the generated generic
// functions, so boxing one is exact (proved by differential oracle).
// EXTERNAL instantiated generics stay fail-closed: their stubs are not
// per-instance, so no vtable surface exists.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// An owned generic struct instance boxed into an owned interface: exact.
func TestOwnedInstantiatedGenericBoxExact(t *testing.T) {
	runOracle(t, `package fixture

type Box[T any] struct{ v T }

func (b Box[T]) Get() T { return b.v }

type IntGetter interface{ Get() int }

func Case() int {
	var g IntGetter = Box[int]{v: 5}
	return g.Get()
}
`)
}

// A pointer to an external instantiated generic (atomic.Pointer[int])
// boxed into the empty interface stays fail-closed: no per-instance
// stub surface exists.
func TestExternalInstantiatedGenericPointerBoxWithheld(t *testing.T) {
	_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": `package fixture

import "sync/atomic"

func Case() any {
	var p atomic.Pointer[int]
	return &p
}
`})
	if err == nil {
		t.Fatal("expected the instantiated-external-generic box to be withheld")
	}
	if !strings.Contains(err.Error(), "instantiated external generic type") {
		t.Fatalf("expected the external-instantiation block, got: %v", err)
	}
}
