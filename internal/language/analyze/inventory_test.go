package analyze

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"

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

func loadFixture(t *testing.T) *source.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.go")
	if err := os.WriteFile(path, []byte(reconcileFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	loaded, err := source.Load(path)
	if err != nil {
		t.Fatalf("source.Load: %v", err)
	}
	return loaded
}

// TestBuildInventoryMatchesWalk proves the parent-directed visitor reaches
// exactly the nodes a standard toolchain walk reaches: a missed or extra child
// edge changes the occurrence count. This is the reconciliation of children().
func TestBuildInventoryMatchesWalk(t *testing.T) {
	src := loadFixture(t)
	inv, err := BuildInventory(src)
	if err != nil {
		t.Fatalf("BuildInventory: %v", err)
	}
	walked := 0
	ast.Inspect(src.Syntax, func(n ast.Node) bool {
		if n != nil {
			walked++
		}
		return true
	})
	if len(inv.Occurrences) != walked {
		t.Errorf("inventory recorded %d occurrences, standard walk visited %d", len(inv.Occurrences), walked)
	}
}

// TestBuildInventoryParentEdges proves the per-occurrence records are the
// authoritative evidence: the root is the file with no parent, every other
// occurrence names an existing parent, spans are well formed, and canonical IDs
// are unique.
func TestBuildInventoryParentEdges(t *testing.T) {
	src := loadFixture(t)
	inv, err := BuildInventory(src)
	if err != nil {
		t.Fatalf("BuildInventory: %v", err)
	}
	if len(inv.Occurrences) == 0 {
		t.Fatal("inventory is empty")
	}
	root := inv.Occurrences[0]
	if root.Kind != catalog.KindFile {
		t.Errorf("root kind = %s, want File", root.Kind)
	}
	if root.Parent != "" {
		t.Errorf("root has parent %q, want none", root.Parent)
	}
	ids := map[OccurrenceID]bool{}
	for _, occurrence := range inv.Occurrences {
		ids[occurrence.ID] = true
	}
	if len(ids) != len(inv.Occurrences) {
		t.Errorf("canonical IDs not unique: %d ids for %d occurrences", len(ids), len(inv.Occurrences))
	}
	for i, occurrence := range inv.Occurrences {
		if occurrence.Span.Filename == "" {
			t.Errorf("occurrence %s has empty filename", occurrence.Kind)
		}
		if occurrence.Span.Start.Offset > occurrence.Span.End.Offset {
			t.Errorf("occurrence %s span start offset exceeds end", occurrence.Kind)
		}
		if i == 0 {
			continue
		}
		if occurrence.Parent == "" {
			t.Errorf("non-root occurrence %s has no parent", occurrence.Kind)
		}
		if !ids[occurrence.Parent] {
			t.Errorf("occurrence %s references unknown parent %q", occurrence.Kind, occurrence.Parent)
		}
	}
}

// TestInventoryCountsAreProjection proves counts are a faithful projection of
// the authoritative occurrences, sorted and summing to the total.
func TestInventoryCountsAreProjection(t *testing.T) {
	src := loadFixture(t)
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
