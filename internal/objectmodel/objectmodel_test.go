package objectmodel

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func analyzeSrc(t *testing.T, src string) *Plan {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "m.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("m", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var named []*types.Named
	for _, name := range pkg.Scope().Names() {
		if tn, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok {
			if n, ok := tn.Type().(*types.Named); ok {
				named = append(named, n)
			}
		}
	}
	return Analyze(named)
}

// The AST shape: a spine Node <- NodeDefault <- NodeBase <- ExprBase,
// concrete leaves embedding the spine plus a secondary capability base.
func TestRecoversSpineAndSecondary(t *testing.T) {
	plan := analyzeSrc(t, `package m
type nodeData interface { Name() *Node; Kind() int }
type Node struct { data nodeData }
type NodeDefault struct { Node }
type NodeBase struct { NodeDefault }
type ExprBase struct { NodeBase }
type FlowBase struct { flow int }
type Identifier struct { ExprBase; FlowBase; Text string }
type Binary struct { ExprBase; Op int }
type Block struct { NodeBase; Stmts int }
func (n *Node) Name() *Node { return n.data.Name() }
func (n *Node) Kind() int { return n.data.Kind() }
func (i *Identifier) Name() *Node { return &i.Node }
func (i *Identifier) Kind() int { return 1 }
func (b *Binary) Name() *Node { return nil }
func (b *Binary) Kind() int { return 2 }
func (b *Block) Name() *Node { return nil }
func (b *Block) Kind() int { return 3 }
`)
	fams := plan.Families()
	if len(fams) != 1 {
		t.Fatalf("expected one family, got %v", fams)
	}
	fam, _ := plan.Family(fams[0])
	if fam.RootType != "Node" {
		t.Fatalf("root = %q; want Node", fam.RootType)
	}
	id, ok := plan.Class("m.Identifier")
	if !ok || id.PrimaryBaseType != "ExprBase" {
		t.Fatalf("Identifier primary base = %+v; want ExprBase", id)
	}
	if len(id.Secondary) != 1 || id.Secondary[0] != "FlowBase" {
		t.Fatalf("Identifier secondary = %v; want [FlowBase]", id.Secondary)
	}
	root, _ := plan.Class("m.Node")
	if !root.Root || root.PrimaryBase != "" {
		t.Fatalf("Node should be a root with no primary base: %+v", root)
	}
	expr, _ := plan.Class("m.ExprBase")
	if expr.PrimaryBaseType != "NodeBase" {
		t.Fatalf("ExprBase primary = %q; want NodeBase", expr.PrimaryBaseType)
	}
}

// Genuine multiple inheritance (two embedded bases both reaching the
// root) is not expressible as a single spine → the family is skipped.
func TestMultipleSpinesSkipped(t *testing.T) {
	plan := analyzeSrc(t, `package m
type I interface { M() int }
type Root struct { x int }
type A struct { Root }
type B struct { Root }
type Diamond struct { A; B }
type P struct { Root; p int }
type Q struct { Root; q int }
func (r *Root) M() int { return 0 }
`)
	// Diamond embeds A and B, both reaching Root → no single spine.
	if _, ok := plan.Class("m.Diamond"); ok {
		t.Fatal("diamond multiple inheritance must not be planned")
	}
}
