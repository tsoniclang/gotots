package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// moduleUnits maps every module-owned unit's display name to its identity and
// kind, and childParent maps each referenced child unit to its parent owner
// string and grammatical edge — the structure the conservation join certifies.
func moduleUnits(t *testing.T, insp *Inspection) (byName map[string]identity.SourceUnitID, kind map[identity.SourceUnitID]identity.UnitKind) {
	t.Helper()
	byName = map[string]identity.SourceUnitID{}
	kind = map[identity.SourceUnitID]identity.UnitKind{}
	for _, pkg := range insp.Workspace().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, file := range pkg.Files() {
			for _, u := range file.Units() {
				byName[u.Name()] = u.ID()
				kind[u.ID()] = u.Kind()
			}
		}
	}
	return byName, kind
}

type edgeTo struct {
	parent string
	edge   string
	ord    int
}

func childEdges(t *testing.T, insp *Inspection) map[string]edgeTo {
	t.Helper()
	_, refs, _ := regionModel(t, insp)
	out := map[string]edgeTo{}
	for _, r := range refs {
		out[r.Child().Source().String()] = edgeTo{parent: r.Parent().String(), edge: r.Edge().String(), ord: r.Ordinal()}
	}
	return out
}

// TestFixtureMatrixConservation certifies the depth/nesting/initializer shapes
// of the reference model. Each fixture is inspected under the default (all-full)
// contract, so the exact site<->reference conservation join runs inside
// VerifyInventory; the test then asserts the precise reference topology, so a
// silent structural drift would fail either the join or the assertion.
func TestFixtureMatrixConservation(t *testing.T) {
	t.Run("three-level nesting", func(t *testing.T) {
		dir := writeFixture(t, map[string]string{
			"go.mod": "module m3.example/m\n\ngo 1.26\n",
			"main.go": "package m\n\nfunc Outer() {\n" +
				"\tmid := func() {\n\t\tinner := func() int { return 1 }\n\t\t_ = inner\n\t}\n\t_ = mid\n}\n",
		})
		insp, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
		if err != nil {
			t.Fatal(err)
		}
		byName, kind := moduleUnits(t, insp)
		edges := childEdges(t, insp)

		outer := byName["Outer"]
		if kind[outer] != identity.UnitFuncBody {
			t.Fatalf("Outer kind = %v", kind[outer])
		}
		// Outer is referenced from the file declaration region.
		if e := edges[outer.String()]; e.parent[:5] != "decl:" {
			t.Errorf("Outer parent = %s, want the file declaration region", e.parent)
		}
		// Exactly one funclit hangs off Outer's body (mid); exactly one hangs
		// off that funclit (inner) — a depth-three chain.
		outerChildren, midOfOuter := 0, ""
		for id, e := range edges {
			if e.parent == "unit:"+outer.String() && kind[mustID(t, byName, id)] == identity.UnitFuncLitBody {
				outerChildren++
				midOfOuter = id
			}
		}
		if outerChildren != 1 {
			t.Fatalf("Outer body references %d funclits, want 1 (mid)", outerChildren)
		}
		midChildren := 0
		for _, e := range edges {
			if e.parent == "unit:"+midOfOuter {
				midChildren++
			}
		}
		if midChildren != 1 {
			t.Errorf("mid body references %d units, want 1 (inner) — the depth-three chain is broken", midChildren)
		}
	})

	t.Run("two literals on one line", func(t *testing.T) {
		dir := writeFixture(t, map[string]string{
			"go.mod": "module m2.example/m\n\ngo 1.26\n",
			"main.go": "package m\n\nfunc Pair() {\n" +
				"\tf, g := func() {}, func() {}\n\t_, _ = f, g\n}\n",
		})
		insp, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
		if err != nil {
			t.Fatal(err)
		}
		byName, kind := moduleUnits(t, insp)
		edges := childEdges(t, insp)
		pair := byName["Pair"]

		var lits []edgeTo
		for id, e := range edges {
			if e.parent == "unit:"+pair.String() && kind[mustID(t, byName, id)] == identity.UnitFuncLitBody {
				lits = append(lits, e)
			}
		}
		if len(lits) != 2 {
			t.Fatalf("Pair body references %d funclits, want 2 on one line", len(lits))
		}
		// Both share the same edge but have distinct source ordinals — ordinal
		// disambiguation on a single line.
		if lits[0].edge != lits[1].edge {
			t.Errorf("two-literal edges differ: %s vs %s", lits[0].edge, lits[1].edge)
		}
		if lits[0].ord == lits[1].ord {
			t.Errorf("two literals on one line share ordinal %d — ordinal disambiguation failed", lits[0].ord)
		}
	})

	t.Run("literal in initializer", func(t *testing.T) {
		dir := writeFixture(t, map[string]string{
			"go.mod":  "module mi.example/m\n\ngo 1.26\n",
			"main.go": "package m\n\nvar Handler = func() int { return 7 }\n",
		})
		insp, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
		if err != nil {
			t.Fatal(err)
		}
		byName, kind := moduleUnits(t, insp)
		edges := childEdges(t, insp)

		handler := byName["Handler"]
		if kind[handler] != identity.UnitVarInitializer {
			t.Fatalf("Handler kind = %v, want var-initializer", kind[handler])
		}
		// The package-level initializer is a unit referenced from the file
		// declaration region through the value-spec edge.
		if e := edges[handler.String()]; e.parent[:5] != "decl:" || e.edge != "GenDecl.Specs" {
			t.Errorf("Handler ref = %+v, want a file-decl GenDecl.Specs edge", e)
		}
		// The nested literal is referenced from inside the initializer's region.
		nested := 0
		for _, e := range edges {
			if e.parent == "unit:"+handler.String() {
				nested++
			}
		}
		if nested != 1 {
			t.Errorf("initializer region references %d nested units, want 1 (the literal)", nested)
		}
	})
}

func mustID(t *testing.T, byName map[string]identity.SourceUnitID, s string) identity.SourceUnitID {
	t.Helper()
	for _, id := range byName {
		if id.String() == s {
			return id
		}
	}
	t.Fatalf("unit id %s not found in the module census", s)
	return identity.SourceUnitID{}
}
