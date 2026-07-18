// A generic type alias (type Foo[T any] = Bar[T]) is transparent at use
// sites (Foo[int] denotes Bar[int]) but its DECLARATION carries its own
// type parameters. Canonicalizing the alias declaration must bind those
// parameters so the aliased target resolves (Bar[$#0]) instead of failing
// closed on a free parameter — the declaration identity stays the alias's
// own, the target is the transparent type. These run both instantiations
// through the differential oracle.
package translate_test

import "testing"

func TestOracleGenericAliasDeclaration(t *testing.T) {
	runOracle(t, `package fixture

type Bar[T any] struct{ v T }

func (b Bar[T]) Get() T { return b.v }

type Foo[T any] = Bar[T]

func Case() (int, string) {
	f := Foo[int]{v: 7}
	g := Foo[string]{v: "hi"}
	return f.Get(), g.Get()
}
`)
}
