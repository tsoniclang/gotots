package compiler

import (
	"errors"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
	"github.com/tsoniclang/gotots/internal/stagecheck"
)

// auditWorkspaceAndArtifact reproduces the audit producer's finalized workspace
// and artifact, so a test can tamper with the artifact's embedded provider
// graph and re-run the certification gate the producer runs.
func auditWorkspaceAndArtifact(t *testing.T, req source.Request) (*source.Workspace, *analyze.AuditArtifact) {
	t.Helper()
	contract, err := scope.ResolveContract(req.ProviderContract, req.ProviderContractDigest, req.ProviderContractArtifact)
	if err != nil {
		t.Fatal(err)
	}
	auditPolicy, err := contract.AuditAcquisitionPolicy()
	if err != nil {
		t.Fatal(err)
	}
	ordinaryPolicy, err := contract.AcquisitionPolicy()
	if err != nil {
		t.Fatal(err)
	}
	universe, err := source.LoadUniverse(req, auditPolicy, source.UnitManifest{})
	if err != nil {
		t.Fatal(err)
	}
	selection, err := scope.Select(universe, contract)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := selectionProjection(selection)
	if err != nil {
		t.Fatal(err)
	}
	ws, err := source.Finalize(universe, selection.Depths(), selection.ImplicitDepths(), projection)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := analyze.ExtractProviderGraph(universe)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := analyze.AuditCatalog(ws, auditMeta(req, contract), req.Overlay, ordinaryPolicy, graph)
	if err != nil {
		t.Fatal(err)
	}
	return ws, artifact
}

func cloneRecord(r analyze.AuditFile) analyze.AuditFile {
	c := r
	c.Units = append([]analyze.ManifestUnit(nil), r.Units...)
	c.Definitions = append([]analyze.ManifestDefinition(nil), r.Definitions...)
	c.References = append([]analyze.ManifestReference(nil), r.References...)
	return c
}

// TestProviderGraphMutationsFailAtGate mechanically demonstrates that every
// named provider-graph mutation fails at its owning gate — the independent
// certification join at artifact production — with exact, one-sided
// differences. Each case tampers with the embedded graph of one produced
// provider record and re-runs VerifyProviderGraph, which is the gate the audit
// producer runs before the artifact digest is trusted.
func TestProviderGraphMutationsFailAtGate(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module mut.example/m\n\ngo 1.26\n",
		"main.go": "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(hi()) }\n\nfunc hi() string { return \"x\" }\n",
	})
	ws, artifact := auditWorkspaceAndArtifact(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})

	// The clean artifact certifies.
	if err := stagecheck.VerifyProviderGraph(ws, artifact.Files, nil); err != nil {
		t.Fatalf("clean provider graph failed certification: %v", err)
	}

	// Pick one provider file rich enough to mutate distinctly.
	base := analyze.AuditFile{}
	found := false
	for _, f := range artifact.Files {
		if len(f.References) >= 1 && len(f.Definitions) >= 1 {
			base = f
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no provider file with a reference and a definition to mutate")
	}

	// verifyFails runs the gate against a single tampered record and asserts a
	// typed provider-graph failure whose message contains each exact needle.
	verifyFails := func(t *testing.T, mutated analyze.AuditFile, needles ...string) {
		t.Helper()
		err := stagecheck.VerifyProviderGraph(ws, []analyze.AuditFile{mutated}, nil)
		if err == nil {
			t.Fatal("gate accepted a tampered provider graph")
		}
		var verr *stagecheck.VerificationError
		if !errors.As(err, &verr) {
			t.Fatalf("error = %T, want *stagecheck.VerificationError", err)
		}
		if verr.Stage != "provider-graph" {
			t.Errorf("failed at stage %q, want provider-graph", verr.Stage)
		}
		for _, needle := range needles {
			if !strings.Contains(err.Error(), needle) {
				t.Errorf("gate message missing %q; got: %v", needle, err)
			}
		}
	}

	t.Run("mis-parent ref", func(t *testing.T) {
		m := cloneRecord(base)
		realParent := m.References[0].Parent
		m.References[0].Parent = "unit:mut.example/m::forged#0-0/func-body"
		// The forged parent has no derived support; the real parent's reference
		// is now missing from the artifact — both one-sided differences appear.
		verifyFails(t,
			m,
			"artifact reference {owner=unit:mut.example/m::forged#0-0/func-body",
			"derived reference {owner="+realParent,
		)
	})

	t.Run("change edge role", func(t *testing.T) {
		m := cloneRecord(base)
		realEdge := m.References[0].Edge
		m.References[0].Edge = "CallExpr.Fun"
		verifyFails(t, m, "edge=CallExpr.Fun", "edge="+realEdge)
	})

	t.Run("change order", func(t *testing.T) {
		m := cloneRecord(base)
		m.References[0].Ordinal += 100
		verifyFails(t, m, "ord=", "artifact has 0")
	})

	t.Run("omit reference", func(t *testing.T) {
		m := cloneRecord(base)
		dropped := m.References[0]
		m.References = m.References[1:]
		verifyFails(t, m, "derived reference {owner="+dropped.Parent, "artifact has 0")
	})

	t.Run("dup definition", func(t *testing.T) {
		m := cloneRecord(base)
		m.Definitions = append(m.Definitions, m.Definitions[0])
		verifyFails(t, m,
			"artifact definition {unit="+m.Definitions[0].Unit,
			"x2, derived 1",
		)
	})

	t.Run("expose backing storage", func(t *testing.T) {
		m := cloneRecord(base)
		// A fabricated definition — backing storage surfaced as a spurious unit —
		// has no independent derivation from source.
		m.Definitions = append(m.Definitions, analyze.ManifestDefinition{
			Unit: "mut.example/m::backing-storage#0-0/func-body", Kind: 1,
		})
		verifyFails(t,
			m,
			"artifact definition {unit=mut.example/m::backing-storage#0-0/func-body",
			"derived 0",
		)
	})
}
