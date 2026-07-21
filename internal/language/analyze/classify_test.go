package analyze

import (
	"errors"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// astInstances is a zero instance of every concrete type implementing ast.Node.
// TestClassifyIsToolchainBijection proves this table equals the toolchain's set
// exactly, so it cannot silently omit a type, and drives Classify over every
// one — a removed classifier arm fails there.
var astInstances = map[string]ast.Node{
	"ArrayType": &ast.ArrayType{}, "AssignStmt": &ast.AssignStmt{}, "BadDecl": &ast.BadDecl{},
	"BadExpr": &ast.BadExpr{}, "BadStmt": &ast.BadStmt{}, "BasicLit": &ast.BasicLit{},
	"BinaryExpr": &ast.BinaryExpr{}, "BlockStmt": &ast.BlockStmt{}, "BranchStmt": &ast.BranchStmt{},
	"CallExpr": &ast.CallExpr{}, "CaseClause": &ast.CaseClause{}, "ChanType": &ast.ChanType{},
	"CommClause": &ast.CommClause{}, "Comment": &ast.Comment{}, "CommentGroup": &ast.CommentGroup{},
	"CompositeLit": &ast.CompositeLit{}, "DeclStmt": &ast.DeclStmt{}, "DeferStmt": &ast.DeferStmt{},
	"Directive": &ast.Directive{}, "Ellipsis": &ast.Ellipsis{}, "EmptyStmt": &ast.EmptyStmt{},
	"ExprStmt": &ast.ExprStmt{}, "Field": &ast.Field{}, "FieldList": &ast.FieldList{},
	"File": &ast.File{}, "ForStmt": &ast.ForStmt{}, "FuncDecl": &ast.FuncDecl{},
	"FuncLit": &ast.FuncLit{}, "FuncType": &ast.FuncType{}, "GenDecl": &ast.GenDecl{},
	"GoStmt": &ast.GoStmt{}, "Ident": &ast.Ident{}, "IfStmt": &ast.IfStmt{},
	"ImportSpec": &ast.ImportSpec{}, "IncDecStmt": &ast.IncDecStmt{}, "IndexExpr": &ast.IndexExpr{},
	"IndexListExpr": &ast.IndexListExpr{}, "InterfaceType": &ast.InterfaceType{},
	"KeyValueExpr": &ast.KeyValueExpr{}, "LabeledStmt": &ast.LabeledStmt{}, "MapType": &ast.MapType{},
	"Package": &ast.Package{}, "ParenExpr": &ast.ParenExpr{}, "RangeStmt": &ast.RangeStmt{},
	"ReturnStmt": &ast.ReturnStmt{}, "SelectStmt": &ast.SelectStmt{}, "SelectorExpr": &ast.SelectorExpr{},
	"SendStmt": &ast.SendStmt{}, "SliceExpr": &ast.SliceExpr{}, "StarExpr": &ast.StarExpr{},
	"StructType": &ast.StructType{}, "SwitchStmt": &ast.SwitchStmt{}, "TypeAssertExpr": &ast.TypeAssertExpr{},
	"TypeSpec": &ast.TypeSpec{}, "TypeSwitchStmt": &ast.TypeSwitchStmt{}, "UnaryExpr": &ast.UnaryExpr{},
	"ValueSpec": &ast.ValueSpec{},
}

// toolchainNodeTypes derives, from the selected toolchain's own go/ast sources,
// the set of concrete type names whose pointer implements ast.Node. This is the
// independent truth the catalog is reconciled against.
func toolchainNodeTypes(t *testing.T) map[string]bool {
	t.Helper()
	pkg, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import("go/ast")
	if err != nil {
		t.Fatalf("import go/ast from source: %v", err)
	}
	node, ok := pkg.Scope().Lookup("Node").Type().Underlying().(*types.Interface)
	if !ok {
		t.Fatal("go/ast.Node is not an interface")
	}
	names := map[string]bool{}
	for _, name := range pkg.Scope().Names() {
		typeName, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := typeName.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, isIface := named.Underlying().(*types.Interface); isIface {
			continue
		}
		if types.Implements(types.NewPointer(named), node) {
			names[name] = true
		}
	}
	return names
}

// TestClassifyIsToolchainBijection proves a total, toolchain-derived bijection:
// the instance table equals the toolchain node set, every instance classifies
// to the identically named catalog kind, and the classifier covers exactly the
// catalog. Removing a classifier arm, a catalog kind, or a toolchain type each
// fails this test.
func TestClassifyIsToolchainBijection(t *testing.T) {
	truth := toolchainNodeTypes(t)
	for name := range truth {
		if _, ok := astInstances[name]; !ok {
			t.Errorf("toolchain ast.Node type %s missing from instance table", name)
		}
	}
	for name := range astInstances {
		if !truth[name] {
			t.Errorf("instance table entry %s is not a toolchain ast.Node type", name)
		}
	}
	covered := map[catalog.Kind]bool{}
	for name, node := range astInstances {
		kind, err := Classify(node)
		if err != nil {
			t.Errorf("Classify(%s) failed closed: %v", name, err)
			continue
		}
		if kind.Name() != name {
			t.Errorf("Classify(%s) = %s, want a kind named %s", name, kind, name)
		}
		covered[kind] = true
	}
	for _, kind := range catalog.All() {
		if !covered[kind] {
			t.Errorf("catalog kind %s is not reached by any ast node instance", kind)
		}
	}
	if len(covered) != len(catalog.All()) {
		t.Errorf("classifier covered %d kinds, catalog has %d", len(covered), len(catalog.All()))
	}
}

// unknownNode is a synthetic ast.Node with no catalog identity.
type unknownNode struct{}

func (unknownNode) Pos() token.Pos { return token.NoPos }
func (unknownNode) End() token.Pos { return token.NoPos }

// TestClassifyFailsClosedOnUnknown is the mutation proof: an unrecognized node
// yields a typed UnknownConstructError, never a default classification.
func TestClassifyFailsClosedOnUnknown(t *testing.T) {
	kind, err := Classify(unknownNode{})
	if err == nil {
		t.Fatalf("Classify accepted an unknown node as %s", kind)
	}
	var unknown *UnknownConstructError
	if !errors.As(err, &unknown) {
		t.Fatalf("Classify error = %T (%v), want *UnknownConstructError", err, err)
	}
	if kind != catalog.KindInvalid {
		t.Errorf("Classify returned %s with an error, want KindInvalid", kind)
	}
}
