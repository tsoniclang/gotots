package stagecheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestCgoStructuralMutationsFailIndependentJoin(t *testing.T) {
	actual, expected := cgoStructuralLedgers(t)
	if err := compareLedgers(
		"cgo-clean",
		actual,
		expected,
	); err != nil {
		t.Fatalf("clean cgo structure failed: %v", err)
	}
	mapping := firstCheckedMapping(t, actual)
	synthetic, syntheticWithDifferentRole :=
		syntheticDefinitionRoleMutation(
			t,
			actual,
		)
	mutations := map[string]func(*structuralLedger){
		"missing-origin": func(ledger *structuralLedger) {
			delete(ledger.checkedMappings, mapping)
		},
		"duplicate-origin": func(ledger *structuralLedger) {
			ledger.checkedMappings[mapping]++
		},
		"extra-origin": func(ledger *structuralLedger) {
			mutated := mapping
			mutated.checkedDigest += "extra"
			addRecord(&ledger.checkedMappings, mutated)
		},
		"relocated-origin": func(ledger *structuralLedger) {
			delete(ledger.checkedMappings, mapping)
			mutated := mapping
			mutated.originLine += 100
			addRecord(&ledger.checkedMappings, mutated)
		},
		"name-only-synthetic": func(ledger *structuralLedger) {
			delete(ledger.definitions, synthetic)
			addRecord(&ledger.definitions, syntheticWithDifferentRole)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			mutated := cloneStructuralLedger(actual)
			mutate(mutated)
			if err := compareLedgers(
				"cgo-"+name,
				mutated,
				expected,
			); err == nil {
				t.Fatal("mutated cgo structure exact-joined")
			}
		})
	}
}

func TestIndependentCgoOriginRejectsAmbiguity(t *testing.T) {
	actual, _ := cgoStructuralLedgers(t)
	mapping := firstCheckedMapping(t, actual)
	definition := mapping.definition
	secondRoot, err := identity.NewOccurrenceID(
		mustShiftSpan(t, definition.Root().Span()),
		uint16(catalog.KindFuncDecl),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := identity.NewSourceDefinitionID(
		secondRoot,
		definition.Kind(),
	)
	if err != nil {
		t.Fatal(err)
	}
	key := independentOrigin{
		file:   definition.File().String(),
		line:   mapping.originLine,
		column: mapping.originColumn,
		kind:   definition.Kind(),
	}
	if _, _, err := independentResolveOrigin(
		map[independentOrigin][]identity.DefinitionID{
			key: {definition, second},
		},
		key,
	); err == nil {
		t.Fatal("ambiguous cgo origin resolved")
	}
}

func cgoStructuralLedgers(
	t *testing.T,
) (*structuralLedger, *structuralLedger) {
	t.Helper()
	if output, err := exec.Command(
		"go",
		"env",
		"CGO_ENABLED",
	).Output(); err != nil ||
		strings.TrimSpace(string(output)) != "1" {
		t.Skip("cgo is unavailable")
	}
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("C compiler is unavailable")
	}
	directory := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/cgo-mutation\n\ngo 1.26.0\n",
		"main.go": `package main

/*
static int add(int left, int right) { return left + right; }
*/
import "C"

func external() int { return int(C.add(1, 2)) }
func main() { _ = external() }
`,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(content),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
	}
	request := source.Request{
		Dir:      directory,
		Patterns: []string{"."},
		Env:      []string{"CGO_ENABLED=1"},
	}
	base, err := source.ResolveUniverse(request)
	if err != nil {
		t.Skipf("cgo resolution unavailable: %v", err)
	}
	var packageID identity.PackageID
	for _, pkg := range base.Packages() {
		if pkg.RequestedRoot() {
			packageID = pkg.ID()
			break
		}
	}
	if packageID.IsZero() {
		t.Fatal("cgo fixture has no root package")
	}
	universe, err := source.ForkForHydration(
		base,
		[]identity.PackageID{packageID},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = source.DiscardHydratedUniverse(universe)
	})
	var files []identity.FileID
	var synthetic []identity.PackageID
	for _, pkg := range universe.Packages() {
		if pkg.ID() != packageID {
			continue
		}
		for _, file := range pkg.Files() {
			files = append(files, file.ID())
		}
		if pkg.HasCheckedView() {
			synthetic = append(synthetic, packageID)
		}
	}
	hydration, err := source.NewHydrationRequest(files, synthetic)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.HydrateUniverse(universe, hydration); err != nil {
		t.Skipf("cgo hydration unavailable: %v", err)
	}
	graph, _, err := structure.BuildPackages(
		universe,
		[]identity.PackageID{packageID},
	)
	if err != nil {
		t.Fatal(err)
	}
	var pkg structure.PackageGraph
	if err := graph.VisitPackages(func(
		record structure.PackageGraph,
	) error {
		pkg = record
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	actual := ledgerForPackage(pkg)
	expected, err := deriveExpectedGraph(
		universe,
		nil,
		nil,
		map[identity.PackageID]bool{packageID: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	return actual, expected
}

func firstCheckedMapping(
	t *testing.T,
	ledger *structuralLedger,
) checkedMappingLedgerRecord {
	t.Helper()
	for mapping := range ledger.checkedMappings {
		return mapping
	}
	t.Fatal("ledger has no checked mapping")
	return checkedMappingLedgerRecord{}
}

func syntheticDefinitionRoleMutation(
	t *testing.T,
	ledger *structuralLedger,
) (definitionLedgerRecord, definitionLedgerRecord) {
	t.Helper()
	for record := range ledger.definitions {
		definition := record.id
		if !definition.SyntheticRole().Valid() {
			continue
		}
		for role := identity.SyntheticDefinitionRole(1); role.Valid(); role++ {
			if role == definition.SyntheticRole() {
				continue
			}
			replacement, err := identity.NewSyntheticDefinitionID(
				definition.Package(),
				role,
				definition.SyntheticName(),
			)
			if err != nil {
				t.Fatal(err)
			}
			mutated := record
			mutated.id = replacement
			return record, mutated
		}
	}
	t.Fatal("cgo ledger has no synthetic definition")
	return definitionLedgerRecord{}, definitionLedgerRecord{}
}

func cloneStructuralLedger(
	source *structuralLedger,
) *structuralLedger {
	out := newStructuralLedger()
	out.merge(source)
	return out
}

func mustShiftSpan(
	t *testing.T,
	span identity.SpanID,
) identity.SpanID {
	t.Helper()
	shifted, err := identity.NewSpanID(
		span.File(),
		span.Start()+1,
		span.End()+1,
	)
	if err != nil {
		t.Fatal(err)
	}
	return shifted
}
