// Focused negative tests for the acceptance gate's evidence-integrity
// logic: each test forges one defect class and proves the gate detects
// it — mutation coverage for the gate itself, not the happy path.
package main

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/translate"
)

// censusWith builds a minimal census result holding the given production
// declarations and function shapes.
func censusWith(declarations []census.DeclarationRecord, functions []census.FunctionShape) *census.Result {
	return &census.Result{
		Report: &census.Report{Declarations: declarations},
		Shapes: &census.DeclarationShapes{Functions: functions},
	}
}

func productionDecl(id, pkg, kind string) census.DeclarationRecord {
	return census.DeclarationRecord{ID: id, Package: pkg, Kind: kind, Scope: "production"}
}

func TestStage05MissingInputBlocks(t *testing.T) {
	// The guard must run before any dereference (the audit's guard-order
	// defect).
	status, _, err := runSignatureCompletenessGate(nil, nil)
	if err != nil || status != "blocked" {
		t.Fatalf("nil input: status=%q err=%v; want blocked", status, err)
	}
	status, _, _ = runSignatureCompletenessGate(&census.Result{}, &translate.Generated{})
	if status != "blocked" {
		t.Fatalf("missing shapes: status=%q; want blocked", status)
	}
}

func TestStage05MissingProofFails(t *testing.T) {
	first := censusWith(
		[]census.DeclarationRecord{productionDecl("p::func::F", "p", "func")},
		[]census.FunctionShape{{ID: "p::func::F", Signature: "func F()"}},
	)
	generated := &translate.Generated{}
	status, details, _ := runSignatureCompletenessGate(first, generated)
	if status != "fail" {
		t.Fatalf("missing proof and empty proof set must fail; got %q %v", status, details)
	}
}

func TestStage05OrphanProofFails(t *testing.T) {
	first := censusWith(
		[]census.DeclarationRecord{productionDecl("p::func::F", "p", "func")},
		[]census.FunctionShape{{ID: "p::func::F", Signature: "func F()"}},
	)
	generated := &translate.Generated{Proofs: []translate.Proof{
		{ID: "p::func::F", SignatureHash: signatureHashOf("func F()")},
		{ID: "p::func::Ghost", SignatureHash: signatureHashOf("func Ghost()")},
	}}
	status, details, _ := runSignatureCompletenessGate(first, generated)
	if status != "fail" || !strings.Contains(strings.Join(details, "\n"), "orphan proof") {
		t.Fatalf("orphan proof must fail; got %q %v", status, details)
	}
}

func TestStage05SignatureDriftFails(t *testing.T) {
	first := censusWith(
		[]census.DeclarationRecord{productionDecl("p::func::F", "p", "func")},
		[]census.FunctionShape{{ID: "p::func::F", Signature: "func F()"}},
	)
	generated := &translate.Generated{Proofs: []translate.Proof{
		{ID: "p::func::F", SignatureHash: signatureHashOf("func F(changed int)")},
	}}
	status, details, _ := runSignatureCompletenessGate(first, generated)
	if status != "fail" || !strings.Contains(strings.Join(details, "\n"), "signature drift") {
		t.Fatalf("signature drift must fail; got %q %v", status, details)
	}
}

func TestStage05UnimplementedRecordAccounts(t *testing.T) {
	first := censusWith(
		[]census.DeclarationRecord{productionDecl("p::func::F", "p", "func")},
		[]census.FunctionShape{{ID: "p::func::F", Signature: "func F()"}},
	)
	generated := &translate.Generated{
		Proofs:  []translate.Proof{{ID: "p::func::Other", SignatureHash: signatureHashOf("x")}},
		Support: []translate.BodySupport{{ID: "p::func::F", State: "unimplemented"}},
	}
	// The one production shape is unimplemented; the sole proof is an
	// orphan and must still fail.
	status, details, _ := runSignatureCompletenessGate(first, generated)
	if status != "fail" {
		t.Fatalf("orphan alongside unimplemented must still fail; got %q %v", status, details)
	}
	// With the orphan removed, the explicit unimplemented record fully
	// accounts for the denominator: the join succeeds (no defect), but the
	// stage is BLOCKED, not pass — it verifies only signatures/funclits/
	// initializers and only by shared-typeid agreement, not independent
	// structural TS-vs-Go comparison across every declaration class.
	generated.Proofs = nil
	status, details, _ = runSignatureCompletenessGate(first, generated)
	if status != "blocked" {
		t.Fatalf("fully unimplemented denominator with a successful join must block (not overclaim completeness); got %q %v", status, details)
	}
	// But a denominator with an unaccounted shape AND zero proofs is a
	// disjoint evidence set and must fail.
	first2 := censusWith(
		[]census.DeclarationRecord{productionDecl("p::func::F", "p", "func"), productionDecl("p::func::G", "p", "func")},
		[]census.FunctionShape{{ID: "p::func::F", Signature: "func F()"}, {ID: "p::func::G", Signature: "func G()"}},
	)
	status, _, _ = runSignatureCompletenessGate(first2, generated)
	if status != "fail" {
		t.Fatalf("unaccounted shape with empty proofs must fail; got %q", status)
	}
}

// censusWithConst builds a census result whose sole production declaration
// is one constant shape.
func censusWithConst(shape census.ConstShape) *census.Result {
	return &census.Result{
		Report: &census.Report{Declarations: []census.DeclarationRecord{
			productionDecl(shape.ID, "p", "const")}},
		Shapes: &census.DeclarationShapes{Constants: []census.ConstShape{shape}},
	}
}

func TestStage05ConstMissingEvidenceFails(t *testing.T) {
	first := censusWithConst(census.ConstShape{ID: "p::const::C", Type: "int", Value: "42"})
	// A const with no ConstHash proof is unaccounted evidence.
	status, details, _ := runSignatureCompletenessGate(first, &translate.Generated{})
	if status != "fail" || !strings.Contains(strings.Join(details, "\n"), "no constant evidence") {
		t.Fatalf("missing const evidence must fail; got %q %v", status, details)
	}
}

func TestStage05ConstShapeDriftFails(t *testing.T) {
	first := censusWithConst(census.ConstShape{ID: "p::const::C", Type: "int", Value: "42"})
	// The translator hashed a DIFFERENT value (7): a value drift the join
	// must catch even though the identity matches.
	generated := &translate.Generated{Proofs: []translate.Proof{{
		ID:        "p::const::C",
		ConstHash: signatureHashOf(census.ConstShapeSignature(census.ConstShape{Type: "int", Value: "7"})),
	}}}
	status, details, _ := runSignatureCompletenessGate(first, generated)
	if status != "fail" || !strings.Contains(strings.Join(details, "\n"), "constant shape drift") {
		t.Fatalf("const shape drift must fail; got %q %v", status, details)
	}
}

func TestStage05ConstOrphanProofFails(t *testing.T) {
	first := censusWithConst(census.ConstShape{ID: "p::const::C", Type: "int", Value: "42"})
	generated := &translate.Generated{Proofs: []translate.Proof{
		{ID: "p::const::C", ConstHash: signatureHashOf(census.ConstShapeSignature(census.ConstShape{Type: "int", Value: "42"}))},
		{ID: "p::const::Ghost", ConstHash: signatureHashOf(census.ConstShapeSignature(census.ConstShape{Type: "int", Value: "0"}))},
	}}
	status, details, _ := runSignatureCompletenessGate(first, generated)
	if status != "fail" || !strings.Contains(strings.Join(details, "\n"), "orphan constant proof") {
		t.Fatalf("orphan const proof must fail; got %q %v", status, details)
	}
}

func TestStage05ConstJoinsThenBlocks(t *testing.T) {
	first := censusWithConst(census.ConstShape{ID: "p::const::C", Type: "int", Value: "42"})
	generated := &translate.Generated{Proofs: []translate.Proof{{
		ID:        "p::const::C",
		ConstHash: signatureHashOf(census.ConstShapeSignature(census.ConstShape{Type: "int", Value: "42"})),
	}}}
	// The constant joins cleanly, but the stage still BLOCKS: the remaining
	// declaration classes and independent TS comparison are not yet done.
	status, details, _ := runSignatureCompletenessGate(first, generated)
	if status != "blocked" {
		t.Fatalf("a clean const join must still block (join incomplete); got %q %v", status, details)
	}
	if !strings.Contains(strings.Join(details, "\n"), "constants joined by canonical type and exact value: 1") {
		t.Fatalf("blocked details must report the const join count; got %v", details)
	}
}

func TestReconcileDispositionsForgedEvidence(t *testing.T) {
	first := censusWith([]census.DeclarationRecord{
		productionDecl("p::func::F", "p", "func"),
	}, nil)
	base := func() *translate.Generated {
		return &translate.Generated{
			Files:  map[string]string{"core/p/package.ts": "export function F() {}"},
			Proofs: []translate.Proof{{ID: "p::func::F", Package: "p", GeneratedFile: "core/p/package.ts", ModuleRetained: true}},
		}
	}
	cases := []struct {
		name   string
		mutate func(*translate.Generated)
		expect string
	}{
		{"duplicate proof", func(g *translate.Generated) {
			g.Proofs = append(g.Proofs, g.Proofs[0])
		}, "duplicate proof record"},
		{"duplicate support", func(g *translate.Generated) {
			g.Support = []translate.BodySupport{{ID: "x", State: ir.SupportIRAdmitted}, {ID: "x", State: ir.SupportIRAdmitted}}
		}, "duplicate support record"},
		{"conflicting states (proof vs unimplemented)", func(g *translate.Generated) {
			g.Support = []translate.BodySupport{{ID: "p::func::F", State: ir.SupportUnimplemented}}
		}, "proof=ir-admitted vs support"},
		{"phantom generated file", func(g *translate.Generated) {
			g.Proofs[0].GeneratedFile = "core/p/absent.ts"
		}, "phantom generated file"},
		{"retained inside non-materialized package", func(g *translate.Generated) {
			// A merely publication-withheld package legitimately retains its
			// bodies; only a package with NO analyzable file must not.
			g.NotMaterialized = map[string]string{"p": "reason"}
			g.Withheld = map[string]string{"p": "reason"}
			delete(g.Files, "core/p/package.ts")
		}, "module-retained inside non-materialized package"},
		{"unretained despite materialized package", func(g *translate.Generated) {
			g.Proofs[0].ModuleRetained = false
		}, "unretained despite a materialized package"},
		{"no-output with generated file", func(g *translate.Generated) {
			g.Proofs[0].ModuleRetained = false
			g.Proofs[0].NoOutput = true
		}, "no-output disposition with a generated file"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := base()
			c.mutate(g)
			_, _, conflicts, _ := reconcileDispositions(nil, first, g)
			if !strings.Contains(strings.Join(conflicts, "\n"), c.expect) {
				t.Fatalf("forged %s not detected; conflicts: %v", c.name, conflicts)
			}
		})
	}
}

func TestBinaryProvenanceFailsClosed(t *testing.T) {
	if err := checkBinaryProvenance("", "abc"); err == nil {
		t.Fatal("missing VCS stamp must fail closed")
	}
	if err := checkBinaryProvenance("stale", "abc"); err == nil {
		t.Fatal("stale binary revision must fail closed")
	}
	if err := checkBinaryProvenance("abc", "abc"); err != nil {
		t.Fatalf("matching provenance must pass: %v", err)
	}
}

// TestRepresentationRegistryRejectsUnknown proves an unknown candidate
// in ANY family fails classification — not only slices.
func TestRepresentationRegistryRejectsUnknown(t *testing.T) {
	cases := []struct {
		key, candidate string
		ok             bool
	}{
		{"slice-local:x", "native-array", true},
		{"slice-local:x", "quantum-array", false},
		{"carrier:int", "bigint-exact-64", true},
		{"int", "bigint-exact-64", false},
		{"carrier:int", "forged-carrier", false},
		{"carrier:T", "unreviewed-kind(chan int)", false},
		{"decl:Name", "const-folded-at-use", true},
		{"Name", "const-folded-at-use", false},
		{"decl:Name", "erased-to-carrier(bigint-exact-64)", true},
		{"decl:Name", "erased-to-carrier(const-folded-at-use)", false},
		{"decl:Name", "erased-to-carrier(bigint)", false},
		{"decl:Name", "module-let(live-binding,boolean)", true},
		{"decl:Name", "module-let(garbage", false},
		{"carrier:int8", "number-wrapped-8", true},
		{"carrier:int8", "number-wrapped-anything", false},
		{"other", "native-array", false},
		{"decl:Name", "totally-made-up", false},
		{"garbage", "boolean", false},
	}
	for _, c := range cases {
		_, ok := translate.ClassifyRepresentation(c.key, c.candidate)
		if ok != c.ok {
			t.Errorf("ClassifyRepresentation(%q, %q) = %v; want %v", c.key, c.candidate, ok, c.ok)
		}
	}
}
