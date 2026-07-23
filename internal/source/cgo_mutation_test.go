package source

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

// writeFiles writes a file tree under a fresh temp dir (internal-test local; the
// external test package's writeTree is not visible here).
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// recursivePolicy is a census policy that recurses every owner class — the
// policy the cgo tests load under, built without importing scope (which would
// create an import cycle for this internal test).
func recursivePolicy(t *testing.T) AcquisitionPolicy {
	t.Helper()
	policy, err := NewAcquisitionPolicy(map[identity.OwnerClass]CensusMode{
		identity.OwnerModule:          CensusRecursive,
		identity.OwnerStandardLibrary: CensusRecursive,
		identity.OwnerToolchain:       CensusRecursive,
		identity.OwnerLanguagePseudo:  CensusRecursive,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

// loadCgoPackage loads a real cgo package and returns it with its universe and
// originals map, so a test can re-derive and tamper the origin graph and drive
// the actual end-to-end verifier. Skips when cgo is unavailable.
func loadCgoPackage(t *testing.T) (*Universe, *LoadedPackage, map[string]*LoadedFile) {
	t.Helper()
	dir := writeFiles(t, map[string]string{
		"go.mod": "module cgo.example/mut\n\ngo 1.26\n",
		"main.go": "package main\n\n/*\n#include <stdlib.h>\n*/\nimport \"C\"\n\n" +
			"func main() { C.free(nil) }\n\nfunc pure() int { return 1 }\n\n" +
			"func hasLit() {\n\tf := func() { C.free(nil) }\n\t_ = f\n}\n",
	})
	u, err := LoadUniverse(Request{Dir: dir, Env: []string{"CGO_ENABLED=1"}}, recursivePolicy(t), UnitManifest{})
	if err != nil {
		t.Skipf("cgo unavailable in this environment: %v", err)
	}
	for _, pkg := range u.packages {
		if len(pkg.checkedDecls) > 0 {
			originals := map[string]*LoadedFile{}
			for _, file := range pkg.files {
				originals[file.path] = file
			}
			return u, pkg, originals
		}
	}
	t.Skip("no cgo package with checked declarations was loaded")
	return nil, nil, nil
}

// TestCgoOriginMutationsFailEndToEnd drives the ACTUAL end-to-end origin
// verifier (verifyOriginGraph) with a re-derived producer graph, tampering it
// per named mutation and asserting each fails with its exact one-sided identity.
// The independent extractor inside verifyOriginGraph is the join partner — the
// same code the loader runs — so these are not standalone string demonstrations.
func TestCgoOriginMutationsFailEndToEnd(t *testing.T) {
	u, pkg, originals := loadCgoPackage(t)

	// The clean re-derived producer verifies end-to-end.
	clean, err := deriveOriginGraph(u, pkg, originals)
	if err != nil {
		t.Fatalf("derive origin graph: %v", err)
	}
	if err := verifyOriginGraph(u, pkg, originals, clean); err != nil {
		t.Fatalf("clean origin graph failed end-to-end verification: %v", err)
	}
	if len(clean.counterparts) == 0 {
		t.Fatal("no cgo counterparts derived — the mutations would be vacuous")
	}

	// oneUnit returns any mapped origin unit and its counterpart.
	var sampleUnit identity.SourceUnitID
	var sampleCP checkedCounterpart
	for id, cp := range clean.counterparts {
		sampleUnit, sampleCP = id, cp
		break
	}

	fails := func(t *testing.T, producer *originGraph, needles ...string) {
		t.Helper()
		err := verifyOriginGraph(u, pkg, originals, producer)
		if err == nil {
			t.Fatal("end-to-end verifier accepted a tampered origin graph")
		}
		for _, needle := range needles {
			if !strings.Contains(err.Error(), needle) {
				t.Errorf("verifier message missing %q; got: %v", needle, err)
			}
		}
	}

	t.Run("missing origin", func(t *testing.T) {
		p := cloneOriginGraph(clean)
		delete(p.counterparts, sampleUnit)
		fails(t, p, "independent extractor maps unit "+sampleUnit.String()+" the producer does not")
	})

	t.Run("extra origin", func(t *testing.T) {
		p := cloneOriginGraph(clean)
		forged, err := identity.ParseSourceUnitID(sampleUnit.String())
		if err != nil {
			t.Fatal(err)
		}
		// A fabricated origin unit the independent extractor never derives.
		fabricated := forgeUnit(t, forged)
		p.counterparts[fabricated] = sampleCP
		fails(t, p, "producer maps unit "+fabricated.String()+" the independent extractor does not")
	})

	t.Run("relocated origin", func(t *testing.T) {
		p := cloneOriginGraph(clean)
		// Point the counterpart at a different node — the topology diverges.
		p.counterparts[sampleUnit] = checkedCounterpart{node: &ast.Ident{Name: "relocated"}, span: sampleCP.span}
		fails(t, p, "counterpart node diverges for unit "+sampleUnit.String())
	})

	t.Run("name-only synthetic join", func(t *testing.T) {
		if len(clean.synthetics) == 0 {
			t.Skip("no synthetics to mutate")
		}
		p := cloneOriginGraph(clean)
		// Keep the NAME, change the ROLE: a name-only comparison would accept
		// this, but the real join keys on package|name|role and rejects it.
		orig := p.synthetics[0]
		altered, err := NewSyntheticUnit(orig.Pkg(), orig.Name(), otherRole(orig.Role()))
		if err != nil {
			t.Fatal(err)
		}
		p.synthetics = append([]SyntheticUnit{altered}, p.synthetics[1:]...)
		// The join keys on package|name|role: the role-altered producer synthetic
		// no longer matches the independent extractor's, so one side is absent.
		// A name-only comparison would have accepted it.
		fails(t, p, "synthetic identity", "absent from the")
	})
}

// TestCgoAmbiguousOriginRejected drives the ambiguity guard the end-to-end
// verifier runs: verifyOriginGraph's independent extractor indexes each checked
// element by its origin position through indexCounterpart, which rejects two
// elements sharing one position. A checked view that mapped two units to a
// single origin — an ambiguous origin — fails here with the exact position, so
// it can never seal.
func TestCgoAmbiguousOriginRejected(t *testing.T) {
	u, pkg, _ := loadCgoPackage(t)
	if len(pkg.checkedDecls) == 0 {
		t.Skip("no checked declarations")
	}
	root := pkg.checkedDecls[0].node
	index := map[originPosKey]ast.Node{}
	var msg string
	fail := func(r string) error { msg = r; return &LoadError{Reason: r} }
	if err := indexCounterpart(u, index, root, root, identity.UnitFuncBody, fail); err != nil {
		t.Fatalf("first index: %v", err)
	}
	// A second checked element at the exact same origin position is the
	// ambiguity — the verifier's extractor rejects it.
	if err := indexCounterpart(u, index, root, root, identity.UnitFuncBody, fail); err == nil {
		t.Fatal("ambiguous origin position accepted")
	}
	if !strings.Contains(msg, "share origin position") {
		t.Errorf("failure %q does not name the shared origin position", msg)
	}
}

// cloneOriginGraph deep-copies the mutable maps/slices a mutation tampers.
func cloneOriginGraph(g *originGraph) *originGraph {
	out := &originGraph{
		counterparts: make(map[identity.SourceUnitID]checkedCounterpart, len(g.counterparts)),
		mappings:     append([]CheckedUnitMapping(nil), g.mappings...),
		synthetics:   append([]SyntheticUnit(nil), g.synthetics...),
		syntheticObj: g.syntheticObj,
	}
	for k, v := range g.counterparts {
		out.counterparts[k] = v
	}
	return out
}

// forgeUnit builds a source unit id at a distinct span so it cannot collide with
// any real origin unit.
func forgeUnit(t *testing.T, like identity.SourceUnitID) identity.SourceUnitID {
	t.Helper()
	span, err := identity.NewSpanID(like.Span().File(), 999001, 999002)
	if err != nil {
		t.Fatal(err)
	}
	id, err := identity.NewSourceUnitID(span, identity.UnitFuncBody)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// otherRole returns a synthetic role distinct from r.
func otherRole(r SyntheticRole) SyntheticRole {
	if r == SyntheticAdapter {
		return SyntheticTypeDecl
	}
	return SyntheticAdapter
}
