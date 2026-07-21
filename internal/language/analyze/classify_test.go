package analyze

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// reconcileFixture exercises a broad span of Go grammatical forms so the
// classifier is reconciled against constructs the toolchain parser actually
// produces. Bad* recovery nodes appear only on parse errors and are covered by
// the catalog but not by this valid fixture.
const reconcileFixture = `// Package p exercises many Go constructs for catalog reconciliation.
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

// TestClassifyReconcilesFixture proves Classify is total over the constructs the
// toolchain parser emits: every node in a broad fixture classifies without a
// fail-closed error, and a required set of distinctive forms is covered.
func TestClassifyReconcilesFixture(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", reconcileFixture, parser.ParseComments)
	if err != nil {
		t.Fatalf("fixture failed to parse: %v", err)
	}
	covered := map[catalog.Kind]bool{}
	var classifyErr error
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil || classifyErr != nil {
			return false
		}
		kind, err := Classify(n)
		if err != nil {
			classifyErr = err
			return false
		}
		if !kind.Valid() {
			t.Errorf("Classify returned invalid kind for %T", n)
		}
		covered[kind] = true
		return true
	})
	if classifyErr != nil {
		t.Fatalf("Classify failed closed on a parser-produced node: %v", classifyErr)
	}
	required := []catalog.Kind{
		catalog.KindFuncDecl, catalog.KindStructType, catalog.KindInterfaceType,
		catalog.KindRangeStmt, catalog.KindTypeSwitchStmt, catalog.KindSelectStmt,
		catalog.KindIndexListExpr, catalog.KindDeferStmt, catalog.KindGoStmt,
		catalog.KindSendStmt, catalog.KindEllipsis, catalog.KindCommentGroup,
		catalog.KindMapType, catalog.KindChanType, catalog.KindSliceExpr,
	}
	for _, kind := range required {
		if !covered[kind] {
			t.Errorf("fixture did not cover required construct %s", kind)
		}
	}
}

// unknownNode is a synthetic ast.Node with no catalog identity. It stands in
// for a construct the classifier does not recognize.
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
		t.Errorf("Classify returned kind %s with an error, want KindInvalid", kind)
	}
}

// TestInspectConstructsReportsFile proves the fail-closed inspection path over a
// real file: it parses, inventories, and returns sorted occurrences.
func TestInspectConstructsReportsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(path, []byte(reconcileFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	inv, err := InspectConstructs(path)
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	if inv.Path != path {
		t.Errorf("inventory path = %q, want %q", inv.Path, path)
	}
	if len(inv.Occurrences) == 0 {
		t.Fatal("inventory has no occurrences")
	}
	for i, occ := range inv.Occurrences {
		if !occ.Kind.Valid() {
			t.Errorf("occurrence %d has invalid kind", i)
		}
		if occ.Count <= 0 {
			t.Errorf("occurrence %s has non-positive count %d", occ.Kind, occ.Count)
		}
		if i > 0 && inv.Occurrences[i-1].Kind.Name() >= occ.Kind.Name() {
			t.Errorf("occurrences not strictly sorted by name at %d", i)
		}
	}
}
