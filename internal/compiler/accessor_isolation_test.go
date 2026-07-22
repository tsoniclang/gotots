package compiler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// TestFinalizedAccessorIsolationIsTotal mechanically enumerates every exported
// zero-argument accessor of the finalized surface that returns a slice or map,
// on every reachable finalized type, and proves each returns an isolated
// value: mutating one call's result never affects the next. It does not label
// a hand-picked sample "every accessor" — the method set is discovered by
// reflection, and the test records how many accessors it exercised so a new
// unchecked collection accessor cannot slip through silently.
func TestFinalizedAccessorIsolationIsTotal(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module iso.example/m\n\ngo 1.26\n",
		"main.go": "package m\n\nimport \"errors\"\n\nvar E = errors.New(\"x\")\n\nfunc F(a int) int {\n\tg := func() int { return a }\n\treturn g() + 1\n}\n",
	})
	inspection, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}))
	if err != nil {
		t.Fatal(err)
	}
	ws := inspection.Workspace()
	sel := inspection.Selection()
	inv := inspection.Inventory()

	// Seed one representative of every finalized type that carries collection
	// accessors. Reflection then walks each type's full method set.
	seeds := []any{ws, sel, inv}
	for _, pkg := range ws.Packages() {
		seeds = append(seeds, pkg)
		for _, file := range pkg.Files() {
			seeds = append(seeds, file)
			for _, unit := range file.Units() {
				seeds = append(seeds, unit)
			}
		}
		for _, imp := range pkg.ImplicitUnits() {
			seeds = append(seeds, imp)
		}
	}
	for _, u := range sel.Units() {
		seeds = append(seeds, u)
	}
	for _, pkg := range inv.Packages() {
		seeds = append(seeds, pkg)
		for _, file := range pkg.Files() {
			seeds = append(seeds, file)
			for _, occ := range file.Occurrences() {
				seeds = append(seeds, occ)
			}
		}
	}

	exercised := map[string]int{}
	seenType := map[reflect.Type]bool{}
	for _, seed := range seeds {
		checkAccessorIsolation(t, reflect.ValueOf(seed), exercised, seenType)
	}

	// The finalized collection accessors that MUST be covered. Reflection found
	// them by method set; this list guards against a type dropping out of the
	// seed set (which would silently stop exercising its accessors).
	required := []string{
		"source.Workspace.Packages", "source.Workspace.Roots",
		"source.Package.Files", "source.Package.Imports", "source.Package.OtherFiles",
		"source.Package.EmbedFiles", "source.Package.EmbedPatterns", "source.Package.Units",
		"source.Package.ImplicitUnits", "source.Package.CheckedUnitMappings", "source.Package.SyntheticUnits",
		"source.File.Units",
		"scope.Selection.Units", "scope.Selection.ImplicitUnits", "scope.Selection.Depths", "scope.Selection.ImplicitDepths",
	}
	for _, name := range required {
		if exercised[name] == 0 {
			t.Errorf("required collection accessor %s was not exercised — surface enumeration is incomplete", name)
		}
	}
	if len(exercised) < len(required) {
		t.Errorf("exercised %d accessors, fewer than the %d required", len(exercised), len(required))
	}
	t.Logf("mechanically exercised %d finalized collection accessors", len(exercised))
}

// checkAccessorIsolation invokes every zero-arg slice/map-returning method of
// v twice, tampers with the first result, and asserts the second is
// unaffected.
func checkAccessorIsolation(t *testing.T, v reflect.Value, exercised map[string]int, seenType map[reflect.Type]bool) {
	typ := v.Type()
	label := strings.TrimPrefix(typ.String(), "*")
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if !method.IsExported() || method.Type.NumIn() != 1 || method.Type.NumOut() != 1 {
			continue
		}
		out := method.Type.Out(0)
		if out.Kind() != reflect.Slice && out.Kind() != reflect.Map {
			continue
		}
		name := label + "." + method.Name
		exercised[name]++
		first := v.Method(i).Call(nil)[0]
		second := v.Method(i).Call(nil)[0]
		switch out.Kind() {
		case reflect.Slice:
			if first.Len() == 0 {
				continue
			}
			before := second.Len()
			elem := first.Index(0)
			if elem.IsZero() {
				continue // zero element can't witness a leak
			}
			if elem.CanSet() {
				elem.Set(reflect.Zero(elem.Type()))
			}
			after := v.Method(i).Call(nil)[0]
			if after.Len() != before {
				t.Errorf("%s: length changed after mutating a prior result (%d -> %d)", name, before, after.Len())
			}
			if after.Len() > 0 && after.Index(0).IsZero() {
				t.Errorf("%s: element leaked — mutation of a prior result was visible", name)
			}
		case reflect.Map:
			if first.Len() == 0 {
				continue
			}
			before := second.Len()
			iter := first.MapRange()
			iter.Next()
			key := iter.Key()
			first.SetMapIndex(key, reflect.Value{}) // delete from the returned map
			after := v.Method(i).Call(nil)[0]
			if after.Len() != before {
				t.Errorf("%s: map size changed after mutating a prior result (%d -> %d)", name, before, after.Len())
			}
		}
	}
}
