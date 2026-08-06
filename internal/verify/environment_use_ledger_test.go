package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnvironmentUseHasOneCanonicalOwner proves no parallel environment-use
// ledger exists: the canonical scheduler record and the immutable obligation
// projection are the only production holders of implementation-route state
// in the emit root, and the deleted environmentUses ledger never returns.
func TestEnvironmentUseHasOneCanonicalOwner(t *testing.T) {
	root := repositoryRoot(t)
	emitDirectory := filepath.Join(root, "internal", "emit")
	entries, err := os.ReadDir(emitDirectory)
	if err != nil {
		t.Fatal(err)
	}
	allowedRouteHolders := map[string]struct{}{
		"declarationRecord":     {},
		"EnvironmentObligation": {},
	}
	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(emitDirectory, entry.Name())
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(payload), "environmentUses") {
			t.Errorf(
				"%s resurrects the parallel environmentUses ledger",
				entry.Name(),
			)
		}
		file, err := parser.ParseFile(fileSet, path, payload, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			specification, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			structure, ok := specification.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range structure.Fields.List {
				selector, ok := field.Type.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				if selector.Sel.Name != "ImplementationRoute" {
					continue
				}
				if _, allowed := allowedRouteHolders[specification.Name.Name]; !allowed {
					t.Errorf(
						"%s: struct %s duplicates implementation-route state outside the canonical scheduler record",
						entry.Name(),
						specification.Name.Name,
					)
				}
			}
			return true
		})
	}
}
