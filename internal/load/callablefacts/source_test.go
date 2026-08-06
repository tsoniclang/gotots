package callablefacts

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestPointeeValueEvidenceSeparatesEntrySnapshotsFromLocations(t *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "source.go", `package source
func Read(value *int) int { return *value }
type Reader struct{}
func (Reader) Method(value *int) int { return *value }
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
	expected := map[string]PointeeReadEvidence{
		"Read":        PointeeReadEntryStable,
		"Method":      PointeeReadEntryStable,
		"Conditional": PointeeReadInvalid,
		"LocalWork":   PointeeReadEntryStable,
		"Convert":     PointeeReadEntryStable,
		"Mutate":      PointeeReadInvalid,
		"Escape":      PointeeReadInvalid,
		"Delayed":     PointeeReadInvalid,
		"StoreGlobal": PointeeReadInvalid,
		"StoreAlias":  PointeeReadInvalid,
		"Capture":     PointeeReadInvalid,
		"Synchronize": PointeeReadInvalid,
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		object := info.Defs[function.Name].(*types.Func)
		identity, identityErr := SourceCallableIdentity(object)
		if identityErr != nil || identity == "" {
			t.Fatalf("%s identity = %q, error %v", function.Name.Name, identity, identityErr)
		}
		actual := PointeeValueEvidence(
			function,
			object.Signature().Params().At(0),
			info,
		)
		if actual != expected[function.Name.Name] {
			t.Fatalf(
				"%s pointee-value evidence = %d, want %d",
				function.Name.Name,
				actual,
				expected[function.Name.Name],
			)
		}
	}
}
