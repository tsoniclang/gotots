// Interface equality over external dynamic types matches Go's runtime
// behavior by comparability, read from the emitted equality function:
//
//   - an UNCOMPARABLE external type (a slice/map/func carrier such as
//     net.IP, or a struct with such a field) PANICS in Go; the generated
//     code emits Go's exact "comparing uncomparable type" panic, never a
//     silent === on the opaque handle.
//   - a COMPARABLE external struct compares field-wise in Go, but its TS
//     value is an opaque handle: equality fails closed loudly with the
//     unimplemented marker (the stub-provided-equality completion contract
//     is future work), never a wrong === on distinct-but-equal instances.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func fixturePackageSource(t *testing.T, fixture string) string {
	t.Helper()
	gen, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": fixture})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	var combined strings.Builder
	for path, src := range gen.Files {
		if strings.HasSuffix(path, "oracle.fixture/fixture/package.ts") ||
			strings.HasSuffix(path, "interfaces/package.ts") {
			combined.WriteString(src)
		}
	}
	if combined.Len() == 0 {
		t.Fatal("fixture package.ts not found in generated output")
	}
	return combined.String()
}

func TestUncomparableExternalInterfaceEqPanics(t *testing.T) {
	src := fixturePackageSource(t, `package fixture

import "net"

type Stringy interface{ String() string }

func Case() bool {
	var a Stringy = net.IP{1, 2, 3, 4}
	var b Stringy = net.IP{1, 2, 3, 4}
	return a == b
}
`)
	if !strings.Contains(src, "goPanicUncomparable") {
		t.Errorf("expected the uncomparable-external comparison to emit goPanicUncomparable:\n%s", src)
	}
}

func TestComparableExternalStructInterfaceEqFailsClosed(t *testing.T) {
	src := fixturePackageSource(t, `package fixture

import "time"

type HasZero interface{ IsZero() bool }

func Case() bool {
	var a HasZero = time.Time{}
	var b HasZero = time.Time{}
	return a == b
}
`)
	if !strings.Contains(src, "goPanicExternalEq") {
		t.Errorf("expected the comparable-external-struct comparison to fail closed via goPanicExternalEq:\n%s", src)
	}
}
