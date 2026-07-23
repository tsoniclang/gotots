package stagecheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	mapping := firstLedgerIdentity(t, actual, "checked-mapping")
	synthetic, syntheticWithDifferentRole :=
		syntheticDefinitionRoleMutation(
			t,
			actual,
		)
	mutations := map[string]func(*structuralLedger){
		"missing-origin": func(ledger *structuralLedger) {
			delete(ledger.classes["checked-mapping"], mapping)
		},
		"duplicate-origin": func(ledger *structuralLedger) {
			ledger.classes["checked-mapping"][mapping]++
		},
		"extra-origin": func(ledger *structuralLedger) {
			ledger.add(
				"checked-mapping",
				mapping+"|extra",
			)
		},
		"relocated-origin": func(ledger *structuralLedger) {
			delete(ledger.classes["checked-mapping"], mapping)
			parts := strings.Split(mapping, "|")
			if len(parts) != 5 {
				t.Fatalf("checked mapping %q has %d fields", mapping, len(parts))
			}
			line, err := strconv.Atoi(parts[1])
			if err != nil {
				t.Fatal(err)
			}
			parts[1] = strconv.Itoa(line + 100)
			ledger.add(
				"checked-mapping",
				strings.Join(parts, "|"),
			)
		},
		"name-only-synthetic": func(ledger *structuralLedger) {
			delete(ledger.classes["definition"], synthetic)
			ledger.add("definition", syntheticWithDifferentRole)
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
	mapping := firstLedgerIdentity(t, actual, "checked-mapping")
	parts := strings.Split(mapping, "|")
	if len(parts) != 5 {
		t.Fatalf("checked mapping %q has %d fields", mapping, len(parts))
	}
	definition, err := identity.ParseDefinitionID(parts[0])
	if err != nil {
		t.Fatal(err)
	}
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
	line, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	column, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	key := independentOrigin{
		file:   definition.File().String(),
		line:   line,
		column: column,
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

func firstLedgerIdentity(
	t *testing.T,
	ledger *structuralLedger,
	class string,
) string {
	t.Helper()
	for identity := range ledger.classes[class] {
		return identity
	}
	t.Fatalf("ledger has no %s identity", class)
	return ""
}

func syntheticDefinitionRoleMutation(
	t *testing.T,
	ledger *structuralLedger,
) (string, string) {
	t.Helper()
	for record := range ledger.classes["definition"] {
		parts := strings.Split(record, "|")
		if len(parts) == 0 {
			continue
		}
		definition, err := identity.ParseDefinitionID(parts[0])
		if err != nil || !definition.SyntheticRole().Valid() {
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
			parts[0] = replacement.String()
			return record, strings.Join(parts, "|")
		}
	}
	t.Fatal("cgo ledger has no synthetic definition")
	return "", ""
}

func cloneStructuralLedger(
	source *structuralLedger,
) *structuralLedger {
	out := newStructuralLedger()
	for class, records := range source.classes {
		out.classes[class] = map[string]int{}
		for identity, count := range records {
			out.classes[class][identity] = count
		}
	}
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
