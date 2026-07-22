package analyze

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// reconcileFixture exercises a broad span of Go grammatical forms so the
// parent-directed visitor is reconciled against constructs the toolchain parser
// actually produces.
const reconcileFixture = `// Package p exercises many Go constructs for inventory reconciliation.
package p

import (
	"fmt"
	m "math"
)

// C is a constant.
const C = (1 + 2)

var V, W = 3, "s"

// T is a struct type.
type T struct {
	A int
	B []string
}

// I is an interface type.
type I interface {
	M(x int) (int, error)
}

// Pair is generic.
type Pair[K comparable, V any] struct {
	Key K
	Val V
}

type Alias = T

type Fn func(a int, b ...string) int

// M is a method on T.
func (t *T) M(x int) (int, error) {
	var mp map[string]int = map[string]int{"a": 1}
	ch := make(chan int, 1)
	arr := [3]int{1, 2, 3}
	s := arr[1:2]
	e := arr[0]
	p := &t.A
	*p = x + e
	i := 0
	i++
	i--
	go func() { fmt.Println("g") }()
	defer func() { _ = recover() }()
	ch <- 1
	select {
	case v := <-ch:
		_ = v
	default:
	}
Loop:
	for k, val := range mp {
		_ = val
		if k == "a" {
			continue Loop
		} else {
			break
		}
	}
	switch x {
	case 1:
		fallthrough
	default:
	}
	var a interface{} = t
	switch a.(type) {
	case *T:
	}
	y := a.(*T)
	pair := Pair[int, string]{Key: 1, Val: "x"}
	f := Fn(func(a int, b ...string) int { return a })
	_ = f(1, "a", "b")
	_, _, _, _, _ = s, y, pair, mp, m.Pi
	return x, nil
}
`

func loadSource(t *testing.T, content string) *source.File {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	loaded, err := source.Load(dir, path)
	if err != nil {
		t.Fatalf("source.Load: %v", err)
	}
	return loaded
}

// TestBuildInventoryExactJoinsIndependentWalk joins the inventory to an
// independent toolchain traversal on node identity (concrete type and physical
// span), exact preorder position, and exact parent edge — not counts. A
// reordered child visit, a wrong parent, or a missed/extra node all fail.
func TestBuildInventoryExactJoinsIndependentWalk(t *testing.T) {
	src := loadSource(t, reconcileFixture)
	inv, err := BuildInventory(src)
	if err != nil {
		t.Fatalf("BuildInventory: %v", err)
	}
	type expectation struct {
		goType string
		start  int
		end    int
		parent int // preorder index of the parent, -1 for the root
	}
	var walk []expectation
	var stack []int
	ast.Inspect(src.Syntax(), func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		parent := -1
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		walk = append(walk, expectation{
			goType: strings.TrimPrefix(fmt.Sprintf("%T", n), "*ast."),
			start:  src.Fset().PositionFor(n.Pos(), false).Offset,
			end:    src.Fset().PositionFor(n.End(), false).Offset,
			parent: parent,
		})
		stack = append(stack, len(walk)-1)
		return true
	})
	if len(inv.Occurrences) != len(walk) {
		t.Fatalf("inventory has %d occurrences, independent walk has %d", len(inv.Occurrences), len(walk))
	}
	for i, occurrence := range inv.Occurrences {
		expected := walk[i]
		if occurrence.Kind.Name() != expected.goType {
			t.Errorf("preorder %d: kind %s, independent walk has %s", i, occurrence.Kind, expected.goType)
		}
		if occurrence.Span.Start.Offset != expected.start || occurrence.Span.End.Offset != expected.end {
			t.Errorf("preorder %d (%s): span %d-%d, independent walk has %d-%d", i, occurrence.Kind,
				occurrence.Span.Start.Offset, occurrence.Span.End.Offset, expected.start, expected.end)
		}
		var wantParent identity.OccurrenceID
		if expected.parent >= 0 {
			wantParent = inv.Occurrences[expected.parent].ID
		}
		if occurrence.Parent != wantParent {
			t.Errorf("preorder %d (%s): parent %q, independent walk has %q", i, occurrence.Kind,
				occurrence.Parent, wantParent)
		}
	}
}

// TestBuildInventoryInvariants proves the structural invariants: the root is
// the file with no parent, canonical IDs are unique, and spans are well formed.
func TestBuildInventoryInvariants(t *testing.T) {
	src := loadSource(t, reconcileFixture)
	inv, err := BuildInventory(src)
	if err != nil {
		t.Fatalf("BuildInventory: %v", err)
	}
	if len(inv.Occurrences) == 0 {
		t.Fatal("inventory is empty")
	}
	if inv.File == "" {
		t.Error("inventory has no canonical file identity")
	}
	root := inv.Occurrences[0]
	if root.Kind != catalog.KindFile {
		t.Errorf("root kind = %s, want File", root.Kind)
	}
	if root.Parent != "" {
		t.Errorf("root has parent %q, want none", root.Parent)
	}
	ids := map[identity.OccurrenceID]bool{}
	for _, occurrence := range inv.Occurrences {
		ids[occurrence.ID] = true
		if occurrence.Span.Start.Offset > occurrence.Span.End.Offset {
			t.Errorf("occurrence %s span start offset exceeds end", occurrence.Kind)
		}
	}
	if len(ids) != len(inv.Occurrences) {
		t.Errorf("canonical IDs not unique: %d ids for %d occurrences", len(ids), len(inv.Occurrences))
	}
}

// TestOccurrenceIdentityIsMachineIndependent is the regression for
// machine-local identity: identical content loaded from two different
// directories yields identical file and occurrence identities, and no identity
// embeds the load directory.
func TestOccurrenceIdentityIsMachineIndependent(t *testing.T) {
	load := func() (Inventory, string) {
		dir := t.TempDir()
		path := filepath.Join(dir, "fixture.go")
		if err := os.WriteFile(path, []byte(reconcileFixture), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		src, err := source.Load(dir, path)
		if err != nil {
			t.Fatalf("source.Load: %v", err)
		}
		inv, err := BuildInventory(src)
		if err != nil {
			t.Fatalf("BuildInventory: %v", err)
		}
		return inv, dir
	}
	a, dirA := load()
	b, dirB := load()
	if dirA == dirB {
		t.Fatal("test directories must differ")
	}
	if a.File != b.File {
		t.Fatalf("file identity differs across directories: %q vs %q", a.File, b.File)
	}
	if len(a.Occurrences) != len(b.Occurrences) {
		t.Fatalf("occurrence counts differ: %d vs %d", len(a.Occurrences), len(b.Occurrences))
	}
	for i := range a.Occurrences {
		if a.Occurrences[i].ID != b.Occurrences[i].ID {
			t.Fatalf("occurrence %d identity differs: %q vs %q", i, a.Occurrences[i].ID, b.Occurrences[i].ID)
		}
		if a.Occurrences[i].Parent != b.Occurrences[i].Parent {
			t.Fatalf("occurrence %d parent differs", i)
		}
		if strings.Contains(string(a.Occurrences[i].ID), dirA) {
			t.Fatalf("occurrence identity %q embeds the load directory", a.Occurrences[i].ID)
		}
	}
}

// lineDirectiveFixture adjusts display positions with a //line directive.
const lineDirectiveFixture = `package p

//line generated.go:100
func F() int { return 1 }
`

// TestDisplayIsSeparateFromIdentity proves //line adjustments reach only the
// display span: physical spans and identities are unaffected.
func TestDisplayIsSeparateFromIdentity(t *testing.T) {
	src := loadSource(t, lineDirectiveFixture)
	inv, err := BuildInventory(src)
	if err != nil {
		t.Fatalf("BuildInventory: %v", err)
	}
	var fn *Occurrence
	for i := range inv.Occurrences {
		if inv.Occurrences[i].Kind == catalog.KindFuncDecl {
			fn = &inv.Occurrences[i]
			break
		}
	}
	if fn == nil {
		t.Fatal("no FuncDecl occurrence found")
	}
	if got := filepath.Base(fn.Display.Filename); got != "generated.go" {
		t.Errorf("display filename = %q, want generated.go", got)
	}
	if fn.Display.Start.Line != 100 {
		t.Errorf("display line = %d, want 100", fn.Display.Start.Line)
	}
	if fn.Span.Start.Line != 4 {
		t.Errorf("physical line = %d, want 4", fn.Span.Start.Line)
	}
	if strings.Contains(string(fn.ID), "generated.go") {
		t.Errorf("identity %q embeds the display filename", fn.ID)
	}
	if !strings.HasPrefix(string(fn.ID), "fixture.go#") {
		t.Errorf("identity %q does not begin with the physical file identity", fn.ID)
	}
}

// TestInventoryCountsAreProjection proves counts are a faithful projection of
// the authoritative occurrences, sorted and summing to the total.
func TestInventoryCountsAreProjection(t *testing.T) {
	src := loadSource(t, reconcileFixture)
	inv, err := BuildInventory(src)
	if err != nil {
		t.Fatalf("BuildInventory: %v", err)
	}
	total := 0
	prev := ""
	for i, count := range inv.CountsByKind() {
		if count.Count <= 0 {
			t.Errorf("kind %s has non-positive count %d", count.Kind, count.Count)
		}
		if i > 0 && prev >= count.Kind.Name() {
			t.Errorf("counts not strictly sorted by name at %d", i)
		}
		prev = count.Kind.Name()
		total += count.Count
	}
	if total != len(inv.Occurrences) {
		t.Errorf("projected counts sum to %d, want %d occurrences", total, len(inv.Occurrences))
	}
}

// TestVisitorRejectsNonActiveConstructs proves the catalog disposition is the
// single admission policy: deprecated and recovery kinds produce a typed
// ConstructError carrying the catalog's own disposition, and nothing is
// recorded.
func TestVisitorRejectsNonActiveConstructs(t *testing.T) {
	fileID, err := identity.NewFileID("synthetic.go")
	if err != nil {
		t.Fatalf("NewFileID: %v", err)
	}
	fset := token.NewFileSet()
	base := fset.AddFile("synthetic.go", -1, 100)
	pos, end := base.Pos(0), base.Pos(10)
	cases := []ast.Node{
		&ast.Package{},
		&ast.BadExpr{From: pos, To: end},
		&ast.BadStmt{From: pos, To: end},
		&ast.BadDecl{From: pos, To: end},
	}
	for _, node := range cases {
		v := &visitor{fset: fset, file: fileID}
		_, err := v.occurrence(node, "")
		if err == nil {
			t.Errorf("%T was admitted", node)
			continue
		}
		var rejected *ConstructError
		if !errors.As(err, &rejected) {
			t.Errorf("%T error = %T, want *ConstructError", node, err)
			continue
		}
		if rejected.Disposition != rejected.Kind.Disposition() {
			t.Errorf("%T rejection carries disposition %s, catalog says %s",
				node, rejected.Disposition, rejected.Kind.Disposition())
		}
		if rejected.File != fileID {
			t.Errorf("%T rejection carries file %q, want %q", node, rejected.File, fileID)
		}
		if len(v.occurrences) != 0 {
			t.Errorf("%T was recorded despite rejection", node)
		}
	}
}
