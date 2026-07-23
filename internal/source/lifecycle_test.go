package source

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestSelectiveHydrationAndFinalizationLifecycle(t *testing.T) {
	dir := writeSourceFixture(t)
	universe := resolveFixture(t, dir)
	app := loadedByPath(t, universe, "example.com/app/cmd/app")
	dependency := loadedByPath(t, universe, "example.com/app/internal/dep")

	assertNoTransientEvidence(t, universe)
	if len(universe.Roots()) != 1 || universe.Roots()[0] != app {
		t.Fatalf("roots = %v, want only application package", packagePaths(universe.Roots()))
	}
	request, err := NewHydrationRequest(
		[]identity.FileID{app.Files()[0].ID()},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := HydrateUniverse(universe, request); err != nil {
		t.Fatal(err)
	}
	if got, want := universe.HydrationStats(), (HydrationStats{
		SemanticPackages: 1,
		LocalFiles:       1,
		LocalBytes:       int64(len(app.Files()[0].SelectedBytes())),
	}); got != want {
		t.Fatalf("hydration stats = %+v, want %+v", got, want)
	}
	if dependency.Types() != nil || dependency.CheckerView() != nil {
		t.Fatal("unselected dependency retained a checker package or types.Info")
	}
	for _, file := range dependency.Files() {
		if file.PhysicalSyntax() != nil || file.CheckedSyntax() != nil ||
			len(file.SelectedBytes()) != 0 {
			t.Fatalf("unselected dependency file %s retained source interiors", file.ID())
		}
	}

	resolutionFingerprint := universe.ResolutionFingerprint()
	workspace, err := Finalize(universe)
	if err != nil {
		t.Fatal(err)
	}
	if !universe.Finalized() || universe.Hydrated() {
		t.Fatalf("finalized=%t hydrated=%t", universe.Finalized(), universe.Hydrated())
	}
	assertNoTransientEvidence(t, universe)
	if workspace.ResolutionFingerprint() != resolutionFingerprint {
		t.Fatal("finalization changed the resolution fingerprint")
	}
	finalApp := finalizedByPath(t, workspace, app.ID().ImportPath())
	if finalApp.Files()[0].ByteDigest() != app.Files()[0].ByteDigest() ||
		finalApp.Files()[0].EffectiveGoVersion() == "" {
		t.Fatal("finalized file lost selected-byte or language-version evidence")
	}
}

func TestHydrationFailureIsAtomic(t *testing.T) {
	dir := writeSourceFixture(t)
	universe := resolveFixture(t, dir)
	app := loadedByPath(t, universe, "example.com/app/cmd/app")
	request, err := NewHydrationRequest(
		[]identity.FileID{app.Files()[0].ID()},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "cmd", "app", "main.go"),
		[]byte("package main\nfunc main() {}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := HydrateUniverse(universe, request); err == nil ||
		!strings.Contains(err.Error(), "bytes changed after resolution") {
		t.Fatalf("HydrateUniverse error = %v, want byte-drift rejection", err)
	}
	if universe.Hydrated() || universe.Finalized() {
		t.Fatalf("failed hydration left state hydrated=%t finalized=%t", universe.Hydrated(), universe.Finalized())
	}
	assertNoTransientEvidence(t, universe)
}

func TestHydrationForkDoesNotMutateResolvedBase(t *testing.T) {
	dir := writeSourceFixture(t)
	base := resolveFixture(t, dir)
	app := loadedByPath(t, base, "example.com/app/cmd/app")
	request, err := NewHydrationRequest(
		[]identity.FileID{app.Files()[0].ID()},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		fork, err := ForkForHydration(base, []identity.PackageID{app.ID()})
		if err != nil {
			t.Fatal(err)
		}
		if loadedByPath(t, fork, app.ID().ImportPath()) == app {
			t.Fatal("selected package was shared rather than cloned")
		}
		if err := HydrateUniverse(fork, request); err != nil {
			t.Fatal(err)
		}
		if fork.HydrationStats().LocalFiles != 1 {
			t.Fatalf("fork hydration stats = %+v", fork.HydrationStats())
		}
		assertNoTransientEvidence(t, base)
		if err := DiscardHydratedUniverse(fork); err != nil {
			t.Fatal(err)
		}
		assertNoTransientEvidence(t, fork)
	}
	if base.Finalized() || base.Hydrated() {
		t.Fatal("fork lifecycle mutated the resolved base")
	}
}

func TestResolutionIdentityAndFingerprintAreRelocatable(t *testing.T) {
	first := resolveFixture(t, writeSourceFixture(t))
	second := resolveFixture(t, writeSourceFixture(t))
	if first.ResolutionFingerprint() != second.ResolutionFingerprint() {
		t.Fatalf(
			"relocated identical modules have fingerprints %s and %s",
			first.ResolutionFingerprint(),
			second.ResolutionFingerprint(),
		)
	}
	firstIDs := packageAndFileIDs(first)
	secondIDs := packageAndFileIDs(second)
	if !reflect.DeepEqual(firstIDs, secondIDs) {
		t.Fatalf("relocated identities differ:\nfirst=%v\nsecond=%v", firstIDs, secondIDs)
	}
}

func TestFinalizedCollectionsDoNotExposeBackingStorage(t *testing.T) {
	universe := resolveFixture(t, writeSourceFixture(t))
	workspace, err := FinalizeResolved(universe)
	if err != nil {
		t.Fatal(err)
	}
	packages := workspace.Packages()
	originalPackage := packages[0]
	packages[0] = nil
	if workspace.Packages()[0] != originalPackage {
		t.Fatal("Workspace.Packages exposes backing storage")
	}
	roots := workspace.Roots()
	originalRoot := roots[0]
	roots[0] = nil
	if workspace.Roots()[0] != originalRoot {
		t.Fatal("Workspace.Roots exposes backing storage")
	}
	pkg := finalizedByPath(t, workspace, "example.com/app/cmd/app")
	imports := pkg.Imports()
	originalImport := imports[0]
	imports[0] = "mutated"
	if pkg.Imports()[0] != originalImport {
		t.Fatal("Package.Imports exposes backing storage")
	}
	files := pkg.Files()
	originalFile := files[0]
	files[0] = nil
	if pkg.Files()[0] != originalFile {
		t.Fatal("Package.Files exposes backing storage")
	}
}

func TestInputKindCatalogIsClosed(t *testing.T) {
	tests := map[string]InputKind{
		"x.c": InputC, "x.cc": InputCXX, "x.cpp": InputCXX,
		"x.cxx": InputCXX, "x.m": InputObjectiveC, "x.h": InputHeader,
		"x.hh": InputHeader, "x.hpp": InputHeader, "x.hxx": InputHeader,
		"x.f": InputFortran, "x.F": InputFortran, "x.for": InputFortran,
		"x.f90": InputFortran, "x.s": InputAssembly, "x.S": InputAssembly,
		"x.sx": InputAssembly, "x.swig": InputSWIG,
		"x.swigcxx": InputSWIGCXX, "x.syso": InputSyso,
	}
	seen := map[InputKind]bool{}
	for path, want := range tests {
		got, err := inputKindForPath(path)
		if err != nil || got != want {
			t.Errorf("inputKindForPath(%q) = %s, %v; want %s", path, got, err, want)
		}
		seen[got] = true
	}
	for kind := InputC; kind < numInputKinds; kind++ {
		if !kind.Valid() || kind.String() == "" {
			t.Errorf("input kind %d is not a valid named member", kind)
		}
		if kind != InputEmbed && !seen[kind] {
			t.Errorf("input kind %s has no physical extension case", kind)
		}
	}
	if got, err := inputKindForPath("x.rs"); err == nil || got.Valid() {
		t.Fatalf("unknown supplemental input admitted as %s, %v", got, err)
	}
}

func TestPackageAdmissionRejectsNoncanonicalEvidence(t *testing.T) {
	universe := resolveFixture(t, writeSourceFixture(t))
	loaded := loadedByPath(t, universe, "example.com/app/cmd/app")
	valid, err := finalizePackage(loaded)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*Package){
		"invalid provenance": func(pkg *Package) {
			pkg.provenance = ProvenanceStandardLibrary
		},
		"unsorted imports": func(pkg *Package) {
			pkg.imports = []string{"z", "a"}
		},
		"duplicate imports": func(pkg *Package) {
			pkg.imports = []string{"a", "a"}
		},
		"zero file digest": func(pkg *Package) {
			pkg.files[0].byteDigest = SourceSpanHash{}
		},
		"checked intrinsic": func(pkg *Package) {
			pkg.disposition = DispositionUnsafeIntrinsic
			pkg.hasCheckedView = true
		},
		"cgo without checked view": func(pkg *Package) {
			pkg.files[0].cgoOriginal = true
			pkg.hasCheckedView = false
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := cloneFinalizedPackage(valid)
			mutate(candidate)
			if _, err := finishPackage(candidate); err == nil {
				t.Fatal("mutated package passed admission")
			}
		})
	}
	if _, err := finishPackage(cloneFinalizedPackage(valid)); err != nil {
		t.Fatalf("valid package rejected: %v", err)
	}
}

func writeSourceFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixtureFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26.0\n")
	writeFixtureFile(
		t,
		dir,
		"internal/dep/dep.go",
		"package dep\n\nconst Value = 7\n",
	)
	writeFixtureFile(
		t,
		dir,
		"cmd/app/main.go",
		"package main\n\nimport \"example.com/app/internal/dep\"\n\nfunc main() { println(dep.Value) }\n",
	)
	return dir
}

func writeFixtureFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func resolveFixture(t *testing.T, dir string) *Universe {
	t.Helper()
	universe, err := ResolveUniverse(Request{
		Dir: dir, Patterns: []string{"./cmd/app"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return universe
}

func loadedByPath(t *testing.T, universe *Universe, path string) *LoadedPackage {
	t.Helper()
	for _, pkg := range universe.Packages() {
		if pkg.ID().ImportPath() == path {
			return pkg
		}
	}
	t.Fatalf("package %s absent from universe", path)
	return nil
}

func finalizedByPath(t *testing.T, workspace *Workspace, path string) *Package {
	t.Helper()
	for _, pkg := range workspace.Packages() {
		if pkg.ID().ImportPath() == path {
			return pkg
		}
	}
	t.Fatalf("package %s absent from workspace", path)
	return nil
}

func assertNoTransientEvidence(t *testing.T, universe *Universe) {
	t.Helper()
	if universe.Fset() != nil || universe.HydrationStats() != (HydrationStats{}) {
		t.Fatalf("universe retains transient evidence: %+v", universe.HydrationStats())
	}
	for _, pkg := range universe.Packages() {
		if pkg.Types() != nil || pkg.CheckerView() != nil ||
			len(pkg.CheckedDeclarations()) != 0 {
			t.Fatalf("package %s retains checker evidence", pkg.ID())
		}
		for _, file := range pkg.Files() {
			if file.PhysicalSyntax() != nil ||
				file.PhysicalFileSet() != nil ||
				file.CheckedSyntax() != nil ||
				len(file.SelectedBytes()) != 0 {
				t.Fatalf("file %s retains syntax or bytes", file.ID())
			}
		}
	}
}

func packagePaths(packages []*LoadedPackage) []string {
	out := make([]string, 0, len(packages))
	for _, pkg := range packages {
		out = append(out, pkg.ID().ImportPath())
	}
	return out
}

func packageAndFileIDs(universe *Universe) []string {
	var out []string
	for _, pkg := range universe.Packages() {
		out = append(out, pkg.ID().String())
		for _, file := range pkg.Files() {
			out = append(out, file.ID().String())
		}
	}
	return out
}

func cloneFinalizedPackage(pkg *Package) *Package {
	out := *pkg
	out.imports = append([]string(nil), pkg.imports...)
	out.embedPatterns = append([]string(nil), pkg.embedPatterns...)
	out.inputs = append([]Input(nil), pkg.inputs...)
	out.files = make([]*File, 0, len(pkg.files))
	for _, file := range pkg.files {
		copy := *file
		out.files = append(out.files, &copy)
	}
	return &out
}
