package stagecheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func TestFinalizedSurfaceScannerRejectsRawToolchainCarriers(t *testing.T) {
	controls := []any{
		struct{ File *ast.File }{},
		struct{ Expression ast.Expr }{},
		struct{ FileSet *token.FileSet }{},
		struct{ File *token.File }{},
		struct{ Info *types.Info }{},
		struct{ Package *types.Package }{},
		struct{ Scope *types.Scope }{},
		struct{ Selection *types.Selection }{},
		struct{ Instance types.Instance }{},
		struct{ Type types.Type }{},
		struct{ Object types.Object }{},
		struct{ Concrete *types.Named }{},
	}
	for _, control := range controls {
		typ := reflect.TypeOf(control)
		if path := rawFacadePath(typ, map[reflect.Type]bool{}); path == "" {
			t.Errorf("raw toolchain control %s escaped the surface scan", typ)
		}
	}
	safe := struct {
		Definition identity.DefinitionID
		Names      []string
	}{}
	if path := rawFacadePath(
		reflect.TypeOf(safe), map[reflect.Type]bool{},
	); path != "" {
		t.Fatalf("identity-only control was rejected at %s", path)
	}
}

func TestFinalizedStage2ModelRejectsRawToolchainCarriers(t *testing.T) {
	modelType := reflect.TypeOf((*semantic.Model)(nil))
	if path := rawFacadePath(
		modelType, map[reflect.Type]bool{},
	); path != "" {
		t.Fatalf("semantic model retains raw transient state at %s", path)
	}
	rawControl := struct {
		Checker *types.Package
	}{}
	if path := rawFacadePath(
		reflect.TypeOf(rawControl), map[reflect.Type]bool{},
	); path == "" {
		t.Fatal("raw checker negative control escaped Stage-2 lifecycle gate")
	}
}

func TestStructuralLedgerMutationsFailWithExactResidualEvidence(t *testing.T) {
	canonical := func() *structuralLedger {
		ledger := newStructuralLedger()
		ledger.add("definition", "D1|owner|header|boundary|Outer")
		ledger.add("definition-reference", "D1|parent|edge=7|ordinal=0|D2")
		ledger.add("definition", "D2|owner|header|boundary|literal")
		return ledger
	}
	mutations := map[string]func(*structuralLedger){
		"omit": func(ledger *structuralLedger) {
			delete(ledger.classes["definition"], "D2|owner|header|boundary|literal")
		},
		"duplicate": func(ledger *structuralLedger) {
			ledger.add("definition", "D1|owner|header|boundary|Outer")
		},
		"reparent": func(ledger *structuralLedger) {
			delete(
				ledger.classes["definition-reference"],
				"D1|parent|edge=7|ordinal=0|D2",
			)
			ledger.add(
				"definition-reference",
				"D1|wrong-parent|edge=7|ordinal=0|D2",
			)
		},
		"edge": func(ledger *structuralLedger) {
			delete(
				ledger.classes["definition-reference"],
				"D1|parent|edge=7|ordinal=0|D2",
			)
			ledger.add(
				"definition-reference",
				"D1|parent|edge=8|ordinal=0|D2",
			)
		},
		"ordinal": func(ledger *structuralLedger) {
			delete(
				ledger.classes["definition-reference"],
				"D1|parent|edge=7|ordinal=0|D2",
			)
			ledger.add(
				"definition-reference",
				"D1|parent|edge=7|ordinal=1|D2",
			)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			actual := canonical()
			expected := canonical()
			mutate(actual)
			err := compareLedgers("mutation", actual, expected)
			if err == nil {
				t.Fatal("mutated structural ledger exact-joined")
			}
			verification, ok := err.(*VerificationError)
			if !ok || verification.Stage != "mutation" ||
				!strings.Contains(
					verification.Reason,
					"exact structural join failed",
				) ||
				!strings.Contains(verification.Reason, "digest=") ||
				!strings.Contains(verification.Reason, "sample=") {
				t.Fatalf("mutation error lacks exact residual evidence: %v", err)
			}
		})
	}
	if err := compareLedgers("clean", canonical(), canonical()); err != nil {
		t.Fatalf("identical ledgers failed: %v", err)
	}
}
