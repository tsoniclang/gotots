package api_test

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestCallableControlUsesOneCanonicalArtifactOwner(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/control", "control")
	firstObject := callableControlFunction(sourcePackage, "First", 12)
	secondObject := callableControlFunction(sourcePackage, "Second", 112)
	firstSource := callableControlDeclaration("First", 10, 90)
	secondSource := callableControlDeclaration("Second", 110, 190)

	context, err := (Context{}).WithSourceArtifactOwner(
		MustSourceArtifactOwner(firstObject),
	)
	if err != nil {
		t.Fatal(err)
	}
	if context.ArtifactOwner() != MustSourceArtifactOwner(firstObject) {
		t.Fatal("context lost the canonical artifact owner")
	}
	_, err = context.WithCallableControls(
		MustSourceArtifactOwner(secondObject),
		secondSource,
		nil,
	)
	var invariant *InvariantError
	if !errors.As(err, &invariant) {
		t.Fatalf("divergent owner error = %#v, want InvariantError", err)
	}

	context, err = context.WithCallableControls(
		MustSourceArtifactOwner(firstObject),
		firstSource,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if owner, ok := context.FunctionArtifactOwner(); !ok || owner != firstObject {
		t.Fatalf("derived function owner = %#v, %t", owner, ok)
	}
}

func TestCallableControlRequiresFullCallableContainment(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/control", "control")
	ownerObject := callableControlFunction(sourcePackage, "Outer", 12)
	owner := MustSourceArtifactOwner(ownerObject)
	enclosing := callableControlDeclaration("Outer", 10, 100)
	inside := callableControlLiteral(30, 60)
	outside := callableControlLiteral(70, 120)

	requirement, err := NewCallableControlRequirement(
		owner,
		enclosing,
		inside,
		CallableControlDefer,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedOwner, selectedEnclosing, selectedCallable, facet, ok :=
		requirement.CallableControl()
	if !ok ||
		selectedOwner != owner ||
		selectedEnclosing != enclosing ||
		selectedCallable != inside ||
		facet != CallableControlDefer {
		t.Fatalf("callable-control identity = %#v", requirement)
	}
	if _, err := NewCallableControlRequirement(
		owner,
		enclosing,
		outside,
		CallableControlDefer,
	); err == nil {
		t.Fatal("partially contained function literal acquired callable control")
	}
	if _, err := NewCallableControlRequirement(
		owner,
		inside,
		inside,
		CallableControlDefer,
	); err == nil {
		t.Fatal("foreign enclosing anchor acquired callable control")
	}
}

func TestGotoControlCarriesExactLabelUseIdentity(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/control", "control")
	ownerObject := callableControlFunction(sourcePackage, "Outer", 12)
	owner := MustSourceArtifactOwner(ownerObject)
	enclosing := callableControlDeclaration("Outer", 10, 100)
	label := types.NewLabel(token.Pos(30), sourcePackage, "again")
	foreign := types.NewLabel(token.Pos(40), sourcePackage, "again")
	requirement, err := NewGotoControlRequirement(
		owner,
		enclosing,
		enclosing,
		label,
		token.Pos(50),
	)
	if err != nil {
		t.Fatal(err)
	}
	context, err := (Context{}).WithCallableControls(
		owner,
		enclosing,
		[]DeclarationRequirement{requirement},
	)
	if err != nil {
		t.Fatal(err)
	}
	context = context.EnterCallable(enclosing, nil)
	uses := context.GotoUses(label)
	if len(uses) != 1 || uses[0] != token.Pos(50) {
		t.Fatalf("exact goto uses = %v", uses)
	}
	if len(context.GotoUses(foreign)) != 0 {
		t.Fatal("same-spelling foreign label acquired a goto use")
	}
	if _, err := NewCallableControlRequirement(
		owner,
		enclosing,
		enclosing,
		CallableControlGoto,
	); err == nil {
		t.Fatal("goto control without an exact edge was admitted")
	}
	if _, err := NewDirectCallableControlRequirement(
		ownerObject,
		CallableControlGoto,
	); err == nil {
		t.Fatal("anchor-free goto control was admitted")
	}
	if _, err := NewGotoControlRequirement(
		owner,
		enclosing,
		enclosing,
		label,
		token.Pos(101),
	); err == nil {
		t.Fatal("out-of-callable goto use was admitted")
	}
}

func TestIteratorReturnControlCarriesExactRangeIdentity(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/control", "control")
	ownerObject := callableControlFunction(sourcePackage, "Outer", 12)
	owner := MustSourceArtifactOwner(ownerObject)
	enclosing := callableControlDeclaration("Outer", 10, 100)
	selected := callableControlRange(30, 50)
	sibling := callableControlRange(60, 80)
	requirement, err := NewIteratorReturnControlRequirement(
		owner,
		enclosing,
		enclosing,
		selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedRange, ok := requirement.IteratorReturnControl()
	if !ok || selectedRange != selected {
		t.Fatalf("iterator-return identity = %#v", requirement)
	}
	context, err := (Context{}).WithCallableControls(
		owner,
		enclosing,
		[]DeclarationRequirement{requirement},
	)
	if err != nil {
		t.Fatal(err)
	}
	demand := context.CallableControlFor(enclosing)
	if !demand.IteratorReturn(selected) {
		t.Fatal("selected range lacks iterator-return control")
	}
	if demand.IteratorReturn(sibling) {
		t.Fatal("same-callable sibling range acquired iterator-return control")
	}
	if _, err := NewCallableControlRequirement(
		owner,
		enclosing,
		enclosing,
		CallableControlIteratorReturn,
	); err == nil {
		t.Fatal("range-free iterator-return control was admitted")
	}
	if _, err := NewDirectCallableControlRequirement(
		ownerObject,
		CallableControlIteratorReturn,
	); err == nil {
		t.Fatal("anchor-free iterator-return control was admitted")
	}
	if _, err := NewIteratorReturnControlRequirement(
		owner,
		enclosing,
		enclosing,
		callableControlRange(90, 120),
	); err == nil {
		t.Fatal("out-of-callable iterator-return range was admitted")
	}
}

func callableControlFunction(
	sourcePackage *types.Package,
	name string,
	position token.Pos,
) *types.Func {
	return types.NewFunc(
		position,
		sourcePackage,
		name,
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
}

func callableControlDeclaration(
	name string,
	start token.Pos,
	end token.Pos,
) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: &ast.Ident{NamePos: start + 2, Name: name},
		Type: &ast.FuncType{
			Func:   start,
			Params: &ast.FieldList{Opening: start + 4, Closing: start + 5},
		},
		Body: &ast.BlockStmt{Lbrace: start + 6, Rbrace: end - 1},
	}
}

func callableControlLiteral(start token.Pos, end token.Pos) *ast.FuncLit {
	return &ast.FuncLit{
		Type: &ast.FuncType{
			Func:   start,
			Params: &ast.FieldList{Opening: start + 4, Closing: start + 5},
		},
		Body: &ast.BlockStmt{Lbrace: start + 6, Rbrace: end - 1},
	}
}

func callableControlRange(start token.Pos, end token.Pos) *ast.RangeStmt {
	return &ast.RangeStmt{
		For:  start,
		X:    &ast.Ident{NamePos: start + 4, Name: "iterator"},
		Body: &ast.BlockStmt{Lbrace: start + 12, Rbrace: end - 1},
	}
}
