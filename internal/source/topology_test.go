package source

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestPackageTopologyCarriesTypedNamesEdgesAndInitialization(t *testing.T) {
	workspace := resolveTopologyFixture(t)
	topology := workspace.PackageTopology()
	nodes := topology.Nodes()
	if len(nodes) != len(workspace.Packages()) {
		t.Fatalf(
			"topology nodes = %d, packages = %d",
			len(nodes), len(workspace.Packages()),
		)
	}
	byPath := map[string]PackageNode{}
	for _, node := range nodes {
		byPath[node.Package().ImportPath()] = node
	}
	if got := byPath["example.com/topology/alpha"].DeclaredName(); got != "foundation" {
		t.Fatalf("alpha declared name = %q, want foundation", got)
	}
	for _, path := range []string{"builtin", "unsafe"} {
		node, present := byPath[path]
		if !present ||
			node.Initialization().Kind() != PackageInitializationNone {
			t.Fatalf("%s initialization = %+v, want none", path, node)
		}
	}
	var initialized []PackageNode
	for _, node := range nodes {
		if _, ok := node.Initialization().GoOrdinal(); ok {
			initialized = append(initialized, node)
		}
	}
	sort.Slice(initialized, func(left, right int) bool {
		leftOrdinal, _ := initialized[left].Initialization().GoOrdinal()
		rightOrdinal, _ := initialized[right].Initialization().GoOrdinal()
		return leftOrdinal < rightOrdinal
	})
	gotOrder := make([]string, 0, len(initialized))
	for ordinal, node := range initialized {
		actual, _ := node.Initialization().GoOrdinal()
		if actual != ordinal {
			t.Fatalf("initialization ordinal = %d, want %d", actual, ordinal)
		}
		gotOrder = append(gotOrder, node.Package().ImportPath())
	}
	wantOrder := []string{
		"example.com/topology/alpha",
		"example.com/topology/beta",
		"example.com/topology/cmd/app",
	}
	if !equalStrings(gotOrder, wantOrder) {
		t.Fatalf("initialization order = %v, want %v", gotOrder, wantOrder)
	}
	actualEdges := map[string]bool{}
	for _, edge := range topology.Imports() {
		actualEdges[edge.Importer().ImportPath()+"->"+
			edge.Imported().ImportPath()] = true
	}
	wantEdges := []string{
		"example.com/topology/beta->example.com/topology/alpha",
		"example.com/topology/cmd/app->example.com/topology/alpha",
		"example.com/topology/cmd/app->example.com/topology/beta",
		"example.com/topology/cmd/app->unsafe",
	}
	if len(actualEdges) != len(wantEdges) {
		t.Fatalf("typed import edges = %v, want %v", actualEdges, wantEdges)
	}
	for _, edge := range wantEdges {
		if !actualEdges[edge] {
			t.Fatalf("typed import edge %s is absent", edge)
		}
	}
}

func TestPackageInitializationKindIDsArePinned(t *testing.T) {
	if PackageInitializationInvalid != 0 ||
		PackageInitializationNone != 1 ||
		PackageInitializationGo != 2 {
		t.Fatalf(
			"package initialization IDs changed: invalid=%d none=%d go=%d",
			PackageInitializationInvalid,
			PackageInitializationNone,
			PackageInitializationGo,
		)
	}
	for kind := PackageInitializationNone; kind <= PackageInitializationGo; kind++ {
		if !kind.Valid() || kind.String() == "" {
			t.Fatalf("package initialization kind %d is not closed", kind)
		}
	}
}

func TestPackageTopologyRejectsMutations(t *testing.T) {
	workspace := resolveTopologyFixture(t)
	originals := workspace.Packages()
	tests := map[string]func([]*Package){
		"wrong name": func(packages []*Package) {
			packageByImportPath(t, packages, "example.com/topology/alpha").name = "_"
		},
		"wrong initialization disposition": func(packages []*Package) {
			packageByImportPath(t, packages, "unsafe").initialization =
				mustGoInitialization(t, 0)
		},
		"wrong initialization order": func(packages []*Package) {
			alpha := packageByImportPath(
				t, packages, "example.com/topology/alpha",
			)
			beta := packageByImportPath(
				t, packages, "example.com/topology/beta",
			)
			alpha.initialization, beta.initialization =
				beta.initialization, alpha.initialization
		},
		"wrong importer": func(packages []*Package) {
			app := packageByImportPath(
				t, packages, "example.com/topology/cmd/app",
			)
			app.imports[0].importer = packageByImportPath(
				t, packages, "example.com/topology/alpha",
			).id
		},
		"duplicate edge": func(packages []*Package) {
			app := packageByImportPath(
				t, packages, "example.com/topology/cmd/app",
			)
			app.imports = append(app.imports, app.imports[len(app.imports)-1])
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := clonePackageSlice(originals)
			mutate(candidate)
			if err := validatePackageTopology(candidate); err == nil {
				t.Fatal("mutated topology passed validation")
			}
		})
	}
}

func resolveTopologyFixture(t *testing.T) *Workspace {
	t.Helper()
	directory := t.TempDir()
	writeTopologyFile(
		t, directory, "go.mod",
		"module example.com/topology\n\ngo 1.26.0\n",
	)
	writeTopologyFile(
		t, directory, "alpha/alpha.go",
		"package foundation\n\nconst Value = 1\n",
	)
	writeTopologyFile(
		t, directory, "beta/beta.go",
		"package beta\n\nimport foundation \"example.com/topology/alpha\"\n\nvar Value = foundation.Value\n",
	)
	writeTopologyFile(
		t, directory, "cmd/app/main.go",
		"package main\n\nimport (\n\tfoundation \"example.com/topology/alpha\"\n\t\"example.com/topology/beta\"\n\t\"unsafe\"\n)\n\nvar _ = foundation.Value + beta.Value\nvar _ = unsafe.Sizeof(0)\nfunc main() {}\n",
	)
	universe, err := ResolveUniverse(Request{
		Dir: directory, Patterns: []string{"./cmd/app"},
	})
	if err != nil {
		t.Fatal(err)
	}
	workspace, err := FinalizeResolved(universe)
	if err != nil {
		t.Fatal(err)
	}
	return workspace
}

func writeTopologyFile(
	t *testing.T,
	directory string,
	name string,
	content string,
) {
	t.Helper()
	path := filepath.Join(directory, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func packageByImportPath(
	t *testing.T,
	packages []*Package,
	importPath string,
) *Package {
	t.Helper()
	for _, pkg := range packages {
		if pkg.id.ImportPath() == importPath {
			return pkg
		}
	}
	t.Fatalf("package %s is absent", importPath)
	return nil
}

func clonePackageSlice(packages []*Package) []*Package {
	out := make([]*Package, 0, len(packages))
	for _, pkg := range packages {
		cloned := *pkg
		cloned.imports = append([]PackageImport(nil), pkg.imports...)
		out = append(out, &cloned)
	}
	return out
}

func mustGoInitialization(
	t *testing.T,
	ordinal int,
) PackageInitialization {
	t.Helper()
	initialization, err := goPackageInitialization(ordinal)
	if err != nil {
		t.Fatal(err)
	}
	return initialization
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
