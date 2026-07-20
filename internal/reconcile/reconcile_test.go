package reconcile

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/translate"
)

// fixture builds a clean typed-ledger corpus: two admitted bodies, both
// emitted, one materialized package with its typed disposition.
func fixture() (*census.Result, *translate.Generated) {
	run := &census.Result{Report: &census.Report{Declarations: []census.DeclarationRecord{
		{ID: "p::func::A", Package: "p", Kind: "func", Scope: "production", HasBody: true},
		{ID: "p::func::B", Package: "p", Kind: "func", Scope: "production", HasBody: true},
		{ID: "t::func::T", Package: "t", Kind: "func", Scope: "test", HasBody: true},
	}}}
	generated := &translate.Generated{
		Support: []translate.BodySupport{
			{ID: "p::func::A", Package: "p", Kind: "body", State: "ir-admitted"},
			{ID: "p::func::B", Package: "p", Kind: "body", State: "ir-admitted"},
		},
		Emissions: []emit.EmissionEvent{
			{ID: "p::func::A", Kind: emit.EmissionBody},
			{ID: "p::func::B", Kind: emit.EmissionBody},
		},
		Proofs: []translate.Proof{
			{ID: "p::func::A", Package: "p", GeneratedFile: "core/p/package.ts", LoweredHash: "h1"},
			{ID: "p::func::B", Package: "p", GeneratedFile: "core/p/package.ts", LoweredHash: "h2"},
		},
		ModuleDispositions: []translate.ModuleDisposition{
			{Package: "p", State: "emitted-runtime", Module: "core/p/package.ts"},
		},
		Files:           map[string]string{"core/p/package.ts": "export function A() {}\n"},
		Ownership:       map[string]string{"core/p/package.ts": "generated-core"},
		Withheld:        map[string]string{},
		NotMaterialized: map[string]string{},
	}
	return run, generated
}

// The clean fixture reconciles with zero defects.
func TestCleanJoinHasNoDefects(t *testing.T) {
	run, generated := fixture()
	report := Build("head", run, generated)
	if defects := report.Defects(); len(defects) != 0 {
		t.Fatalf("clean fixture produced defects: %v", defects)
	}
	if report.Denominators[0].Count != 2 {
		t.Fatalf("census bodies denominator: %d", report.Denominators[0].Count)
	}
}

// MUTATION (missing): a census body with no support record is a typed
// defect.
func TestMissingSupportRecordIsDefect(t *testing.T) {
	run, generated := fixture()
	generated.Support = generated.Support[:1]
	report := Build("head", run, generated)
	defects := report.Defects()
	found := false
	for _, d := range defects {
		if d.ID == "p::func::B" && strings.Contains(d.Disposition, "without a support record") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the missing-support defect for B, got %v", defects)
	}
}

// MUTATION (moved-scope): a support record whose census body left
// production scope is a typed defect.
func TestMovedScopeSupportIsDefect(t *testing.T) {
	run, generated := fixture()
	run.Report.Declarations = run.Report.Declarations[:1]
	generated.Support = generated.Support[:1]
	generated.Emissions = generated.Emissions[:1]
	generated.Proofs = generated.Proofs[:1]
	// B's support record stays behind after its declaration left scope.
	generated.Support = append(generated.Support,
		translate.BodySupport{ID: "p::func::B", Package: "p", Kind: "body", State: "ir-admitted"})
	report := Build("head", run, generated)
	defects := report.Defects()
	found := false
	for _, d := range defects {
		if d.ID == "p::func::B" && strings.Contains(d.Disposition, "without a census production body") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the moved-scope defect for B, got %v", defects)
	}
}

// MUTATION (duplicate): the same identity recorded twice in a
// must-be-unique ledger is a typed Duplicate defect — never collapsed
// by a set or map.
func TestDuplicateSupportIdentityIsDefect(t *testing.T) {
	run, generated := fixture()
	generated.Support = append(generated.Support,
		translate.BodySupport{ID: "p::func::A", Package: "p", Kind: "body", State: "ir-admitted"})
	report := Build("head", run, generated)
	if len(report.Duplicates) != 1 || report.Duplicates[0].ID != "p::func::A" || report.Duplicates[0].Count != 2 {
		t.Fatalf("expected the duplicate record for A, got %+v", report.Duplicates)
	}
	found := false
	for _, d := range report.Defects() {
		if d.Defect && strings.Contains(d.Disposition, "duplicate identity") {
			found = true
		}
	}
	if !found {
		t.Fatal("duplicates must surface in Defects()")
	}
}

// MUTATION (orphan/extra): an emission event without a support record
// is a typed defect (untracked emission).
func TestOrphanPlaceholderEmissionIsDefect(t *testing.T) {
	run, generated := fixture()
	generated.Emissions = append(generated.Emissions,
		emit.EmissionEvent{ID: "p::func::Ghost", Kind: emit.EmissionBodyPlaceholder})
	report := Build("head", run, generated)
	found := false
	for _, d := range report.Defects() {
		if d.ID == "p::func::Ghost" && strings.Contains(d.Disposition, "without an unimplemented support record") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the orphan-emission defect, got %v", report.Defects())
	}
}

// Variant re-emissions (one unimplemented identity, several placeholder
// copies) are typed records read from the emission ledger, and the
// surplus copies stay visible in the join.
func TestVariantReemissionsAreTypedFromTheLedger(t *testing.T) {
	run, generated := fixture()
	generated.Support[1].State = "unimplemented"
	generated.Emissions = []emit.EmissionEvent{
		{ID: "p::func::A", Kind: emit.EmissionBody},
		{ID: "p::func::B", Kind: emit.EmissionBodyPlaceholder},
		{ID: "p::func::B", Kind: emit.EmissionBodyPlaceholder},
	}
	generated.Proofs = generated.Proofs[:1]
	report := Build("head", run, generated)
	if len(report.VariantReemissions) != 1 || report.VariantReemissions[0].ID != "p::func::B" ||
		report.VariantReemissions[0].CoreCopies != 2 {
		t.Fatalf("variant reemissions = %+v", report.VariantReemissions)
	}
}

// MUTATION (kind totality): a support record with an unknown kind is a
// typed defect — kind is never inferred from identity spelling.
func TestUnknownSupportKindIsDefect(t *testing.T) {
	run, generated := fixture()
	generated.Support[1].Kind = ""
	report := Build("head", run, generated)
	found := false
	for _, d := range report.Defects() {
		if d.ID == "p::func::B" && strings.Contains(d.Disposition, "unknown kind") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the unknown-kind defect, got %v", report.Defects())
	}
}

// Initializer support joins its own placeholder-emission class, never
// the body join.
func TestInitializerJoinIsSeparate(t *testing.T) {
	run, generated := fixture()
	generated.Support = append(generated.Support,
		translate.BodySupport{ID: "p::var::V", Package: "p", Kind: "initializer", State: "unimplemented"})
	generated.Emissions = append(generated.Emissions,
		emit.EmissionEvent{ID: "p::var::V", Kind: emit.EmissionInitializerPlaceholder})
	report := Build("head", run, generated)
	if defects := report.Defects(); len(defects) != 0 {
		t.Fatalf("clean initializer pair must not defect: %v", defects)
	}
	var bodies, inits int
	for _, d := range report.Denominators {
		switch d.Name {
		case "support-body-records":
			bodies = d.Count
		case "support-initializer-records":
			inits = d.Count
		}
	}
	if bodies != 2 || inits != 1 {
		t.Fatalf("bodies=%d inits=%d", bodies, inits)
	}
}

// MUTATION (missing emission): an unimplemented initializer with no
// placeholder emission surfaces on its join (non-defect disposition —
// the package may be unmaterialized — but never silent).
func TestMissingInitializerEmissionIsVisible(t *testing.T) {
	run, generated := fixture()
	generated.Support = append(generated.Support,
		translate.BodySupport{ID: "p::var::V", Package: "p", Kind: "initializer", State: "unimplemented"})
	report := Build("head", run, generated)
	for _, j := range report.Joins {
		if j.Name == "unimplemented-initializers-vs-placeholder-emissions" {
			if len(j.OnlyLeft) == 1 && j.OnlyLeft[0].ID == "p::var::V" {
				return
			}
			t.Fatalf("initializer join = %+v", j)
		}
	}
	t.Fatal("initializer join absent")
}

// NO-OUTPUT: a package whose typed disposition is no-runtime-output
// reconciles cleanly — the disposition is recorded by the emitter, not
// inferred from a missing file.
func TestNoRuntimeOutputDispositionIsClean(t *testing.T) {
	run, generated := fixture()
	run.Report.Declarations = append(run.Report.Declarations,
		census.DeclarationRecord{ID: "q::const::C", Package: "q", Kind: "const", Scope: "production"})
	generated.ModuleDispositions = append(generated.ModuleDispositions,
		translate.ModuleDisposition{Package: "q", State: "no-runtime-output", Reason: "all compile-time"})
	report := Build("head", run, generated)
	if defects := report.Defects(); len(defects) != 0 {
		t.Fatalf("no-runtime-output package must not defect: %v", defects)
	}
	for _, p := range report.Packages {
		if p.Path == "q" && !p.Materialized {
			t.Fatal("no-runtime-output package counts as materialized (nothing to materialize)")
		}
	}
}

// EMPTY-MODULE / MISSING DISPOSITION: a census package with neither a
// disposition nor a NotMaterialized reason is a typed defect.
func TestPackageWithoutDispositionIsDefect(t *testing.T) {
	run, generated := fixture()
	run.Report.Declarations = append(run.Report.Declarations,
		census.DeclarationRecord{ID: "q::func::F", Package: "q", Kind: "func", Scope: "production", HasBody: true})
	generated.Support = append(generated.Support,
		translate.BodySupport{ID: "q::func::F", Package: "q", Kind: "body", State: "ir-admitted"})
	generated.Emissions = append(generated.Emissions, emit.EmissionEvent{ID: "q::func::F", Kind: emit.EmissionBody})
	generated.Proofs = append(generated.Proofs,
		translate.Proof{ID: "q::func::F", Package: "q", GeneratedFile: "core/q/package.ts", LoweredHash: "h3"})
	report := Build("head", run, generated)
	found := false
	for _, d := range report.Defects() {
		if d.ID == "q" && strings.Contains(d.Disposition, "without a typed module disposition") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the missing-disposition defect for q, got %v", report.Defects())
	}
}

// A NOT-MATERIALIZED package carries its typed reason and does not
// defect.
func TestNotMaterializedCarriesDisposition(t *testing.T) {
	run, generated := fixture()
	generated.Emissions = nil
	generated.Proofs = nil
	generated.ModuleDispositions = nil
	delete(generated.Files, "core/p/package.ts")
	delete(generated.Ownership, "core/p/package.ts")
	generated.NotMaterialized["p"] = "2 declaration blockers"
	report := Build("head", run, generated)
	if defects := report.Defects(); len(defects) != 0 {
		t.Fatalf("not-materialized package must not defect: %v", defects)
	}
	for _, p := range report.Packages {
		if p.Path == "p" && (p.Materialized || p.Published) {
			t.Fatalf("not-materialized package state = %+v", p)
		}
	}
}

// Extern obligations join stub proofs through typed records only.
func TestExternObligationJoin(t *testing.T) {
	run, generated := fixture()
	generated.Files["extern/os/package.ts"] = "stub"
	generated.Ownership["extern/os/package.ts"] = "generated-external-contracts"
	generated.ExternSymbols = []emit.ExternSymbolRecord{
		{Symbol: "Getenv", Obligation: "os.Getenv"},
		{Symbol: "U$eq", Obligation: ""},
	}
	generated.Proofs = append(generated.Proofs,
		translate.Proof{ID: "os.Getenv", Package: "os", GeneratedFile: "extern/os/package.ts"})
	report := Build("head", run, generated)
	if defects := report.Defects(); len(defects) != 0 {
		t.Fatalf("clean extern pair must not defect: %v", defects)
	}
	var throwing, direct, obligations int
	for _, d := range report.Denominators {
		switch d.Name {
		case "extern-throwing-symbols":
			throwing = d.Count
		case "extern-direct-symbols":
			direct = d.Count
		case "extern-obligations":
			obligations = d.Count
		}
	}
	if throwing != 1 || direct != 1 || obligations != 1 {
		t.Fatalf("throwing=%d direct=%d obligations=%d", throwing, direct, obligations)
	}
}

// MUTATION: a thrown obligation with no stub proof record is a typed
// defect.
func TestThrownObligationWithoutProofIsDefect(t *testing.T) {
	run, generated := fixture()
	generated.ExternSymbols = []emit.ExternSymbolRecord{{Symbol: "Getenv", Obligation: "os.Getenv"}}
	report := Build("head", run, generated)
	found := false
	for _, d := range report.Defects() {
		if d.ID == "os.Getenv" && strings.Contains(d.Disposition, "without a stub proof record") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the obligation-without-proof defect, got %v", report.Defects())
	}
}

// The Markdown summary renders FROM the report.
func TestRenderCarriesDenominatorsAndJoins(t *testing.T) {
	run, generated := fixture()
	report := Build("head", run, generated)
	rendered := report.Render()
	for _, want := range []string{"census-production-bodies | 2", "Join bodies-vs-body-support", "both=2"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered summary missing %q:\n%s", want, rendered)
		}
	}
}

// A surplus placeholder copy of a RECORDED unimplemented body is an
// explained family variant, never a defect; a placeholder for an
// unknown identity stays one.
func TestVariantCopiesAreExplainedNotDefects(t *testing.T) {
	run, generated := fixture()
	generated.Support[1].State = "unimplemented"
	generated.Emissions = []emit.EmissionEvent{
		{ID: "p::func::A", Kind: emit.EmissionBody, Implementation: "default"},
		{ID: "p::func::B", Kind: emit.EmissionBodyPlaceholder, Implementation: "default"},
		{ID: "p::func::B", Kind: emit.EmissionBodyPlaceholder, Implementation: "map-key-encoded"},
	}
	generated.Proofs = generated.Proofs[:1]
	report := Build("head", run, generated)
	for _, d := range report.Defects() {
		if d.ID == "p::func::B" {
			t.Fatalf("explained variant copy must not defect: %v", d)
		}
	}
	if len(report.VariantReemissions) != 1 {
		t.Fatalf("variant reemissions = %+v", report.VariantReemissions)
	}
}
