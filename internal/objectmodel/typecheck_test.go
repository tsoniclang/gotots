package objectmodel

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

// namedTypesOf type-checks one source string and returns its package-
// level named types, exactly the set translate.buildObjectModel collects
// from a loaded package's scope. This proves the recovery on real
// go/types data (not hand-built fixtures), the collection contract the
// translate wiring depends on.
func namedTypesOf(t *testing.T, src string) []*types.Named {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}}
	pkg, err := conf.Check("p", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	var named []*types.Named
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || obj.IsAlias() {
			continue
		}
		if n, ok := types.Unalias(obj.Type()).(*types.Named); ok {
			named = append(named, n)
		}
	}
	return named
}

// TestRecoveryOverTypecheckedSource proves the analysis over a real
// go/types package: a self-reference root (Node.data nodeData), a value-
// embedding spine (NodeDefault{Node}, ExprBase{NodeDefault}), and a
// concrete leaf (Ident{ExprBase}) recover as a single family with Ident
// extending ExprBase and inheriting the promoted spine method.
func TestRecoveryOverTypecheckedSource(t *testing.T) {
	const src = `package p

type nodeData interface { name() string }

type Node struct {
	data nodeData
	Pos  int
}
func (n *Node) name() string { return n.data.name() }

type NodeDefault struct { Node }
func (d *NodeDefault) name() string { return "" }

type ExprBase struct { NodeDefault }
func (e *ExprBase) IsExpr() bool { return true }

type Ident struct {
	ExprBase
	Text string
}
func (i *Ident) name() string { return i.Text }
`
	plan := Analyze(namedTypesOf(t, src))

	fams := plan.Families()
	if len(fams) != 1 {
		t.Fatalf("want exactly one family, got %v (blocked=%v)", fams, plan.Blocked())
	}
	fam, _ := plan.Family(fams[0])
	if fam.RootType != "Node" || fam.SelfField != "data" {
		t.Fatalf("family root/self = %q/%q, want Node/data", fam.RootType, fam.SelfField)
	}
	// The contract method (name) is exposed for the root's synthesized
	// abstract declaration.
	if len(fam.ContractMethods) != 1 || fam.ContractMethods[0] != "name" {
		t.Fatalf("contract methods = %v, want [name]", fam.ContractMethods)
	}

	ident, ok := plan.Class("p.Ident")
	if !ok {
		t.Fatalf("Ident not placed; blocked=%v", plan.Blocked())
	}
	if !ident.HasPrimary || ident.Primary.BaseType != "ExprBase" {
		t.Fatalf("Ident primary = %+v, want ExprBase", ident.Primary)
	}
	// name() is redefined on Ident → override; IsExpr promoted through the
	// primary spine → inherited (emit nothing).
	if got := ident.Methods["name"]; got != MethodOverride {
		t.Fatalf("Ident.name disposition = %q, want override", got)
	}
	if got := ident.Methods["IsExpr"]; got != MethodInherited {
		t.Fatalf("Ident.IsExpr disposition = %q, want inherited", got)
	}

	root, ok := plan.Class("p.Node")
	if !ok || !root.Root {
		t.Fatalf("Node not recovered as root: %+v", root)
	}
	// The root's contract method is a trampoline removed to abstract.
	if got := root.Methods["name"]; got != MethodTrampolineRemoved {
		t.Fatalf("Node.name disposition = %q, want trampoline-removed", got)
	}
}
