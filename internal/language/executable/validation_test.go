package executable

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestImplicitFullDefinitionRequiresExactOperationGraph(t *testing.T) {
	graph, selections, inventory, implicit := buildExecutableFixture(t)
	region := inventory.byID[implicit]
	region.implicit = nil
	inventory.byID[implicit] = region
	err := Validate(graph, selections, inventory)
	if err == nil ||
		!strings.Contains(err.Error(), "implicit definition") {
		t.Fatalf("invalid implicit operation graph error = %v", err)
	}
}

func TestCanonicalOccurrencePayloadCannotCrossStores(t *testing.T) {
	graph, selections, inventory, _ := buildExecutableFixture(t)
	var duplicated structure.Occurrence
	for _, region := range inventory.Regions() {
		for _, member := range region.Members() {
			if occurrence, present := graph.ResidentOccurrence(member); present {
				duplicated = occurrence
				break
			}
		}
		if !duplicated.ID().IsZero() {
			break
		}
	}
	if duplicated.ID().IsZero() {
		t.Fatal("fixture has no structurally owned executable member")
	}
	reference, added, err := inventory.occurrences.put(duplicated)
	if err != nil || !added {
		t.Fatalf("install duplicate occurrence: added=%t err=%v", added, err)
	}
	inventory.additional = append(inventory.additional, reference)
	inventory.sort()
	err = Validate(graph, selections, inventory)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"duplicated across structural and executable stores",
		) {
		t.Fatalf("cross-store payload duplication error = %v", err)
	}
}

func TestOccurrenceArenaRejectsConflictingPayload(t *testing.T) {
	_, _, inventory, _ := buildExecutableFixture(t)
	additional := inventory.AdditionalOccurrences()
	if len(additional) == 0 {
		t.Fatal("fixture has no executable-only occurrences")
	}
	original := additional[0]
	display := original.Display()
	display.Start.Filename += ".conflict"
	conflicting, err := structure.NewOccurrence(
		original.ID(),
		original.Kind(),
		original.Parent(),
		original.Edge(),
		original.Ordinal(),
		original.Span(),
		display,
		original.Token(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := inventory.occurrences.put(conflicting); err == nil ||
		!strings.Contains(err.Error(), "conflicting payloads") {
		t.Fatalf("conflicting occurrence payload error = %v", err)
	}
}

func TestOccurrenceArenaOwnsAllRegionReferences(t *testing.T) {
	_, _, inventory, _ := buildExecutableFixture(t)
	if inventory.occurrences.length() <
		inventory.occurrences.payloadLength() {
		t.Fatal("occurrence arena has more payloads than identities")
	}
	for _, region := range inventory.Regions() {
		if region.occurrences != inventory.occurrences {
			t.Fatal("region does not use the inventory occurrence arena")
		}
		for _, reference := range region.members {
			id, err := inventory.occurrences.id(reference)
			if err != nil {
				t.Fatal(err)
			}
			if inventory.occurrences.reference(id) != reference {
				t.Fatalf("occurrence reference %d is not canonical", reference)
			}
		}
	}
	for _, reference := range inventory.additional {
		if inventory.occurrences.payloadFor(reference) == nil {
			t.Fatalf("additional occurrence %d has no payload", reference)
		}
	}
}

func TestAdditionalOccurrenceLookupIsFileScoped(t *testing.T) {
	_, _, inventory, _ := buildExecutableFixture(t)
	additional := inventory.AdditionalOccurrences()
	if len(additional) == 0 {
		t.Fatal("fixture has no executable-only occurrences")
	}
	selectedFile := additional[0].ID().Span().File()
	references, err := inventory.AdditionalOccurrenceRefsForFiles(
		[]identity.FileID{selectedFile},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, occurrence := range additional {
		if occurrence.ID().Span().File() == selectedFile {
			want++
		}
	}
	if len(references) != want {
		t.Fatalf(
			"file-scoped additional occurrences = %d, want %d",
			len(references),
			want,
		)
	}
	for _, reference := range references {
		if reference.ID().Span().File() != selectedFile {
			t.Fatalf(
				"file-scoped lookup leaked %s",
				reference.ID(),
			)
		}
	}
	if _, err := inventory.AdditionalOccurrenceRefsForFiles(
		[]identity.FileID{selectedFile, selectedFile},
	); err == nil {
		t.Fatal("duplicate file lookup did not fail")
	}
}

func buildExecutableFixture(
	t *testing.T,
) (
	*structure.Graph,
	*scope.DefinitionSelections,
	*Inventory,
	identity.DefinitionID,
) {
	t.Helper()
	directory := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":  "module example.com/executable\n\ngo 1.26.0\n",
		"main.go": "package executablefixture\n\nfunc Main() int { return 1 }\n",
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(content),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
	}
	universe, err := source.ResolveUniverse(source.Request{
		Dir: directory, Patterns: []string{"."},
	})
	if err != nil {
		t.Fatal(err)
	}
	var files []identity.FileID
	for _, pkg := range universe.Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, file := range pkg.Files() {
			files = append(files, file.ID())
		}
	}
	hydration, err := source.NewHydrationRequest(files, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.HydrateUniverse(universe, hydration); err != nil {
		t.Fatal(err)
	}
	graph, index, err := structure.Build(universe)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := contract.Default()
	if err != nil {
		t.Fatal(err)
	}
	facts, err := selectionfacts.MaterializeForAudit(
		universe,
		graph,
		index,
		selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	selections, err := scope.SelectDefinitions(
		universe,
		graph,
		facts,
		selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := Build(graph, index, selections)
	if err != nil {
		t.Fatal(err)
	}
	var implicit identity.DefinitionID
	for _, definition := range graph.ResidentDefinitions() {
		if definition.ID().ImplicitOp() ==
			identity.ImplicitDefinitionPackageInit {
			implicit = definition.ID()
			break
		}
	}
	if implicit.IsZero() {
		t.Fatal("fixture has no package-initialization definition")
	}
	return graph, selections, inventory, implicit
}
