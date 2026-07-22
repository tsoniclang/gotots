package compiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

func writeFixture(t *testing.T, files map[string]string) string {
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

// TestDeepNestedFunctionLiterals proves the pre-scope ledger is total over
// arbitrarily nested literals: Outer + literal + literal-in-literal +
// initializer literal = five independently selected units.
func TestDeepNestedFunctionLiterals(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod": "module deep.example/fl\n\ngo 1.26\n",
		"main.go": `package fl

var Init = func() int { return 7 }

func Outer() func() func() int {
	middle := func() func() int {
		inner := func() int { return 1 }
		return inner
	}
	return middle
}
`,
	})
	inspection, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	kinds := map[identity.UnitKind]int{}
	for _, pkg := range inspection.Workspace().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, unit := range pkg.Units() {
			kinds[unit.Kind()]++
			if unit.Depth() != source.DepthFullSemantic {
				t.Errorf("unit %s depth = %s", unit.ID(), unit.Depth())
			}
		}
	}
	if kinds[identity.UnitFuncBody] != 1 || kinds[identity.UnitFuncLitBody] != 3 || kinds[identity.UnitVarInitializer] != 1 {
		t.Errorf("census = %v, want 1 func body + 3 literals + 1 initializer", kinds)
	}
}

// TestAuditDriftBattery proves the content-addressed audit rejects every
// tamper class: zeroed counts, per-file byte drift, duplicate rows, and
// build-context drift — while the untouched artifact verifies.
func TestAuditDriftBattery(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module drift.example/m\n\ngo 1.26\n",
		"main.go": "package m\n\nimport \"errors\"\n\nvar E = errors.New(\"x\")\n",
	})
	req := source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}
	artifact, err := AuditCatalog(req)
	if err != nil {
		t.Fatalf("AuditCatalog: %v", err)
	}
	inspection, err := InspectConstructs(req)
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	write := func(a analyze.AuditArtifact) string {
		path := filepath.Join(t.TempDir(), "audit.json")
		if err := analyze.WriteAuditArtifact(&a, path); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if err := VerifyAuditArtifact(inspection, req, write(*artifact)); err != nil {
		t.Fatalf("untouched artifact rejected: %v", err)
	}
	zeroed := *artifact
	zeroed.Files = append([]analyze.AuditFile(nil), artifact.Files...)
	for i := range zeroed.Files {
		zeroed.Files[i].Occurrences = 0
	}
	zeroed.Occurrences = 0
	if err := VerifyAuditArtifact(inspection, req, write(zeroed)); err == nil {
		t.Error("zeroed counts verified")
	}
	drifted := *artifact
	drifted.Files = append([]analyze.AuditFile(nil), artifact.Files...)
	drifted.Files[0].Digest = "0000000000000000000000000000000000000000000000000000000000000000"
	if err := VerifyAuditArtifact(inspection, req, write(drifted)); err == nil {
		t.Error("byte-drifted record verified")
	}
	duplicated := *artifact
	duplicated.Files = append(append([]analyze.AuditFile(nil), artifact.Files...), artifact.Files[0])
	if err := VerifyAuditArtifact(inspection, req, write(duplicated)); err == nil {
		t.Error("duplicate record verified")
	}
	// Context drift: same artifact consumed under different build flags.
	driftedReq := req
	driftedReq.BuildFlags = []string{"-tags=other"}
	driftedInspection, err := InspectConstructs(driftedReq)
	if err != nil {
		t.Fatalf("drifted inspection: %v", err)
	}
	if err := VerifyAuditArtifact(driftedInspection, driftedReq, write(*artifact)); err == nil {
		t.Error("artifact verified under drifted build context")
	}
}

// TestFinalizedArtifactAccessorImmutability proves mutation attempts through
// every collection accessor leave the artifacts unchanged.
func TestFinalizedArtifactAccessorImmutability(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module immut.example/m\n\ngo 1.26\n",
		"main.go": "package m\n\nfunc F(a int) int { return a + 1 }\n",
	})
	inspection, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
	if err != nil {
		t.Fatal(err)
	}
	ws := inspection.Workspace()
	packagesBefore := len(ws.Packages())
	mutated := ws.Packages()
	mutated[0] = nil
	if got := ws.Packages(); len(got) != packagesBefore || got[0] == nil {
		t.Error("workspace package collection mutated through accessor")
	}
	pkg := ws.Roots()[0]
	filesBefore := len(pkg.Files())
	files := pkg.Files()
	files[0] = nil
	if got := pkg.Files(); len(got) != filesBefore || got[0] == nil {
		t.Error("package file collection mutated through accessor")
	}
	units := pkg.Units()
	if len(units) > 0 {
		units[0] = source.SourceUnit{}
		if pkg.Units()[0].ID().IsZero() {
			t.Error("unit collection mutated through accessor")
		}
	}
	imports := pkg.Imports()
	if len(imports) > 0 {
		imports[0] = "tampered"
		if pkg.Imports()[0] == "tampered" {
			t.Error("import collection mutated through accessor")
		}
	}
	inv := inspection.Inventory().Packages()[0].Files()[0]
	occurrencesBefore := len(inv.Occurrences())
	occurrences := inv.Occurrences()
	occurrences[0] = analyze.Occurrence{}
	if got := inv.Occurrences(); len(got) != occurrencesBefore || got[0].ID().IsZero() {
		t.Error("occurrence collection mutated through accessor")
	}
}
