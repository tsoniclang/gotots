package emit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/tsident"
)

// TestNoBareHostGlobalInjection is the structural gate proving generated
// code can never let a source-owned declaration capture an injected host
// global: every host-global reference the emitter injects must be
// globalThis-qualified (tsident.Global), so a legal Go binding of the same
// spelling cannot shadow it. It reads every emission string literal in
// this package and fails on any bare `Global(` call — the exact form a
// source binding could capture. globalThis.Global( and identifier-embedded
// spellings (goNumber, myString) are allowed.
func TestNoBareHostGlobalInjection(t *testing.T) {
	globals := map[string]bool{}
	for _, g := range tsident.HostGlobals() {
		if g == "globalThis" {
			continue // the qualifier itself
		}
		globals[g] = true
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			text := lit.Value // includes quotes/backticks; substring search is enough
			for g := range globals {
				for _, idx := range bareCallIndices(text, g) {
					t.Errorf("%s: emission literal injects bare host global %q(...) at offset %d — qualify it through tsident.Global(%q):\n\t%s",
						name, g, idx, g, text)
				}
			}
			return true
		})
	}
}

// bareCallIndices returns the offsets in text where global appears as a
// standalone identifier immediately followed by "(" — not preceded by "."
// (a qualified/property access) or an identifier character (a longer
// name).
func bareCallIndices(text, global string) []int {
	var out []int
	needle := global + "("
	from := 0
	for {
		i := strings.Index(text[from:], needle)
		if i < 0 {
			return out
		}
		at := from + i
		from = at + 1
		if at > 0 {
			prev := text[at-1]
			if prev == '.' || isIdentByte(prev) {
				continue
			}
		}
		out = append(out, at)
	}
}

func isIdentByte(b byte) bool {
	return b == '_' || b == '$' ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
