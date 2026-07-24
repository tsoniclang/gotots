package stagecheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
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
	module, err := identity.NewModuleID("example.com/ledger", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	file, err := identity.NewFileID(owner, "ledger.go")
	if err != nil {
		t.Fatal(err)
	}
	definitionAt := func(start, end int) identity.DefinitionID {
		span, spanErr := identity.NewSpanID(file, start, end)
		if spanErr != nil {
			t.Fatal(spanErr)
		}
		occurrence, occurrenceErr := identity.NewOccurrenceID(span, 47)
		if occurrenceErr != nil {
			t.Fatal(occurrenceErr)
		}
		definition, definitionErr := identity.NewSourceDefinitionID(
			occurrence, identity.DefinitionFuncDecl,
		)
		if definitionErr != nil {
			t.Fatal(definitionErr)
		}
		return definition
	}
	outer := definitionAt(10, 20)
	literal := definitionAt(12, 18)
	outerHeader, _ := identity.NewHeaderRegionID(outer)
	outerBoundary, _ := identity.NewExecutionBoundaryID(outer)
	literalHeader, _ := identity.NewHeaderRegionID(literal)
	literalBoundary, _ := identity.NewExecutionBoundaryID(literal)
	ownerRegion, _ := structure.SourceFileOwner(file)
	outerRecord := definitionLedgerRecord{
		id: outer, owner: ownerRegion,
		header: outerHeader, boundary: outerBoundary, name: "Outer",
	}
	literalRecord := definitionLedgerRecord{
		id: literal, owner: ownerRegion,
		header: literalHeader, boundary: literalBoundary, name: "literal",
	}
	canonical := func() *structuralLedger {
		ledger := newStructuralLedger()
		addRecord(&ledger.definitions, outerRecord)
		addRecord(&ledger.definitions, literalRecord)
		return ledger
	}
	mutations := map[string]func(*structuralLedger){
		"omit": func(ledger *structuralLedger) {
			delete(ledger.definitions, literalRecord)
		},
		"duplicate": func(ledger *structuralLedger) {
			addRecord(&ledger.definitions, outerRecord)
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

	arena := newExecutableLedgerArena()
	reference := compactDefinitionReference{
		region: arena.definition(outer),
		parent: outer.Root(),
		edge:   uint16(catalog.Edge(7)),
		child:  arena.definition(literal),
	}
	canonicalExecutable := func() *compactExecutableLedger {
		ledger := newCompactExecutableLedger(arena)
		addRecord(&ledger.regions, reference.region)
		addRecord(
			&ledger.definitionReferences,
			reference,
		)
		return ledger
	}
	executableMutations := map[string]func(*compactExecutableLedger){
		"omit-reference": func(ledger *compactExecutableLedger) {
			delete(ledger.definitionReferences, reference)
		},
		"duplicate-reference": func(ledger *compactExecutableLedger) {
			addRecord(
				&ledger.definitionReferences,
				reference,
			)
		},
		"reparent": func(ledger *compactExecutableLedger) {
			delete(ledger.definitionReferences, reference)
			mutated := reference
			mutated.parent = literal.Root()
			addRecord(
				&ledger.definitionReferences,
				mutated,
			)
		},
		"edge": func(ledger *compactExecutableLedger) {
			delete(ledger.definitionReferences, reference)
			mutated := reference
			mutated.edge = uint16(catalog.Edge(8))
			addRecord(
				&ledger.definitionReferences,
				mutated,
			)
		},
		"ordinal": func(ledger *compactExecutableLedger) {
			delete(ledger.definitionReferences, reference)
			mutated := reference
			mutated.ordinal = 1
			addRecord(
				&ledger.definitionReferences,
				mutated,
			)
		},
	}
	for name, mutate := range executableMutations {
		t.Run("executable-"+name, func(t *testing.T) {
			actual := canonicalExecutable()
			expected := canonicalExecutable()
			mutate(actual)
			err := compareCompactExecutableLedgers(
				"mutation", actual, expected,
			)
			if err == nil {
				t.Fatal("mutated executable ledger exact-joined")
			}
			verification, ok := err.(*VerificationError)
			if !ok ||
				verification.Stage != "mutation" ||
				!strings.Contains(
					verification.Reason,
					"exact executable join failed",
				) ||
				!strings.Contains(verification.Reason, "digest=") ||
				!strings.Contains(verification.Reason, "sample=") {
				t.Fatalf(
					"mutation error lacks exact residual evidence: %v",
					err,
				)
			}
		})
	}
	if err := compareCompactExecutableLedgers(
		"clean", canonicalExecutable(), canonicalExecutable(),
	); err != nil {
		t.Fatalf("identical executable ledgers failed: %v", err)
	}
}
