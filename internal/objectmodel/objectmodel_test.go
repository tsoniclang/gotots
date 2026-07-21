package objectmodel

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func analyze(t *testing.T, src string) *Plan {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "m.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("m", fset, []*ast.File{file}, nil)
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

const astShape = `package m
type nodeData interface { Name() *Node; NodeKind() int }
type Node struct { Kind int; data nodeData }
type NodeDefault struct { Node }
type NodeBase struct { NodeDefault }
type ExprBase struct { NodeBase }
type TypeNodeBase struct { NodeBase }
type FlowBase struct { flow int }
type Identifier struct { ExprBase; FlowBase; Text string }
type Block struct { NodeBase; Stmts int }
type FuncOrCtorBase struct { TypeNodeBase; mods int }
type FunctionTypeNode struct { TypeNodeBase; FuncOrCtorBase }
func (n *Node) Name() *Node { return n.data.Name() }
func (n *Node) NodeKind() int { return n.data.NodeKind() }
func (i *Identifier) Name() *Node { return &i.Node }
func (i *Identifier) NodeKind() int { return 1 }
func (b *Block) Name() *Node { return nil }
func (b *Block) NodeKind() int { return 3 }
func (f *FunctionTypeNode) Name() *Node { return nil }
func (f *FunctionTypeNode) NodeKind() int { return 4 }
func (node *NodeDefault) Name() *Node { return nil }
func (node *NodeDefault) NodeKind() int { return 0 }
func (b *NodeBase) Loc() int { return 0 }
func (e *ExprBase) IsExpr() bool { return true }
`

func TestRecognitionBySelfReference(t *testing.T) {
	plan := analyze(t, astShape)
	fams := plan.Families()
	if len(fams) != 1 {
		t.Fatalf("families = %v (blocked=%v)", fams, plan.Blocked())
	}
	fam, _ := plan.Family(fams[0])
	if fam.RootType != "Node" || fam.SelfField != "data" {
		t.Fatalf("family = %+v; want root Node, selfField data", fam)
	}
}

func TestPrimaryAndSecondary(t *testing.T) {
	plan := analyze(t, astShape)
	id, ok := plan.Class("m.Identifier")
	if !ok || id.Primary.BaseType != "ExprBase" {
		t.Fatalf("Identifier primary = %+v; want ExprBase", id.Primary)
	}
	if len(id.Secondary) != 1 || id.Secondary[0].BaseType != "FlowBase" {
		t.Fatalf("Identifier secondary = %+v; want [FlowBase]", id.Secondary)
	}
}

// The shallow-path case: FunctionTypeNode embeds TypeNodeBase (depth to
// root D) AND FuncOrCtorBase (which nests TypeNodeBase, depth D+1). Go
// selects the shallower TypeNodeBase; it is placeable, not a diamond.
func TestShallowPathDominance(t *testing.T) {
	plan := analyze(t, astShape)
	f, ok := plan.Class("m.FunctionTypeNode")
	if !ok {
		t.Fatalf("FunctionTypeNode not placed (blocked=%v)", plan.Blocked())
	}
	if f.Primary.BaseType != "TypeNodeBase" {
		t.Fatalf("primary = %q; want TypeNodeBase (shallower path)", f.Primary.BaseType)
	}
	if len(f.Secondary) != 1 || f.Secondary[0].BaseType != "FuncOrCtorBase" {
		t.Fatalf("secondary = %+v; want [FuncOrCtorBase]", f.Secondary)
	}
}

// Genuine equal-depth ambiguity blocks the whole family (4.5).
func TestEqualDepthAmbiguityBlocks(t *testing.T) {
	plan := analyze(t, `package m
type I interface { M() int }
type Root struct { data I }
type A struct { Root }
type B struct { Root }
type Diamond struct { A; B }
type P struct { Root; p int }
func (r *Root) M() int { return r.data.M() }
func (d *Diamond) M() int { return 1 }
func (p *P) M() int { return 2 }
`)
	if len(plan.Families()) != 0 {
		t.Fatalf("equal-depth diamond must block the family; got %v", plan.Families())
	}
	if len(plan.Blocked()) == 0 {
		t.Fatal("expected a blocked family with a reason")
	}
}

// Pointer embedding is not value embedding: a *Base is a component, not
// a superclass (a nil embedded pointer cannot be a JS superclass).
func TestPointerEmbeddingIsComponent(t *testing.T) {
	plan := analyze(t, `package m
type I interface { M() int }
type Root struct { data I }
type Base struct { Root }
type Ptr struct { *Base; x int }
type Val struct { Base; y int }
func (r *Root) M() int { return r.data.M() }
func (b *Base) M() int { return 0 }
func (p *Ptr) M() int { return 1 }
func (v *Val) M() int { return 2 }
`)
	// Ptr embeds *Base (pointer) — no value path to Root → not placeable
	// via the spine, so the family blocks rather than treating *Base as
	// a superclass.
	if _, ok := plan.Class("m.Ptr"); ok {
		if c, _ := plan.Class("m.Ptr"); c.HasPrimary {
			t.Fatal("pointer-embedded base must not become a superclass")
		}
	}
}

// A coincidental interface whose implementers merely share an embedded
// struct, with NO self-reference root, is not an object-model family.
func TestCoincidentalInterfaceNotAFamily(t *testing.T) {
	plan := analyze(t, `package m
type Stringer interface { String() string }
type Base struct { x int }
type A struct { Base }
type B struct { Base }
type C struct { Base }
func (a A) String() string { return "a" }
func (b B) String() string { return "b" }
func (c C) String() string { return "c" }
`)
	if len(plan.Families()) != 0 {
		t.Fatalf("coincidental interface must not be a family; got %v", plan.Families())
	}
}

// Method dispositions: root contract methods become abstract
// trampolines-removed; the base default class declares/overrides them;
// a concrete leaf inherits the spine, overrides its own, and delegates
// a secondary-component method.
func TestMethodDispositions(t *testing.T) {
	plan := analyze(t, astShape)
	node, _ := plan.Class("m.Node")
	if node.Methods["Name"] != MethodTrampolineRemoved || node.Methods["NodeKind"] != MethodTrampolineRemoved {
		t.Fatalf("Node contract methods must be trampoline-removed: %v", node.Methods)
	}
	nd, _ := plan.Class("m.NodeDefault")
	if nd.Methods["Name"] != MethodOverride {
		t.Fatalf("NodeDefault should override Name (default impl): %v", nd.Methods["Name"])
	}
	id, _ := plan.Class("m.Identifier")
	if id.Methods["Name"] != MethodOverride {
		t.Fatalf("Identifier.Name should be override: %v", id.Methods["Name"])
	}
	// Methods promoted through the primary spine are inherited (emit
	// nothing), never forwarding members.
	inherited := 0
	for _, d := range id.Methods {
		if d == MethodInherited {
			inherited++
		}
	}
	if inherited == 0 {
		t.Fatalf("Identifier must inherit promoted spine methods: %v", id.Methods)
	}
	// Every method has exactly one disposition (completeness).
	for name, d := range id.Methods {
		if d == "" {
			t.Fatalf("method %s has no disposition", name)
		}
	}
}
