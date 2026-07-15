package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// helperSource is a co-translated unit package exercised from the fixture
// package: an exported struct with pointer-receiver methods, a
// constructor, plain functions, and an exported constant.
const helperSource = `package helper

const Answer = 42

type Counter struct {
	Total int
	Label string
}

func (c *Counter) Add(delta int) int {
	c.Total = c.Total + delta
	return c.Total
}

func (c *Counter) Rename(label string) {
	c.Label = label
}

func NewCounter(label string) *Counter {
	return &Counter{Total: 0, Label: label}
}

func Scale(x int, factor int) int {
	return x * factor
}
`

func runCrossPackageOracle(t *testing.T, fixtureSource string) *oracle.Result {
	t.Helper()
	result, err := oracle.Run(t.TempDir(), map[string]string{
		"fixture": fixtureSource,
		"helper":  helperSource,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch across %d cases:\n--- go ---\n%s--- generated ---\n%s",
			result.Cases, result.GoOutput, result.TSOutput)
	}
	return result
}

func TestOracleCrossPackage(t *testing.T) {
	runCrossPackageOracle(t, `package fixture

import "oracle.fixture/helper"

func CrossCall() int {
	return helper.Scale(6, 7)
}

func CrossConst() int {
	return helper.Answer
}

func CrossLiteralAndFields() (int, string) {
	c := &helper.Counter{Total: 40, Label: "answer"}
	c.Total++
	c.Total = c.Total + 1
	return c.Total, c.Label
}

func CrossMethods() (int, string) {
	c := helper.NewCounter("start")
	c.Add(3)
	c.Rename("renamed")
	return c.Add(4), c.Label
}

func CrossNilComparison() (bool, bool) {
	var absent *helper.Counter
	present := helper.NewCounter("x")
	return absent == nil, present != nil
}

func CrossNilFieldPanic() int {
	var c *helper.Counter
	return c.Total
}

func CrossNilMethodPanic() int {
	var c *helper.Counter
	return c.Add(1)
}

// Wrapper holds a cross-package pointer field inside a local struct.
type Wrapper struct {
	Inner *helper.Counter
}

func CrossPointerField() int {
	w := &Wrapper{Inner: helper.NewCounter("w")}
	w.Inner = helper.NewCounter("swapped")
	return w.Inner.Add(5)
}

func CrossPointerIdentity() bool {
	c := helper.NewCounter("same")
	d := c
	return c == d
}

// ShadowedAliasCandidate declares a local variable spelled exactly like
// the helper package name; generated import aliases carry a "$" suffix
// that no Go identifier can spell, so the reference stays unambiguous.
func ShadowedAliasCandidate() (int, int) {
	helper := 5
	return helper, 6
}

func CrossInDefer() int {
	c := helper.NewCounter("deferred")
	defer c.Add(10)
	return c.Add(1)
}
`)
}

func TestOracleCrossPackageSlicesAndMaps(t *testing.T) {
	runCrossPackageOracle(t, `package fixture

import "oracle.fixture/helper"

func CrossSliceOfPointers() (int, int) {
	counters := []*helper.Counter{helper.NewCounter("a"), nil}
	counters = append(counters, helper.NewCounter("c"))
	total := 0
	for _, c := range counters {
		if c != nil {
			total = total + c.Add(1)
		}
	}
	return total, len(counters)
}

func CrossMapOfPointers() (int, bool) {
	byName := map[string]*helper.Counter{"a": helper.NewCounter("a")}
	byName["b"] = helper.NewCounter("b")
	missing, ok := byName["absent"]
	if missing != nil {
		return -1, ok
	}
	return byName["a"].Add(2) + byName["b"].Add(3), ok
}
`)
}

func TestCrossPackageFailClosed(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		code    string
		mention string
	}{
		{
			name: "external call whose signature cannot be typed",
			source: `package fixture

import "os"

func Case() bool {
	f, err := os.Open("x")
	return f != nil && err == nil
}
`,
			code:    "GOTOTS_UNSUPPORTED_EXPRESSION",
			mention: "call outside the translated unit",
		},
		{
			name: "pointer type outside the unit",
			source: `package fixture

import "bytes"

func Case(b *bytes.Buffer) bool { return b == nil }
`,
			code:    "GOTOTS_UNSUPPORTED_TYPE",
			mention: "pointer to type outside the translated unit",
		},
		{
			name: "blank import",
			source: `package fixture

import _ "oracle.fixture/helper"

func Case() int { return 1 }
`,
			code:    "GOTOTS_UNSUPPORTED_DECLARATION",
			mention: "blank import",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := oracle.Translate(t.TempDir(), map[string]string{
				"fixture": c.source,
				"helper":  helperSource,
			})
			if err == nil {
				t.Fatalf("expected fail-closed diagnostic %s", c.code)
			}
			if !strings.Contains(err.Error(), c.code) {
				t.Fatalf("expected diagnostic code %s, got: %v", c.code, err)
			}
			if !strings.Contains(err.Error(), c.mention) {
				t.Fatalf("expected diagnostic to mention %q, got: %v", c.mention, err)
			}
		})
	}
}

func TestCrossPackageGenerationShape(t *testing.T) {
	generated, _, err := oracle.Translate(t.TempDir(), map[string]string{
		"fixture": `package fixture

import "oracle.fixture/helper"

func Use() int { return helper.Scale(2, 3) }
`,
		"helper": helperSource,
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	fixtureModule, ok := generated.Files["core/oracle.fixture/fixture/package.ts"]
	if !ok {
		t.Fatalf("fixture module missing; generated files: %v", len(generated.Files))
	}
	if !strings.Contains(fixtureModule, `import * as helper$ from "../helper/package.js";`) {
		t.Fatalf("fixture module lacks the aliased helper import:\n%s", fixtureModule)
	}
	if !strings.Contains(fixtureModule, "helper$.Scale(") {
		t.Fatalf("fixture module lacks the alias-qualified call:\n%s", fixtureModule)
	}

	helperModule, ok := generated.Files["core/oracle.fixture/helper/package.ts"]
	if !ok {
		t.Fatalf("helper module missing")
	}
	if strings.Contains(helperModule, "import * as fixture$") {
		t.Fatalf("helper module must not import the fixture package it never references:\n%s", helperModule)
	}
	if generated.Ownership["core/oracle.fixture/helper/package.ts"] != "generated-core" {
		t.Fatalf("helper module ownership root missing")
	}
}
