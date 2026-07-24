package structure

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestDefinitionSiteStoresOnlyCanonicalContainmentAnchors(t *testing.T) {
	site := reflect.TypeOf(DefinitionSite{})
	want := []struct {
		name string
		typ  reflect.Type
	}{
		{"kind", reflect.TypeOf(DefinitionSiteKind(0))},
		{"definition", reflect.TypeOf(identity.DefinitionID{})},
		{"owner", reflect.TypeOf(OwnerRegionID{})},
		{"parentDefinition", reflect.TypeOf(identity.DefinitionID{})},
		{"terminal", reflect.TypeOf(identity.OccurrenceID{})},
	}
	if site.NumField() != len(want) {
		t.Fatalf(
			"definition site stores %d fields, want canonical %d",
			site.NumField(),
			len(want),
		)
	}
	for index, field := range want {
		actual := site.Field(index)
		if actual.Name != field.name || actual.Type != field.typ {
			t.Fatalf(
				"definition site field %d = %s %s, want %s %s",
				index,
				actual.Name,
				actual.Type,
				field.name,
				field.typ,
			)
		}
	}
}

func TestStructuralGraphAdmissionRejectsCorruptedRelations(t *testing.T) {
	mutations := map[string]func(*Graph){
		"unsealed-occurrence-index": func(graph *Graph) {
			graph.byOccurrence = nil
		},
		"duplicate-package": func(graph *Graph) {
			graph.packages = append(graph.packages, graph.packages[0])
		},
		"missing-definition-census-record": func(graph *Graph) {
			graph.definitions = graph.definitions[1:]
		},
		"invalid-definition-census-identity": func(graph *Graph) {
			graph.definitions[0].id = identity.DefinitionID{}
		},
		"duplicate-definition": func(graph *Graph) {
			file := &graph.packages[0].files[0]
			file.definitions = append(file.definitions, file.definitions[0])
		},
		"definition-without-owner": func(graph *Graph) {
			graph.packages[0].files[0].definitions[0].owner =
				OwnerRegionID{}
		},
		"missing-site": func(graph *Graph) {
			file := &graph.packages[0].files[0]
			file.sites = file.sites[1:]
		},
		"orphan-site": func(graph *Graph) {
			graph.packages[0].files[0].sites[0].definition =
				identity.DefinitionID{}
		},
		"reparent-site": func(graph *Graph) {
			for index := range graph.packages[0].files[0].sites {
				site := &graph.packages[0].files[0].sites[index]
				if !site.parentDefinition.IsZero() {
					site.parentDefinition = identity.DefinitionID{}
					return
				}
			}
		},
		"cycle-site": func(graph *Graph) {
			for index := range graph.packages[0].files[0].sites {
				site := &graph.packages[0].files[0].sites[index]
				if !site.parentDefinition.IsZero() {
					site.parentDefinition = site.definition
					return
				}
			}
		},
		"missing-containment-anchor": func(graph *Graph) {
			file := &graph.packages[0].files[0]
			file.containment.anchors = file.containment.anchors[1:]
		},
		"missing-header": func(graph *Graph) {
			file := &graph.packages[0].files[0]
			file.headers = file.headers[1:]
		},
		"header-root-replaced": func(graph *Graph) {
			file := &graph.packages[0].files[0]
			file.headers[0].members[0] = file.owner.members[0]
		},
		"missing-boundary": func(graph *Graph) {
			file := &graph.packages[0].files[0]
			file.boundaries = file.boundaries[1:]
		},
		"boundary-digest-corrupted": func(graph *Graph) {
			graph.packages[0].files[0].boundaries[0].combinedDigest =
				strings.Repeat("0", 64)
		},
		"site-anchored-at-body-entry": func(graph *Graph) {
			file := &graph.packages[0].files[0]
			for index := range file.sites {
				site := &file.sites[index]
				for _, boundary := range file.boundaries {
					if boundary.id.Definition() == site.definition &&
						len(boundary.entries) != 0 {
						site.terminal = boundary.entries[0].id
						return
					}
				}
			}
		},
		"duplicate-occurrence": func(graph *Graph) {
			file := &graph.packages[0].files[0]
			file.occurrences.records = append(
				file.occurrences.records,
				file.occurrences.records[0],
			)
		},
		"noncanonical-child-edge": func(graph *Graph) {
			for fileIndex := range graph.packages[0].files {
				file := &graph.packages[0].files[fileIndex]
				for occurrenceIndex := range file.occurrences.records {
					occurrence := &file.occurrences.records[occurrenceIndex]
					if occurrence.parentKind == 0 {
						continue
					}
					occurrence.edge = catalog.EdgeInvalid
					return
				}
			}
			t.Fatal("fixture has no child occurrence")
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			graph := buildStructuralFixture(t)
			mutate(graph)
			if err := Validate(graph); err == nil {
				t.Fatal("corrupted structural graph passed admission")
			}
		})
	}
}

func TestStructuralGraphCollectionsDoNotExposeBackingStorage(t *testing.T) {
	graph := buildStructuralFixture(t)
	wantPackages := graph.PackageCount()
	wantDefinitions := len(graph.ResidentDefinitions())
	wantOccurrences := len(graph.ResidentOccurrences())
	wantCensus := len(graph.DefinitionCensus())

	packages := collectPackageGraphs(t, graph)
	packages[0] = PackageGraph{}
	files := collectPackageGraphs(t, graph)[0].Files()
	files[0] = FileGraph{}
	definitions := graph.ResidentDefinitions()
	definitions[0] = ImplementationDefinition{}
	occurrences := graph.ResidentOccurrences()
	occurrences[0] = Occurrence{}
	definitionCensus := graph.DefinitionCensus()
	definitionCensus[0] = DefinitionCensusRecord{}
	packageDefinitions := collectPackageGraphs(t, graph)[0].Definitions()
	packageDefinitions[0] = ImplementationDefinition{}
	ownerMembers := collectPackageGraphs(t, graph)[0].Files()[0].Owner().Members()
	ownerMembers[0] = identity.OccurrenceID{}

	current := collectPackageGraphs(t, graph)
	if graph.PackageCount() != wantPackages ||
		len(graph.ResidentDefinitions()) != wantDefinitions ||
		len(graph.ResidentOccurrences()) != wantOccurrences ||
		len(graph.DefinitionCensus()) != wantCensus ||
		current[0].ID().IsZero() ||
		graph.DefinitionCensus()[0].ID().IsZero() ||
		graph.ResidentDefinitions()[0].ID().IsZero() ||
		graph.ResidentOccurrences()[0].ID().IsZero() ||
		current[0].Definitions()[0].ID().IsZero() ||
		current[0].Files()[0].Owner().Members()[0].IsZero() {
		t.Fatal("structural accessor exposed canonical backing storage")
	}
}

func collectPackageGraphs(
	t *testing.T,
	graph *Graph,
) []PackageGraph {
	t.Helper()
	var packages []PackageGraph
	if err := graph.VisitPackages(func(pkg PackageGraph) error {
		packages = append(packages, pkg)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return packages
}

func buildStructuralFixture(t *testing.T) *Graph {
	t.Helper()
	directory := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/structure\n\ngo 1.26.0\n",
		"fixture.go": `package structurefixture

var Value = makeValue()

func makeValue() int {
	return 1
}

func Outer() func(int) int {
	return func(value int) int {
		return Value + value
	}
}
`,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name), []byte(content), 0o644,
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
		if pkg.Disposition() != source.DispositionOrdinarySource {
			continue
		}
		for _, file := range pkg.Files() {
			files = append(files, file.ID())
		}
	}
	request, err := source.NewHydrationRequest(files, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.HydrateUniverse(universe, request); err != nil {
		t.Fatal(err)
	}
	graph, _, err := Build(universe)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(graph); err != nil {
		t.Fatal(err)
	}
	return graph
}
