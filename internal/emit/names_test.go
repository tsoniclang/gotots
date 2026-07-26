package emit

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestPortableIdentifierEscapesNonASCIIWithoutChangingASCII(t *testing.T) {
	for source, expected := range map[string]string{
		"value":     "value",
		"π":         "__u3c0_",
		"Δelta":     "__u394_elta",
		"class":     "__go_class",
		"await":     "__go_await",
		"arguments": "__go_arguments",
	} {
		if actual := portableIdentifier(source); actual != expected {
			t.Fatalf("portableIdentifier(%q) = %q, want %q", source, actual, expected)
		}
	}
}

func TestNameOwnerSeparatesShadowAndTemporaryNamespaces(t *testing.T) {
	packageScope := types.NewScope(nil, token.NoPos, token.NoPos, "package")
	fileScope := types.NewScope(packageScope, token.NoPos, token.NoPos, "file")
	functionScope := types.NewScope(fileScope, token.NoPos, token.NoPos, "function")
	blockScope := types.NewScope(functionScope, token.NoPos, token.NoPos, "block")
	outer := types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int])
	shadow := types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int])
	reservedShadow := types.NewVar(
		token.NoPos,
		nil,
		"value__shadow_1",
		types.Typ[types.Int],
	)
	reservedTemporary := types.NewVar(
		token.NoPos,
		nil,
		"__gotots_assign_0",
		types.Typ[types.Int],
	)
	reservedResults := types.NewVar(
		token.NoPos,
		nil,
		"__gotots_results_0",
		types.Typ[types.Int],
	)
	functionScope.Insert(outer)
	blockScope.Insert(shadow)
	blockScope.Insert(reservedShadow)
	blockScope.Insert(reservedTemporary)
	blockScope.Insert(reservedResults)
	info := &types.Info{Defs: map[*ast.Ident]types.Object{
		{Name: "value"}:             outer,
		{Name: "shadow"}:            shadow,
		{Name: "reservedShadow"}:    reservedShadow,
		{Name: "reservedTemporary"}: reservedTemporary,
		{Name: "reservedResults"}:   reservedResults,
	}}
	owner := newNameOwner(packageScope, info)

	if name, err := owner.declare(outer, targetBinding{}); err != nil || name != "value" {
		t.Fatalf("outer = %q, %v", name, err)
	}
	if name, err := owner.declare(shadow, targetBinding{}); err != nil ||
		name != "value__shadow_2" {
		t.Fatalf("shadow = %q, %v", name, err)
	}
	file := &fileNames{
		owner:       owner,
		temporaries: make(map[api.TemporaryKind]uint64),
	}
	if name, err := file.Temporary(api.TemporaryAssignmentValue); err != nil ||
		name != "__gotots_assign_1" {
		t.Fatalf("temporary = %q, %v", name, err)
	}
	if name, err := file.Temporary(api.TemporaryMultipleResults); err != nil ||
		name != "__gotots_results_1" {
		t.Fatalf("result temporary = %q, %v", name, err)
	}
}

func TestNameOwnerSeparatesNestedButNotSiblingDeclarationSpaces(t *testing.T) {
	packageScope := types.NewScope(nil, token.NoPos, token.NoPos, "package")
	fileScope := types.NewScope(packageScope, token.NoPos, token.NoPos, "file")
	firstFunction := types.NewScope(fileScope, token.NoPos, token.NoPos, "first function")
	firstBlock := types.NewScope(firstFunction, token.NoPos, token.NoPos, "first block")
	siblingBlock := types.NewScope(firstFunction, token.NoPos, token.NoPos, "sibling block")
	secondFunction := types.NewScope(fileScope, token.NoPos, token.NoPos, "second function")
	firstChild := types.NewVar(token.NoPos, nil, "item", types.Typ[types.Int])
	sibling := types.NewVar(token.NoPos, nil, "item", types.Typ[types.Int])
	lateParent := types.NewVar(token.NoPos, nil, "item", types.Typ[types.Int])
	otherFunction := types.NewVar(token.NoPos, nil, "item", types.Typ[types.Int])
	firstBlock.Insert(firstChild)
	siblingBlock.Insert(sibling)
	firstFunction.Insert(lateParent)
	secondFunction.Insert(otherFunction)
	info := &types.Info{Defs: map[*ast.Ident]types.Object{
		{Name: "firstChild"}:    firstChild,
		{Name: "sibling"}:       sibling,
		{Name: "lateParent"}:    lateParent,
		{Name: "otherFunction"}: otherFunction,
	}}
	owner := newNameOwner(packageScope, info)

	if name, err := owner.declare(firstChild, targetBinding{}); err != nil ||
		name != "item__shadow_1" {
		t.Fatalf("first child = %q, %v", name, err)
	}
	if name, err := owner.declare(sibling, targetBinding{}); err != nil ||
		name != "item__shadow_1" {
		t.Fatalf("sibling = %q, %v", name, err)
	}
	if name, err := owner.declare(lateParent, targetBinding{}); err != nil ||
		name != "item" {
		t.Fatalf("late parent = %q, %v", name, err)
	}
	if name, err := owner.declare(otherFunction, targetBinding{}); err != nil ||
		name != "item" {
		t.Fatalf("other function = %q, %v", name, err)
	}
}

func TestNameOwnerRejectsDeclarationOutsideIndexedTypeGraph(t *testing.T) {
	packageScope := types.NewScope(nil, token.NoPos, token.NoPos, "package")
	fileScope := types.NewScope(packageScope, token.NoPos, token.NoPos, "file")
	functionScope := types.NewScope(fileScope, token.NoPos, token.NoPos, "function")
	owner := newNameOwner(packageScope, &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	})
	unindexed := types.NewVar(token.NoPos, nil, "late", types.Typ[types.Int])
	functionScope.Insert(unindexed)

	_, err := owner.declare(unindexed, targetBinding{})
	var nameError *api.NameError
	if !errors.As(err, &nameError) ||
		nameError.Name != "late" ||
		nameError.Reason != "declaration object was not indexed from its Go scope" {
		t.Fatalf("error = %#v, want indexed-scope NameError", err)
	}
}
