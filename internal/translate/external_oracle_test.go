package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// externalEmulations registers Go-exact behavior for the external
// contracts the fixtures reference, mirroring the product's
// hand-maintained emulation layer.
// Implementation modules assembled in place of the generated stubs:
// each exports exactly the stub's typed symbols with Go-exact behavior,
// mirroring the product's hand-maintained implementation layer.
var externalImplementations = map[string]string{
	"strings": `import { type GoSliceValue } from "../../language-abi/goslice.js";

export function HasPrefix(s: string, prefix: string): boolean {
  return s.startsWith(prefix);
}

export function Repeat(s: string, count: bigint): string {
  return s.repeat(Number(count));
}
`,
	"strconv": `export function Itoa(i: bigint): string {
  return String(i);
}
`,
	"os": `import { goSliceFrom, type GoSliceValue } from "../../language-abi/goslice.js";

const Args: GoSliceValue<string> = goSliceFrom(["fixture"]);
export function Args$get$(): GoSliceValue<string> {
  return Args;
}
`,
	"image": `import { type GoExtern } from "../../language-abi/goextern.js";

type Handle = GoExtern<"image.Point">;
type Rep = { X: bigint; Y: bigint };

export function Point$goZero$(): Handle {
  return { X: 0n, Y: 0n } as Rep as Handle;
}
export function Point$goClone$(v: Handle | undefined): Handle {
  const rep = v as Rep | undefined;
  return { X: rep === undefined ? 0n : rep.X, Y: rep === undefined ? 0n : rep.Y } as Rep as Handle;
}
export function Point$goSet$(dst: Handle | undefined, src: Handle | undefined): void {
  const d = dst as Rep | undefined;
  const s = src as Rep | undefined;
  if (d !== undefined && s !== undefined) {
    d.X = s.X;
    d.Y = s.Y;
  }
}
export function Point$lit$X$Y$(X: bigint, Y: bigint): Handle {
  return { X, Y } as Rep as Handle;
}
export function Point$get$X$(v: Handle | undefined): bigint {
  return (v as Rep).X;
}
export function Point$get$Y$(v: Handle | undefined): bigint {
  return (v as Rep).Y;
}
export function Pt(x: bigint, y: bigint): Handle {
  return { X: x, Y: y } as Rep as Handle;
}
export function Point$eq$(a: Handle | undefined, b: Handle | undefined): boolean {
  const ra = a as Rep | undefined;
  const rb = b as Rep | undefined;
  if (ra === undefined || rb === undefined) {
    return ra === rb;
  }
  return ra.X === rb.X && ra.Y === rb.Y;
}
`,
	"sync": `import { type GoExtern } from "../../language-abi/goextern.js";

// The reviewed sync.Pool implementation: Get() = New() and Put() =
// no-op is EXACT (the Pool spec permits dropping every item at any
// time). The representation is a mutable record behind the phantom
// handle brand; Get's return-position type parameter resolves from the
// caller's contextual type (the declared union), never a runtime cast
// surface.
type Handle = GoExtern<"sync.Pool">;
type Rep = { New: (() => unknown) | undefined };

export function Pool$goZero$(): Handle {
  return { New: undefined } as Rep as Handle;
}
export function Pool$goClone$(v: Handle | undefined): Handle {
  const rep = v as Rep | undefined;
  return { New: rep === undefined ? undefined : rep.New } as Rep as Handle;
}
export function Pool$goSet$(dst: Handle | undefined, src: Handle | undefined): void {
  const d = dst as Rep | undefined;
  const s = src as Rep | undefined;
  if (d !== undefined) {
    d.New = s === undefined ? undefined : s.New;
  }
}
export function Pool$lit$New$<T>(New: () => T): Handle {
  return { New } as Rep as Handle;
}
export function Pool$Get<T>(p: unknown): T {
  const cell = p as { v?: Handle | undefined } | Handle | undefined;
  const handle = cell !== undefined && typeof cell === "object" && "v" in (cell as object)
    ? (cell as { v?: Handle | undefined }).v
    : (cell as Handle | undefined);
  const rep = handle as Rep | undefined;
  if (rep === undefined || rep.New === undefined) {
    throw new Error("sync.Pool: Get on a pool without New");
  }
  return (rep.New as () => T)();
}
export function Pool$Put(p: unknown, v: unknown): void {
  void p;
  void v;
}
`,
	"slices": `import { goSliceLen, goSliceGet, type GoSliceValue } from "../../language-abi/goslice.js";

export function Contains<S, E>(
  s: GoSliceValue<E>, v: E,
  zero$S: () => S, zero$E: () => E,
  eq$S: (a: S, b: S) => boolean, eq$E: (a: E, b: E) => boolean,
  clone$S: (v: S) => S, clone$E: (v: E) => E,
  set$S: ((d: S, s: S) => void) | undefined, set$E: ((d: E, s: E) => void) | undefined,
): boolean {
  void zero$S; void zero$E; void eq$S; void clone$S; void clone$E; void set$S; void set$E;
  const length = Number(goSliceLen(s));
  for (let index = 0; index < length; index++) {
    if (eq$E(goSliceGet(s, BigInt(index)), v)) {
      return true;
    }
  }
  return false;
}
`,
}

func runExternalOracle(t *testing.T, fixtureSource string) {
	t.Helper()
	result, err := oracle.RunAssembled(t.TempDir(), map[string]string{"fixture": fixtureSource}, externalImplementations)
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}

func TestOracleExternalContracts(t *testing.T) {
	runExternalOracle(t, `package fixture

import (
	"strconv"
	"strings"
)

func ExternalCalls() (bool, bool, string, string) {
	return strings.HasPrefix("gopher", "go"),
		strings.HasPrefix("gopher", "x"),
		strconv.Itoa(-42),
		strings.Repeat("ab", 3)
}

func ExternalInExpressions() string {
	parts := ""
	for i := 0; i < 3; i++ {
		if strings.HasPrefix("abc", "a") {
			parts = parts + strconv.Itoa(i)
		}
	}
	return parts
}
`)
}

func TestOracleExternalGenericContracts(t *testing.T) {
	runExternalOracle(t, `package fixture

import "slices"

func ExternalGeneric() (bool, bool, bool) {
	var empty []int
	return slices.Contains([]int{1, 2, 3}, 2),
		slices.Contains([]string{"a"}, "b"),
		slices.Contains(empty, 7)
}
`)
}

func TestExternalStubShape(t *testing.T) {
	generated, _, err := oracle.Translate(t.TempDir(), map[string]string{
		"fixture": `package fixture

import "strings"

func Use() bool { return strings.HasPrefix("a", "b") }
`,
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	stub, ok := generated.Files["external-stubs/strings/package.ts"]
	if !ok {
		t.Fatalf("strings stub module missing; files: %d", len(generated.Files))
	}
	if !strings.Contains(stub, "export function HasPrefix(s: string, prefix: string): boolean {") {
		t.Fatalf("stub lacks the typed signature:\n%s", stub)
	}
	if !strings.Contains(stub, `goext$.goExternalUnimplemented("strings.HasPrefix");`) {
		t.Fatalf("stub does not fail closed statically:\n%s", stub)
	}
	if strings.Contains(stub, "goExternalCall") || strings.Contains(stub, "goExternalRegister") {
		t.Fatalf("stub still references a runtime registry:\n%s", stub)
	}
	if generated.Ownership["external-stubs/strings/package.ts"] != "generated-external-contracts" {
		t.Fatalf("stub ownership root missing")
	}
	fixtureModule := generated.Files["core/oracle.fixture/fixture/package.ts"]
	if !strings.Contains(fixtureModule, `from "../../../external-stubs/strings/package.js";`) {
		t.Fatalf("fixture module lacks the stub import:\n%s", fixtureModule)
	}
}

func TestOracleExternalVarRead(t *testing.T) {
	// The externVar class: an external package VARIABLE read through its
	// typed stub. Only a deterministic derived value is compared (os.Args
	// is non-empty in both hosts).
	runExternalOracle(t, `package fixture

import "os"

func ExternVarRead() bool {
	return len(os.Args) > 0
}
`)
}

func TestOracleExternPoolLiteral(t *testing.T) {
	// The sync.Pool class: a keyed composite literal of an external
	// struct lowers to its reviewed constructor stub; Get() = New() and
	// Put() = no-op is semantically EXACT (the Pool spec permits dropping
	// every item), so the assembled implementation reproduces Go.
	runExternalOracle(t, `package fixture

import "sync"

var pool = sync.Pool{
	New: func() any {
		return 7
	},
}

func ExternPoolLiteral() int {
	v := pool.Get().(int)
	pool.Put(v)
	w := pool.Get().(int)
	return v*10 + w
}
`)
}

func TestOracleExternalGenericValueCopyBinding(t *testing.T) {
	// The checker slices.Contains-with-struct shape: an external generic
	// call bound to a value-copy carrier — the stub contract carries the
	// factory quadruple, so the reviewed implementation reproduces Go's
	// per-binding equality and copy semantics exactly.
	runExternalOracle(t, `package fixture

import "slices"

type key struct {
	a int
	b string
}

func ExternalGenericValueCopyBinding() int {
	items := []key{{1, "x"}, {2, "y"}}
	total := 0
	if slices.Contains(items, key{2, "y"}) {
		total += 100
	}
	if slices.Contains(items, key{2, "z"}) {
		total += 10000
	}
	return total
}
`)
}

func TestOracleExternFuncReference(t *testing.T) {
	// A NON-generic external function referenced as a first-class value:
	// the typed stub export IS the value; the assembled implementation
	// supplies the behavior.
	runExternalOracle(t, `package fixture

import "strings"

func apply(f func(string, string) bool, a, b string) bool {
	return f(a, b)
}

func ExternFuncReference() int {
	check := strings.HasPrefix
	total := 0
	if check("gopher", "go") {
		total += 10
	}
	if apply(strings.HasPrefix, "gopher", "x") {
		total += 1000
	}
	return total
}
`)
}

func TestOracleExternOwnedUnderlyingConversions(t *testing.T) {
	// The checker CacheHashKey/xxh3.Uint128 shape: an OWNED named struct
	// over an EXTERNAL struct underlying, converted both ways — to the
	// class via per-field read stubs, back via the keyed-literal
	// constructor — plus a field read straight off the handle.
	runExternalOracle(t, `package fixture

import "image"

type key image.Point

func (k key) Sum() int {
	return k.X + k.Y
}

func ExternOwnedUnderlyingConversions() int {
	p := image.Pt(3, 4)
	k := key(p)
	back := image.Point(k)
	same := 0
	if p == image.Pt(3, 4) {
		same = 1
	}
	if p == image.Pt(9, 9) {
		same += 100
	}
	return k.Sum()*1000 + back.X*100 + p.Y*10 + same
}
`)
}
