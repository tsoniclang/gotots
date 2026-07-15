package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// externalEmulations registers Go-exact behavior for the external
// contracts the fixtures reference, mirroring the product's
// hand-maintained emulation layer.
const externalEmulations = `import { goExternalRegister } from "./generated/language-abi/goextern.ts";
import { goSliceLen, goSliceGet, type GoSliceValue } from "./generated/language-abi/goslice.ts";

goExternalRegister("strings.HasPrefix", (s: string, prefix: string): boolean => s.startsWith(prefix));
goExternalRegister("strings.Repeat", (s: string, count: bigint): string => s.repeat(Number(count)));
goExternalRegister("strconv.Itoa", (i: bigint): string => String(i));
goExternalRegister("slices.Contains", (s: GoSliceValue<unknown>, v: unknown): boolean => {
  const length = Number(goSliceLen(s));
  for (let index = 0; index < length; index++) {
    if (goSliceGet(s, BigInt(index)) === v) {
      return true;
    }
  }
  return false;
});
`

func runExternalOracle(t *testing.T, fixtureSource string) {
	t.Helper()
	result, err := oracle.RunEmulated(t.TempDir(), map[string]string{"fixture": fixtureSource}, externalEmulations)
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
	if !strings.Contains(stub, `goext$.goExternalCall("strings.HasPrefix", [s, prefix])`) {
		t.Fatalf("stub lacks the registry delegation:\n%s", stub)
	}
	if generated.Ownership["external-stubs/strings/package.ts"] != "generated-external-contracts" {
		t.Fatalf("stub ownership root missing")
	}
	fixtureModule := generated.Files["core/oracle.fixture/fixture/package.ts"]
	if !strings.Contains(fixtureModule, `from "../../../external-stubs/strings/package.js";`) {
		t.Fatalf("fixture module lacks the stub import:\n%s", fixtureModule)
	}
}
