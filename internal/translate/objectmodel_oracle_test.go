package translate_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// familyFixtureSource mirrors the corpus nodeData object model in
// miniature: a virtual-contract interface (nodeData), a self-reference
// root (Node.data), a multi-level value-embedding spine (NodeBase →
// ExprBase), a concrete leaf that also embeds a secondary component
// (Identifier{ExprBase; FlowNodeBase; Text}), virtual dispatch through
// the self-reference (Node.Kind/Node.Name → n.data.…), a promoted field
// access (Node.Pos), and construction by the arena-free zero+assign+
// self-reference pattern the factories use. It is the fast differential
// and shape oracle for the ADR-0012 native-class cutover.
const familyFixtureSource = `package family

// The contract methods share their names with the root trampolines
// (Kind/Name), exactly as the corpus nodeData does — the collapse must not
// recurse. describe() is a default declared ONLY on the root (no
// implementer overrides it), the watcherBase-style default pattern.
type nodeData interface {
	Kind() int
	Name() string
	describe() string
}

type Node struct {
	data nodeData
	Pos  int
}

func (n *Node) Kind() int        { return n.data.Kind() }
func (n *Node) Name() string     { return n.data.Name() }
func (n *Node) describe() string { return "node" }
func (n *Node) Position() int    { return n.Pos }

type NodeBase struct{ Node }

func (b *NodeBase) Kind() int    { return 0 }
func (b *NodeBase) Name() string { return "" }

type ExprBase struct{ NodeBase }

func (e *ExprBase) IsExpr() bool { return true }

type FlowNodeBase struct{ Flow int }

func (f *FlowNodeBase) FlowValue() int { return f.Flow }

type Identifier struct {
	ExprBase
	FlowNodeBase
	Text string
}

func (i *Identifier) Kind() int    { return 1 }
func (i *Identifier) Name() string { return i.Text }

func newIdentifier(text string, pos int) *Node {
	id := &Identifier{Text: text}
	id.Pos = pos
	id.data = id
	return &id.Node
}

type NumericLiteral struct {
	ExprBase
	Value int
}

func (l *NumericLiteral) Kind() int { return 2 }

// NumericLiteral does NOT override Name(): it inherits NodeBase's "".

func newNumericLiteral(value int, pos int) *Node {
	lit := &NumericLiteral{Value: value}
	lit.Pos = pos
	lit.data = lit
	return &lit.Node
}

func MakeIdentifier(text string, pos int) *Node   { return newIdentifier(text, pos) }
func MakeNumericLiteral(value int, pos int) *Node { return newNumericLiteral(value, pos) }
func NodeKindOf(n *Node) int                      { return n.Kind() }
func NodeNameOf(n *Node) string                   { return n.Name() }
func NodePositionOf(n *Node) int                  { return n.Position() }
func NodeDescribeOf(n *Node) string               { return n.describe() }
`

// familyDriverSource is a fixture package exercising the family through
// its public surface: virtual dispatch, promoted access, and the
// secondary component, so the differential covers the semantics the
// native-class cutover must preserve exactly.
const familyDriverSource = `package fixture

import "oracle.fixture/family"

func IdentifierKind() int {
	return family.NodeKindOf(family.MakeIdentifier("x", 5))
}

func IdentifierName() string {
	return family.NodeNameOf(family.MakeIdentifier("hello", 0))
}

func IdentifierPosition() int {
	return family.NodePositionOf(family.MakeIdentifier("y", 42))
}

func LiteralKindAndInheritedName() (int, string) {
	// NumericLiteral inherits NodeBase's default Name() == "".
	n := family.MakeNumericLiteral(7, 3)
	return family.NodeKindOf(n), family.NodeNameOf(n)
}

func VirtualDispatchAcrossKinds() (int, int) {
	return family.NodeKindOf(family.MakeIdentifier("a", 0)),
		family.NodeKindOf(family.MakeNumericLiteral(9, 0))
}

func RootDefaultDescribe() (string, string) {
	// describe() is a default declared only on the root; every node
	// inherits it.
	return family.NodeDescribeOf(family.MakeIdentifier("a", 0)),
		family.NodeDescribeOf(family.MakeNumericLiteral(1, 0))
}
`

func runFamilyOracle(t *testing.T) *oracle.Result {
	t.Helper()
	result, err := oracle.Run(t.TempDir(), map[string]string{
		"fixture": familyDriverSource,
		"family":  familyFixtureSource,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("family differential mismatch across %d cases:\n--- go ---\n%s--- generated ---\n%s",
			result.Cases, result.GoOutput, result.TSOutput)
	}
	return result
}

// TestFamilyObjectModelDifferential proves the recovered object model
// preserves Go semantics exactly: virtual dispatch through the
// self-reference, promoted field access, and multi-kind dispatch all
// return identical results in Go and generated TypeScript. This
// differential is the invariant the ADR-0012 native-class cutover must
// keep green — the emitted shape changes, the observable semantics do
// not.
func TestFamilyObjectModelDifferential(t *testing.T) {
	result := runFamilyOracle(t)
	if result.Cases == 0 {
		t.Fatalf("no differential cases ran")
	}
}
