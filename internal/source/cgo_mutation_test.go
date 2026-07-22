package source

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

// TestDuplicateCgoOriginRejected demonstrates the "dup cgo origin" mutation
// fails at its owning gate: the origin cross-check's counterpart index refuses
// two checked elements at the same //line-adjusted origin position. A duplicated
// origin mapping — two checked units claiming one source position — is rejected
// with the exact shared position, so it can never seal.
func TestDuplicateCgoOriginRejected(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", "package p\n\nfunc a() {}\n", parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	u := &Universe{fset: fset, request: Request{Dir: "t"}}
	fn := file.Decls[0].(*ast.FuncDecl)
	index := map[originPosKey]ast.Node{}
	var msg string
	fail := func(r string) error { msg = r; return &LoadError{Reason: r} }

	if err := indexCounterpart(u, index, fn.Body, fn, identity.UnitFuncBody, fail); err != nil {
		t.Fatalf("first counterpart index: %v", err)
	}
	// A second checked element sharing the exact origin position is the
	// duplicated cgo origin — the gate rejects it.
	if err := indexCounterpart(u, index, fn.Body, fn, identity.UnitFuncBody, fail); err == nil {
		t.Fatal("duplicate cgo origin position accepted")
	}
	if !strings.Contains(msg, "share origin position") {
		t.Errorf("failure %q does not name the shared origin position", msg)
	}
}

// TestSyntheticJoinRejectsNameOnly demonstrates the "name-only synthetic join"
// mutation is defeated at its owning gate: the cgo synthetic identity is the
// full package|name|role key, so three synthetics that share a declared name
// but differ in role stay distinct. A name-only join — the mutation — collapses
// all three into one bucket, wrongly merging distinct synthetics; the role
// component is exactly what prevents that collision.
func TestSyntheticJoinRejectsNameOnly(t *testing.T) {
	pkg, err := identity.NewPackageID(identity.StandardLibraryOwner(), "os")
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{
		syntheticKey(pkg, "Handle", SyntheticAdapter),
		syntheticKey(pkg, "Handle", SyntheticTypeDecl),
		syntheticKey(pkg, "Handle", SyntheticData),
	}
	distinct := map[string]bool{}
	for _, k := range keys {
		distinct[k] = true
	}
	if len(distinct) != 3 {
		t.Fatalf("full synthetic key collapsed %d roles of one name to %d distinct identities", len(keys), len(distinct))
	}
	nameOnly := map[string]int{}
	for _, k := range keys {
		parts := strings.Split(k, "|")
		if len(parts) != 3 {
			t.Fatalf("synthetic key %q is not package|name|role", k)
		}
		nameOnly[parts[1]]++
	}
	if len(nameOnly) != 1 || nameOnly["Handle"] != 3 {
		t.Fatalf("name-only join did not collide three distinct synthetics: %v", nameOnly)
	}
}
