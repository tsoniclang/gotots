package callableabi

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestPointeeValueProofRejectsMutationEscapeAndDelayedRead(t *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "source.go", `package source
func Read(value *int) int { return *value }
func Mutate(value *int) int { (*value)++; return *value }
func Escape(value *int) *int { return value }
func Delayed(value *int) int { println("before"); return *value }
`, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
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
	for _, declaration := range file.Decls {
		function := declaration.(*ast.FuncDecl)
		object := info.Defs[function.Name].(*types.Func)
		actual := PointeeValueReadAtEntry(
			function,
			object.Signature().Params().At(0),
			info,
		)
		if actual != (function.Name.Name == "Read") {
			t.Fatalf("%s pointee-value proof = %t", function.Name.Name, actual)
		}
	}
}
