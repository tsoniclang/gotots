// Unified store-plan differential proof (reviewer Defect 2): a native
// for-header post clause must render a target's store IDENTICALLY to the
// normalized statement store, so an in-place carrier (struct, fixed array,
// external value) stores in place and every existing pointer to it stays
// valid — never a rebind that installs a fresh object. Both loop shapes go
// through the single varStoreExpr, so the two can no longer diverge.
package translate_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// urlImpl is a minimal reviewed implementation of net/url's external value
// contract: goSet$ overwrites the target in place (Object.assign), so a
// pointer taken before the store still observes it.
const urlImpl = `import type { GoExtern } from "../../../language-abi/goextern.js";

type URL = GoExtern<"net/url.URL">;
export function URL$goZero$(): URL { return {} as URL; }
export function URL$goClone$(value: URL): URL { return { ...value } as URL; }
export function URL$goSet$(target: URL, value: URL): void { Object.assign(target, value); }
`

func TestOracleExternalValueForPostStoresInPlace(t *testing.T) {
	// The reviewer's headline case: a package-level external value assigned
	// in a native for-post. A rebind (`global = next`) would leave the
	// earlier &global pointing at the old object; the in-place set stub keeps
	// pointer == &global true, matching Go.
	result, err := oracle.RunAssembled(t.TempDir(),
		map[string]string{"fixture": `package fixture

import "net/url"

var global url.URL

func ExternalGlobalPost() bool {
	pointer := &global
	var next url.URL
	for i := 0; i < 1; global = next {
		i++
	}
	return pointer == &global
}
`},
		map[string]string{"net/url": urlImpl})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("external in-place store diverged:\n--- go ---\n%s--- generated ---\n%s",
			result.GoOutput, result.TSOutput)
	}
}

func TestOracleStructValueForPostStoresInPlace(t *testing.T) {
	// A struct-valued variable stored in a native for-post overwrites in
	// place, so a pointer taken to it observes the final value.
	runOracle(t, `package fixture

type box struct{ n int }

func StructForPostInPlace() int {
	var s box
	p := &s
	xs := []box{{n: 1}, {n: 2}, {n: 3}}
	for i := 0; i < 3; s = xs[i] {
		i++
	}
	return p.n
}
`)
}
