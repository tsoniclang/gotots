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

// builderEmulations implements the strings.Builder external contract
// with Go-exact behavior over the byte-string carrier.
const builderEmulations = `import { goExternalRegister } from "./generated/language-abi/goextern.ts";

import { GoPanic } from "./generated/language-abi/gopanic.ts";

class BuilderEmu {
  buf = "";
  used = false;  // Go's addr self-pointer: set on first write
  stale = false; // a copy of a used builder panics on its next write
}

goExternalRegister("strings.Builder.goZero$", (): BuilderEmu => new BuilderEmu());
goExternalRegister("strings.Builder.goClone$", (b: BuilderEmu): BuilderEmu => {
  const out = new BuilderEmu();
  out.buf = b.buf;
  out.stale = b.used;
  return out;
});
goExternalRegister("strings.Builder.goSet$", (dst: BuilderEmu, src: BuilderEmu): void => {
  dst.buf = src.buf;
  dst.stale = src.used;
});
goExternalRegister("strings.Builder.WriteString", (b: BuilderEmu, s: string): readonly [bigint, undefined] => {
  if (b.stale) {
    throw new GoPanic("strings: illegal use of non-zero Builder copied by value");
  }
  b.used = true;
  b.buf += s;
  return [BigInt(s.length), undefined];
});
goExternalRegister("strings.Builder.String", (b: BuilderEmu): string => b.buf);
goExternalRegister("strings.Builder.Len", (b: BuilderEmu): bigint => BigInt(b.buf.length));
`

func TestOracleExternalTypes(t *testing.T) {
	result, err := oracle.RunEmulated(t.TempDir(), map[string]string{"fixture": `package fixture

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
`}, builderEmulations)
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}

func TestExternalTypeUnregisteredFailsClosed(t *testing.T) {
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
		t.Fatalf("translation must succeed statically (contracts fail at runtime): %v", err)
	}
	var core string
	for path, content := range generated.Files {
		if strings.Contains(path, "fixture/package.ts") {
			core = content
		}
	}
	for _, want := range []string{
		`goext$.goExternalCall("sync.Mutex.goZero$", [])`,
		`goext$.goExternalCall("sync.Mutex.Lock", `,
		`goext$.goExternalCall("sync.Mutex.Unlock", `,
	} {
		if !strings.Contains(core, want) {
			t.Errorf("generated module missing contract dispatch %q:\n%s", want, core)
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
