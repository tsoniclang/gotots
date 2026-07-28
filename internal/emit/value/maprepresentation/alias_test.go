package maprepresentation_test

import (
	"go/ast"
	"go/types"
	"testing"
)

func TestMapAliasMakeUsesSemanticTypeInsteadOfASTShape(t *testing.T) {
	loaded := loadMapValuesProject(t)
	aliasArguments := 0
	for _, file := range loaded.Files() {
		ast.Inspect(file.Syntax(), func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if !ok ||
				loaded.TypesInfo().Uses[identifier] != types.Universe.Lookup("make") {
				return true
			}
			if _, ok := call.Args[0].(*ast.Ident); ok {
				aliasArguments++
			}
			return true
		})
	}
	if aliasArguments != 1 {
		t.Fatalf("make alias identifier arguments = %d, want one", aliasArguments)
	}
	if _, err := compileExportedResult(loaded); err != nil {
		t.Fatalf("semantic map alias failed direct emission: %v", err)
	}
}
