package analyze

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// mustContract resolves the default contract artifact (the test's request
// selection).
func mustContract() scope.ProviderContract {
	contract, err := scope.ResolveContract(scope.DefaultContractID, "")
	if err != nil {
		panic(err)
	}
	return contract
}

// fixtureSource exercises every semantic variant, every inventory-owned
// implicit operation, and a broad edge surface. TestVariantCoverageIsTotal
// proves the totality claim against it.
const fixtureSource = `// Package p is the analysis reconciliation fixture.
package p

import "fmt"

//go:generate echo generated
//custom:note external tool directive
type T struct {
	A int
	B []string
}

func (t T) VM() int       { return t.A }
func (t *T) PM(x int) int { t.A = x; return t.A }

type E struct {
	T
	Extra bool
}

type I interface{ VM() int }

type Alias = T

type Fn func(a int, b ...string) int

type Num interface{ ~int | ~float64 }

type Pair[K comparable, V any] struct {
	Key K
	Val V
}

func Identity[X any](x X) X { return x }

func two() (int, string) { return 1, "a" }

func named() (n int) { n = 3; return }

func void() { return }

func seq(yield func(int) bool) { yield(1) }

func F() {
	t := T{A: 1, B: []string{"x"}}
	arr := [3]int{0: 1, 1: 2}
	mp := map[string]int{"a": 1}
	s := []string{"y"}

	_ = t.A
	e := E{T: t}
	_ = e.A
	mv := t.VM
	_ = mv
	me := (*T).PM
	_ = me
	_ = fmt.Sprint(1)
	_ = t.VM()
	_ = len(arr)
	_ = int64(3)
	f := Fn(func(a int, b ...string) int { return a })
	_ = f(1, "a")

	_ = arr[0]
	one := mp["a"]
	_ = one
	v, ok := mp["a"]
	_, _ = v, ok
	var v2, ok2 = mp["a"]
	_, _ = v2, ok2
	iv := Identity[int]
	_ = iv
	pair := Pair[int, string]{Key: 1, Val: "s"}
	_ = pair

	var a any = t
	y := a.(T)
	_ = y
	av, aok := a.(T)
	_, _ = av, aok
	switch a.(type) {
	case T:
	}

	x := 1
	x = 2
	x += 3
	neg := -x
	_ = neg
	b1, s1 := two()
	b1, s1 = two()
	_, _ = b1, s1
	var pre int
	var preOk bool
	pre, preOk = mp["a"]
	_, _ = pre, preOk

	ch := make(chan int, 2)
	ch <- 1
	rv := <-ch
	_ = rv
	cv, cok := <-ch
	_, _ = cv, cok

	var pt *T
	pt = &t
	_ = (*pt).A
	_ = t.PM(4)
	_ = e.VM()

	var i2 I
	i2 = t
	_ = i2.VM()
	var t3 T
	t3 = t
	_ = t3
	g := func(iface I) { _ = iface }
	g(t)
	h := func(val T) { _ = val }
	h(t)

	for k, val := range mp {
		_, _ = k, val
	}
	for i3 := range arr {
		_ = i3
	}
	for range &arr {
	}
	for _, r := range "ab" {
		_ = r
	}
	for _, sv := range s {
		_ = sv
	}
	go func() {
		for cv2 := range ch {
			_ = cv2
		}
	}()
	for n2 := range 3 {
		_ = n2
	}
	for fv := range seq {
		_ = fv
	}

	switch x {
	case 1:
		fallthrough
	default:
	}
	switch {
	case x > 0:
	}

	select {
	case ch <- 2:
	case cs := <-ch:
		_ = cs
	default:
	}

Loop:
	for j := 0; j < 2; j++ {
		if j == 1 {
			continue Loop
		} else {
			break
		}
	}
	defer fmt.Sprint("d")
	_ = named()
	void()
	if p := 1; p > 0 {
		_ = p
	}
	sl3 := s[0:1:1]
	_ = sl3
	goto End
End:
}
`

// loadFinalized runs the full source pipeline under the default contract.
func loadFinalized(req source.Request) (*source.Workspace, error) {
	universe, err := source.LoadUniverse(req)
	if err != nil {
		return nil, err
	}
	selection, err := scope.Select(universe, mustContract())
	if err != nil {
		return nil, err
	}
	return source.Finalize(universe, selection.Depths())
}

var (
	fixtureOnce sync.Once
	fixtureDir  string
	fixtureWS   *source.Workspace
	fixtureInv  *WorkspaceInventory
	fixtureErr  error
)

// fixture loads the shared reconciliation fixture module exactly once.
func fixture(t *testing.T) (*source.Workspace, *WorkspaceInventory) {
	t.Helper()
	fixtureOnce.Do(func() {
		fixtureDir, fixtureErr = os.MkdirTemp("", "gotots-analyze-fixture-*")
		if fixtureErr != nil {
			return
		}
		files := map[string]string{
			"go.mod":     "module fixture.example/p\n\ngo 1.26\n",
			"fixture.go": fixtureSource,
		}
		for rel, content := range files {
			if fixtureErr = os.WriteFile(filepath.Join(fixtureDir, rel), []byte(content), 0o644); fixtureErr != nil {
				return
			}
		}
		fixtureWS, fixtureErr = loadFinalized(source.Request{Dir: fixtureDir})
		if fixtureErr != nil {
			return
		}
		fixtureInv, fixtureErr = BuildWorkspaceInventory(fixtureWS)
	})
	if fixtureErr != nil {
		t.Fatalf("fixture: %v", fixtureErr)
	}
	return fixtureWS, fixtureInv
}

func TestMain(m *testing.M) {
	code := m.Run()
	if fixtureDir != "" {
		os.RemoveAll(fixtureDir)
	}
	os.Exit(code)
}

// fixtureFile is the single file inventory of the shared fixture.
func fixtureFile(t *testing.T) (*source.File, *FileInventory) {
	ws, inv := fixture(t)
	return ws.Packages()[0].Files()[0], inv.Packages()[0].Files()[0]
}

// TestBuildInventoryExactJoinsIndependentWalk joins the inventory to an
// independent toolchain traversal on node identity, exact preorder position,
// and exact parent edge — not counts.
func TestBuildInventoryExactJoinsIndependentWalk(t *testing.T) {
	file, inv := fixtureFile(t)
	syntax, hasFull := file.FullSyntax()
	if !hasFull {
		t.Fatal("fixture file lacks full-syntax evidence")
	}
	type expectation struct {
		goType string
		start  int
		end    int
		parent int
	}
	var walk []expectation
	var stack []int
	ast.Inspect(syntax, func(n ast.Node) bool {
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
			start:  file.Fset().PositionFor(n.Pos(), false).Offset,
			end:    file.Fset().PositionFor(n.End(), false).Offset,
			parent: parent,
		})
		stack = append(stack, len(walk)-1)
		return true
	})
	occurrences := inv.Occurrences()
	if len(occurrences) != len(walk) {
		t.Fatalf("inventory has %d occurrences, independent walk has %d", len(occurrences), len(walk))
	}
	for i, occurrence := range occurrences {
		expected := walk[i]
		if occurrence.Kind().Name() != expected.goType {
			t.Errorf("preorder %d: kind %s, walk has %s", i, occurrence.Kind(), expected.goType)
		}
		if occurrence.Span().Start.Offset != expected.start || occurrence.Span().End.Offset != expected.end {
			t.Errorf("preorder %d (%s): span %d-%d, walk has %d-%d", i, occurrence.Kind(),
				occurrence.Span().Start.Offset, occurrence.Span().End.Offset, expected.start, expected.end)
		}
		var wantParent identity.OccurrenceID
		if expected.parent >= 0 {
			wantParent = occurrences[expected.parent].ID()
		}
		if occurrence.Parent() != wantParent {
			t.Errorf("preorder %d (%s): parent %q, walk has %q", i, occurrence.Kind(), occurrence.Parent(), wantParent)
		}
	}
}

// TestInventoryInvariants proves the artifact invariants: File root without
// parent, unique IDs, valid edges whose parent kinds match, and numeric kind
// identity inside every occurrence ID.
func TestInventoryInvariants(t *testing.T) {
	_, inv := fixtureFile(t)
	occurrences := inv.Occurrences()
	if occurrences[0].Kind() != catalog.KindFile || !occurrences[0].Parent().IsZero() {
		t.Fatal("root is not a parentless File occurrence")
	}
	kindByID := map[identity.OccurrenceID]catalog.Kind{}
	for _, occurrence := range occurrences {
		kindByID[occurrence.ID()] = occurrence.Kind()
	}
	if len(kindByID) != len(occurrences) {
		t.Errorf("IDs not unique: %d ids for %d occurrences", len(kindByID), len(occurrences))
	}
	for i, occurrence := range occurrences {
		if occurrence.ID().KindID() != uint16(occurrence.Kind()) {
			t.Errorf("occurrence %d identity encodes kind %d, want pinned %d",
				i, occurrence.ID().KindID(), uint16(occurrence.Kind()))
		}
		if i == 0 {
			continue
		}
		if !occurrence.Edge().Valid() {
			t.Errorf("occurrence %d (%s) lacks a parent edge", i, occurrence.Kind())
			continue
		}
		if occurrence.Edge().Parent() != kindByID[occurrence.Parent()] {
			t.Errorf("occurrence %d edge %s does not belong to parent kind %s",
				i, occurrence.Edge(), kindByID[occurrence.Parent()])
		}
		if occurrence.Role() != occurrence.Edge().Role() {
			t.Errorf("occurrence %d role diverges from its edge", i)
		}
	}
}

// TestOccurrenceIdentityIsMachineIndependent proves identical module content
// in two directories yields identical identities and no identity embeds the
// load directory.
func TestOccurrenceIdentityIsMachineIndependent(t *testing.T) {
	load := func() (*WorkspaceInventory, string) {
		dir := t.TempDir()
		for rel, content := range map[string]string{
			"go.mod":     "module fixture.example/p\n\ngo 1.26\n",
			"fixture.go": fixtureSource,
		} {
			if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
		ws, err := loadFinalized(source.Request{Dir: dir})
		if err != nil {
			t.Fatalf("LoadWorkspace: %v", err)
		}
		inv, err := BuildWorkspaceInventory(ws)
		if err != nil {
			t.Fatalf("BuildWorkspaceInventory: %v", err)
		}
		return inv, dir
	}
	a, dirA := load()
	b, dirB := load()
	fa, fb := a.Packages()[0].Files()[0], b.Packages()[0].Files()[0]
	if fa.File() != fb.File() {
		t.Fatalf("file identity differs: %q vs %q", fa.File(), fb.File())
	}
	if len(fa.Occurrences()) != len(fb.Occurrences()) {
		t.Fatal("occurrence counts differ")
	}
	for i := range fa.Occurrences() {
		oa, ob := fa.Occurrences()[i], fb.Occurrences()[i]
		if oa.ID() != ob.ID() || oa.Parent() != ob.Parent() {
			t.Fatalf("occurrence %d identity differs across checkouts", i)
		}
		if strings.Contains(oa.ID().String(), dirA) || strings.Contains(oa.ID().String(), dirB) {
			t.Fatalf("identity %q embeds a load directory", oa.ID())
		}
	}
}

// TestDisplayIsSeparateFromIdentity proves //line adjustments reach only the
// display span.
func TestDisplayIsSeparateFromIdentity(t *testing.T) {
	dir := t.TempDir()
	for rel, content := range map[string]string{
		"go.mod":  "module fixture.example/line\n\ngo 1.26\n",
		"main.go": "package p\n\n//line generated.go:100\nfunc F() int { return 1 }\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	ws, err := loadFinalized(source.Request{Dir: dir})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	inv, err := BuildWorkspaceInventory(ws)
	if err != nil {
		t.Fatalf("BuildWorkspaceInventory: %v", err)
	}
	fileInv := inv.Packages()[0].Files()[0]
	var fn *Occurrence
	occurrences := fileInv.Occurrences()
	for i := range occurrences {
		if occurrences[i].Kind() == catalog.KindFuncDecl {
			fn = &occurrences[i]
			break
		}
	}
	if fn == nil {
		t.Fatal("no FuncDecl occurrence")
	}
	if got := filepath.Base(fn.Display().Filename); got != "generated.go" {
		t.Errorf("display filename = %q, want generated.go", got)
	}
	if fn.Display().Start.Line != 100 {
		t.Errorf("display line = %d, want 100", fn.Display().Start.Line)
	}
	if fn.Span().Start.Line != 4 {
		t.Errorf("physical line = %d, want 4", fn.Span().Start.Line)
	}
	if strings.Contains(fn.ID().String(), "generated.go") {
		t.Errorf("identity %q embeds the display filename", fn.ID())
	}
	// A //line record itself joins the directive inventory.
	foundLine := false
	for _, directive := range fileInv.Directives() {
		if directive.Kind() == catalog.DirectiveLine {
			foundLine = true
		}
	}
	if !foundLine {
		t.Error("//line directive not inventoried")
	}
}

// TestInventoryCountsAreProjection proves counts are a faithful sorted
// projection of the authoritative occurrences.
func TestInventoryCountsAreProjection(t *testing.T) {
	_, inv := fixtureFile(t)
	total := 0
	prev := ""
	for i, count := range inv.CountsByKind() {
		if count.Count <= 0 {
			t.Errorf("kind %s has non-positive count", count.Kind)
		}
		if i > 0 && prev >= count.Kind.Name() {
			t.Errorf("counts not sorted at %d", i)
		}
		prev = count.Kind.Name()
		total += count.Count
	}
	if total != len(inv.Occurrences()) {
		t.Errorf("counts sum %d, want %d", total, len(inv.Occurrences()))
	}
}
