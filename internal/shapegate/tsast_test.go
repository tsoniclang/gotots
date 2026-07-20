package shapegate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func pinnedTS(t *testing.T) (string, string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	tsDir := "../../product/node_modules/typescript"
	if _, err := os.Stat(tsDir); err != nil {
		t.Skip("pinned typescript unavailable")
	}
	return node, tsDir
}

// The extractor parses real TypeScript through the pinned compiler and
// the AST joins replace the textual counters: a definition declared in
// two modules is a duplicate; a call with surplus arguments over the
// source arity is a hidden-argument site.
func TestTypedASTShapeFacts(t *testing.T) {
	node, tsDir := pinnedTS(t)
	dir := t.TempDir()
	one := filepath.Join(dir, "one.ts")
	two := filepath.Join(dir, "two.ts")
	os.WriteFile(one, []byte(`export type U = { k: 1 } | undefined;
declare function Keys(p: unknown): string;
declare const points: unknown, zero$T: unknown, eq$T: unknown;
export function Shared(x: string): string { return Keys(points, zero$T, eq$T); }
`), 0o644)
	os.WriteFile(two, []byte(`export type U = { k: 1 } | undefined;
export function Shared(x: string): string { return Keys(points); }
class Box { m(a: number): void {} }
declare function Keys(p: unknown): string;
declare const points: unknown;
const evade = Keys;
export const evaded = evade(points, 1, 2);
`), 0o644)
	shapes, err := ExtractShapes(node, tsDir, []string{one, two})
	if err != nil {
		t.Fatal(err)
	}
	dups := DuplicateDefinitions(shapes)
	if len(dups["function::Shared"]) != 2 || len(dups["alias::U"]) != 2 {
		t.Fatalf("duplicate join missed: %v", dups)
	}
	// Keys has source arity 1; the call in one.ts carries two hidden
	// operation arguments, and the const-bound alias call in two.ts
	// resolves through the checker — evasion is impossible.
	surplus := CallArgSurplus(shapes, "Keys", 1)
	if len(surplus) != 2 {
		t.Fatalf("hidden-argument join must catch the direct AND aliased call: %+v", surplus)
	}
	// Methods and classes are visible declarations.
	var sawMethod bool
	for _, s := range shapes {
		for _, d := range s.Declarations {
			if d.Kind == "method" && d.Name == "Box.m" && d.Params == 1 {
				sawMethod = true
			}
		}
	}
	if !sawMethod {
		t.Fatal("method declaration fact missing")
	}
}
