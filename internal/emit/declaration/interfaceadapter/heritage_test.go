package interfaceadapter

import (
	"go/token"
	"go/types"
	"testing"
)

func TestDeclarationHeritageRequiresAFileLevelContract(t *testing.T) {
	owner := types.NewPackage("example.com/heritage", "heritage")
	methodSet := types.NewInterfaceType(nil, nil).Complete()
	packageName := types.NewTypeName(token.NoPos, owner, "Package", nil)
	packageType := types.NewNamed(packageName, methodSet, nil)
	if existing := owner.Scope().Insert(packageName); existing != nil {
		t.Fatal("package interface insertion failed")
	}

	functionScope := types.NewScope(
		owner.Scope(),
		token.Pos(1),
		token.Pos(100),
		"function",
	)
	localName := types.NewTypeName(token.Pos(2), owner, "Local", nil)
	localType := types.NewNamed(localName, methodSet, nil)
	if existing := functionScope.Insert(localName); existing != nil {
		t.Fatal("local interface insertion failed")
	}

	if !declarationHeritageSurface(packageType) {
		t.Fatal("package interface was not eligible for declaration heritage")
	}
	if declarationHeritageSurface(localType) {
		t.Fatal("function-local interface escaped into declaration heritage")
	}
	if !declarationHeritageSurface(methodSet) {
		t.Fatal("anonymous canonical contract was excluded from declaration heritage")
	}
	if !declarationHeritageSurface(types.Universe.Lookup("error").Type()) {
		t.Fatal("predeclared canonical contract was excluded from declaration heritage")
	}
}
