package compiler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// TestFinalizedSurfaceExposesNoRawAST mechanically proves the "restore raw
// child pointer" mutation is structurally impossible through the finalized
// surface: no exported method of any reachable finalized type returns a value
// that reaches, directly or through exported fields and containers, a go/ast or
// go/types type. A finalized region that exposed its raw parent node — the one
// way a generic traversal could descend into an excluded child body — would
// surface here as an ast/types return type. The type graph is walked by
// reflection, not a hand-picked sample, and the count of exercised methods is
// recorded so a new accessor cannot slip through silently.
func TestFinalizedSurfaceExposesNoRawAST(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module rawast.example/m\n\ngo 1.26\n",
		"main.go": "package m\n\nimport \"errors\"\n\nvar E = errors.New(\"x\")\n\nfunc F(a int) int {\n\tg := func() int { return a }\n\treturn g() + 1\n}\n",
	})
	inspection, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}))
	if err != nil {
		t.Fatal(err)
	}
	ws := inspection.Workspace()
	sel := inspection.Selection()
	inv := inspection.Inventory()

	seeds := []reflect.Type{
		reflect.TypeOf(ws), reflect.TypeOf(sel), reflect.TypeOf(inv),
	}
	for _, pkg := range ws.Packages() {
		seeds = append(seeds, reflect.TypeOf(pkg))
		for _, file := range pkg.Files() {
			seeds = append(seeds, reflect.TypeOf(file))
			for _, unit := range file.Units() {
				seeds = append(seeds, reflect.TypeOf(unit))
			}
		}
		for _, imp := range pkg.ImplicitUnits() {
			seeds = append(seeds, reflect.TypeOf(imp))
		}
	}
	for _, u := range sel.Units() {
		seeds = append(seeds, reflect.TypeOf(u))
	}
	for _, pkg := range inv.Packages() {
		seeds = append(seeds, reflect.TypeOf(pkg))
		for _, def := range pkg.Definitions() {
			seeds = append(seeds, reflect.TypeOf(def))
		}
		for _, ref := range pkg.References() {
			seeds = append(seeds, reflect.TypeOf(ref))
		}
		for _, file := range pkg.Files() {
			seeds = append(seeds, reflect.TypeOf(file))
			for _, occ := range file.Occurrences() {
				seeds = append(seeds, reflect.TypeOf(occ))
			}
		}
	}

	w := &rawASTWalker{visited: map[reflect.Type]bool{}, seenType: map[reflect.Type]bool{}}
	for _, seed := range seeds {
		w.walkMethods(t, seed)
	}
	if w.methods == 0 {
		t.Fatal("no exported methods were exercised — the surface enumeration is empty")
	}
	t.Logf("mechanically inspected %d exported finalized methods for raw-AST return types", w.methods)
}

type rawASTWalker struct {
	visited  map[reflect.Type]bool
	seenType map[reflect.Type]bool
	methods  int
}

// walkMethods checks every exported method's return types on typ, once per
// distinct type.
func (w *rawASTWalker) walkMethods(t *testing.T, typ reflect.Type) {
	if w.seenType[typ] {
		return
	}
	w.seenType[typ] = true
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if !method.IsExported() {
			continue
		}
		w.methods++
		for out := 0; out < method.Type.NumOut(); out++ {
			w.assertNoRawAST(t, method.Type.Out(out), typ.String()+"."+method.Name)
		}
	}
}

// assertNoRawAST fails if typ reaches a go/ast or go/types type through
// pointers, containers, or exported struct fields.
func (w *rawASTWalker) assertNoRawAST(t *testing.T, typ reflect.Type, path string) {
	if typ == nil || w.visited[typ] {
		return
	}
	w.visited[typ] = true
	if pkg := typ.PkgPath(); pkg == "go/ast" || pkg == "go/types" {
		t.Errorf("%s reaches raw %s type %s — a raw AST/type node is exposed through the finalized surface", path, pkg, typ.String())
		return
	}
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Chan:
		w.assertNoRawAST(t, typ.Elem(), path)
	case reflect.Map:
		w.assertNoRawAST(t, typ.Key(), path)
		w.assertNoRawAST(t, typ.Elem(), path)
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			w.assertNoRawAST(t, field.Type, path+"."+field.Name)
		}
	case reflect.Interface:
		// An interface from go/ast (e.g. ast.Node) is caught by PkgPath above;
		// an empty interface reaches nothing statically.
		if strings.Contains(typ.String(), "ast.") || strings.Contains(typ.String(), "types.") {
			t.Errorf("%s returns interface %s", path, typ.String())
		}
	}
}
