// External-type contract tests: opaque handles with emulated zero,
// clone, set, and method contracts — strings.Builder as the canonical
// mutable external value — plus runtime fail-closed behavior for
// unregistered contracts and fail-closed diagnostics for operations
// outside the reviewed external surface.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// builderImplementation replaces the strings stub module with a
// reviewed strings.Builder contract implementation exporting exactly
// the generated stub symbols, including Go's real use-after-copy panic.
var builderImplementation = map[string]string{
	"strings": `import { GoPanic } from "../../language-abi/gopanic.js";
import { type GoExtern } from "../../language-abi/goextern.js";

class BuilderEmu {
  buf = "";
  used = false;  // Go's addr self-pointer: set on first write
  stale = false; // a copy of a used builder panics on its next write
}

type Handle = GoExtern<"strings.Builder">;

export function Builder$goZero$(): Handle {
  return new BuilderEmu() as Handle;
}

export function Builder$goClone$(v: Handle): Handle {
  const b = v as unknown as BuilderEmu;
  const out = new BuilderEmu();
  out.buf = b.buf;
  out.stale = b.used;
  return out as Handle;
}

export function Builder$goSet$(dst: Handle, src: Handle): void {
  const d = dst as unknown as BuilderEmu;
  const s = src as unknown as BuilderEmu;
  d.buf = s.buf;
  d.stale = s.used;
}

export function Builder$WriteString(recv: Handle, s: string): readonly [bigint, undefined] {
  const b = recv as unknown as BuilderEmu;
  if (b.stale) {
    throw new GoPanic("strings: illegal use of non-zero Builder copied by value");
  }
  b.used = true;
  b.buf += s;
  return [BigInt(s.length), undefined];
}

export function Builder$String(recv: Handle): string {
  return (recv as unknown as BuilderEmu).buf;
}

export function Builder$Len(recv: Handle): bigint {
  return BigInt((recv as unknown as BuilderEmu).buf.length);
}
`,
}

func TestOracleExternalTypes(t *testing.T) {
	result, err := oracle.RunAssembled(t.TempDir(), map[string]string{"fixture": `package fixture

import "strings"

func BuildString() (string, int) {
	var sb strings.Builder
	sb.WriteString("hello")
	sb.WriteString(", world")
	return sb.String(), sb.Len()
}

func PointerReceiverThroughPointer() string {
	sb := &strings.Builder{}
	sb.WriteString("via pointer")
	return sb.String()
}

func CopySemantics() (string, string) {
	var a strings.Builder
	a.WriteString("base")
	b := a
	b.WriteString("+more")
	return a.String(), b.String()
}

func CopyOfZeroBuilderIsIndependent() (string, string) {
	var a strings.Builder
	b := a
	a.WriteString("left")
	b.WriteString("right")
	return a.String(), b.String()
}

func HandleThroughField() string {
	var sb strings.Builder
	p := &sb
	p.WriteString("x")
	sb.WriteString("y")
	return sb.String()
}
`}, builderImplementation)
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}

func TestExternalTypeStaticStubs(t *testing.T) {
	generated, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": `package fixture

import "sync"

func Case() int {
	var mu sync.Mutex
	mu.Lock()
	mu.Unlock()
	return 1
}
`})
	if err != nil {
		t.Fatalf("translation must succeed statically (stubs fail closed at runtime): %v", err)
	}
	var core, stub string
	for path, content := range generated.Files {
		if strings.Contains(path, "fixture/package.ts") {
			core = content
		}
		if path == "external-stubs/sync/package.ts" {
			stub = content
		}
	}
	// The generated module binds statically selected typed stub exports;
	// no registry, lookup, or result recovery cast exists anywhere.
	for _, want := range []string{
		"Mutex$goZero$()",
		"Mutex$Lock(",
		"Mutex$Unlock(",
	} {
		if !strings.Contains(core, want) {
			t.Errorf("generated module missing static stub call %q:\n%s", want, core)
		}
	}
	if strings.Contains(core, "goExternalCall") {
		t.Errorf("generated module still routes through a registry:\n%s", core)
	}
	for _, want := range []string{
		"export function Mutex$goZero$(",
		"export function Mutex$Lock(",
		"goext$.goExternalUnimplemented(\"sync.Mutex.Lock\");",
	} {
		if !strings.Contains(stub, want) {
			t.Errorf("sync stub module missing %q:\n%s", want, stub)
		}
	}
}

func TestExternalValueFailClosedShapes(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		mention string
	}{
		{
			name: "field access on external value",
			source: `package fixture
import "go/token"
func Case() int { var p token.Position; return p.Line }
`,
			mention: "field access on",
		},
		{
			name: "store into slice of external values",
			source: `package fixture
import "time"
func Case() int {
	s := make([]time.Time, 2)
	var t time.Time
	s[0] = t
	return len(s)
}
`,
			mention: "store into a slice of external values",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": c.source})
			if err == nil {
				t.Fatalf("expected a fail-closed diagnostic mentioning %q", c.mention)
			}
			if !strings.Contains(err.Error(), c.mention) {
				t.Fatalf("expected diagnostic to mention %q, got: %v", c.mention, err)
			}
		})
	}
}

// crossPackageMethodImplementations provides two external types from
// DIFFERENT packages that share a method spelling (Len): the canonical
// keying gives each its own obligation member and its own emitted symbol
// (Builder$Len vs Buffer$Len), so neither overwrites the other and both
// dispatch to their own reviewed contract.
var crossPackageMethodImplementations = map[string]string{
	"strings": `import { type GoExtern } from "../../language-abi/goextern.js";
class BuilderEmu { buf = ""; }
type Handle = GoExtern<"strings.Builder">;
export function Builder$goZero$(): Handle { return new BuilderEmu() as Handle; }
export function Builder$goClone$(v: Handle): Handle { const b = v as unknown as BuilderEmu; const o = new BuilderEmu(); o.buf = b.buf; return o as Handle; }
export function Builder$goSet$(dst: Handle, src: Handle): void { (dst as unknown as BuilderEmu).buf = (src as unknown as BuilderEmu).buf; }
export function Builder$WriteString(recv: Handle, s: string): readonly [bigint, undefined] { const b = recv as unknown as BuilderEmu; b.buf += s; return [BigInt(s.length), undefined]; }
export function Builder$Len(recv: Handle): bigint { return BigInt((recv as unknown as BuilderEmu).buf.length); }
`,
	"bytes": `import { type GoExtern } from "../../language-abi/goextern.js";
class BufferEmu { buf = ""; }
type Handle = GoExtern<"bytes.Buffer">;
export function Buffer$goZero$(): Handle { return new BufferEmu() as Handle; }
export function Buffer$goClone$(v: Handle): Handle { const b = v as unknown as BufferEmu; const o = new BufferEmu(); o.buf = b.buf; return o as Handle; }
export function Buffer$goSet$(dst: Handle, src: Handle): void { (dst as unknown as BufferEmu).buf = (src as unknown as BufferEmu).buf; }
export function Buffer$WriteString(recv: Handle, s: string): readonly [bigint, undefined] { const b = recv as unknown as BufferEmu; b.buf += s; return [BigInt(s.length), undefined]; }
export function Buffer$Len(recv: Handle): bigint { return BigInt((recv as unknown as BufferEmu).buf.length); }
`,
}

func TestOracleCrossPackageSameMethodName(t *testing.T) {
	result, err := oracle.RunAssembled(t.TempDir(), map[string]string{"fixture": `package fixture

import (
	"bytes"
	"strings"
)

// Two external types from different packages, each with a Len method.
func BothLen() (int, int) {
	var sb strings.Builder
	sb.WriteString("abc")
	var bb bytes.Buffer
	bb.WriteString("de")
	return sb.Len(), bb.Len()
}
`}, crossPackageMethodImplementations)
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}
