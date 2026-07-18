// An instantiated generic type — owned or external — has per-instantiation
// dynamic identity: each closed instance (Box[int] vs Box[string],
// atomic.Pointer[int] vs atomic.Pointer[string]) is a distinct interface
// union member. The universe does not yet enumerate closed instances, so
// boxing one must fail closed rather than emit a box that belongs to no
// union (a strict tsc failure). These pin that boundary on every path that
// reaches rttiFor: the owned named value, and the external pointer.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func assertGenericBoxWithheld(t *testing.T, fixtureSource string) {
	t.Helper()
	_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": fixtureSource})
	if err == nil {
		t.Fatal("expected the instantiated-generic box to be withheld")
	}
	if !strings.Contains(err.Error(), "instantiated generic type") {
		t.Fatalf("expected the instantiated-generic block, got: %v", err)
	}
}

// An owned generic struct instance boxed into an owned interface.
func TestOwnedInstantiatedGenericBoxWithheld(t *testing.T) {
	assertGenericBoxWithheld(t, `package fixture

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
// boxed into the empty interface. Before the block this reached
// compositeRtti and produced a box absent from every union.
func TestExternalInstantiatedGenericPointerBoxWithheld(t *testing.T) {
	assertGenericBoxWithheld(t, `package fixture

import "sync/atomic"

func Case() any {
	var p atomic.Pointer[int]
	return &p
}
`)
}
