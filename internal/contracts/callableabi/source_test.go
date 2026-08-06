package callableabi

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestPointeeValuePolicySeparatesEntrySnapshotsFromLocations(t *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "source.go", `package source
func Read(value *int) int { return *value }
func Conditional(value *int, selected bool) int {
	if selected { return *value }
	return *value + 1
}
func LocalWork(value *int) int {
	current := *value
	current++
	return current + *value
}
func Convert(value *int) int64 { return int64(*value) }
func Mutate(value *int) int { (*value)++; return *value }
func Escape(value *int) *int { return value }
func Delayed(value *int) int { println("before"); return *value }
var Global int
func StoreGlobal(value *int) int { Global++; return *value }
func StoreAlias(value *int, other *int) int { *other = 1; return *value }
func Capture(value *int) func() int { return func() int { return *value } }
func Synchronize(value *int, ready chan struct{}) int { <-ready; return *value }
`, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	configuration := types.Config{Importer: importer.Default()}
	if _, err := configuration.Check(
		"example.test/source",
		fileSet,
		[]*ast.File{file},
		info,
	); err != nil {
		t.Fatal(err)
	}
	expected := map[string]NilPolicy{
		"Read":        NilPolicyRejectAtBoundary,
		"Conditional": NilPolicyInvalid,
		"LocalWork":   NilPolicyRejectAtBoundary,
		"Convert":     NilPolicyRejectAtBoundary,
		"Mutate":      NilPolicyInvalid,
		"Escape":      NilPolicyInvalid,
		"Delayed":     NilPolicyInvalid,
		"StoreGlobal": NilPolicyInvalid,
		"StoreAlias":  NilPolicyInvalid,
		"Capture":     NilPolicyInvalid,
		"Synchronize": NilPolicyInvalid,
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		object := info.Defs[function.Name].(*types.Func)
		actual := PointeeValuePolicy(
			function,
			object.Signature().Params().At(0),
			info,
		)
		if actual != expected[function.Name.Name] {
			t.Fatalf(
				"%s pointee-value policy = %d, want %d",
				function.Name.Name,
				actual,
				expected[function.Name.Name],
			)
		}
	}
}
