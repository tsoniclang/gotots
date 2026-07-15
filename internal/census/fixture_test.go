package census

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
)

// measuredToolchain caches the (expensive) toolchain identity measurement
// shared by every fixture test.
var measuredToolchain = sync.OnceValues(func() (*pinning.Toolchain, error) {
	goExecutable, err := goenv.Locate()
	if err != nil {
		return nil, err
	}
	resolved, err := goenv.Bootstrap(goExecutable)
	if err != nil {
		return nil, err
	}
	return pinning.ToolchainIdentity(resolved)
})

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=fixture", "GIT_AUTHOR_EMAIL=fixture@test",
		"GIT_COMMITTER_NAME=fixture", "GIT_COMMITTER_EMAIL=fixture@test",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

// writeFixtureRepo creates a committed git repository from the file map.
func writeFixtureRepo(t *testing.T, files map[string]string) (dir, revision string) {
	t.Helper()
	dir = t.TempDir()
	// Resolve symlinks (macOS TMPDIR) so path containment checks are exact.
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	dir = resolved
	for path, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "fixture")
	return dir, runGit(t, dir, "rev-parse", "HEAD")
}

// writeFixtureConfig writes a pin and profile for the fixture repository
// and loads the profile.
func writeFixtureConfig(t *testing.T, revision string, prof map[string]any) *profile.Profile {
	t.Helper()
	toolchain, err := measuredToolchain()
	if err != nil {
		t.Fatalf("measure toolchain: %v", err)
	}
	configDir := t.TempDir()

	pin := map[string]any{
		"schemaVersion": 2,
		"upstream":      "fixture",
		"goModule":      "example.com/fix",
		"revision":      revision,
		"toolchain": map[string]any{
			"version":            toolchain.Version,
			"goos":               toolchain.GOOS,
			"goarch":             toolchain.GOARCH,
			"goExecutableSha256": toolchain.GoExecutableSha256,
			"gorootSrcDigest":    toolchain.GorootSrcDigest,
		},
	}
	pinData, err := json.Marshal(pin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "pin.json"), pinData, 0o644); err != nil {
		t.Fatal(err)
	}

	prof["schemaVersion"] = 1
	prof["goModule"] = "example.com/fix"
	prof["pin"] = "pin.json"
	build := map[string]any{
		"name":       "fixture",
		"goos":       toolchain.GOOS,
		"goarch":     toolchain.GOARCH,
		"cgoEnabled": false,
		"tags":       []string{},
	}
	if toolchain.GOARCH == "amd64" {
		build["goamd64"] = "v1"
	}
	prof["buildProfiles"] = []any{build}
	profData, err := json.Marshal(prof)
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(configDir, "profile.json")
	if err := os.WriteFile(profilePath, profData, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := profile.Load(profilePath)
	if err != nil {
		t.Fatalf("load fixture profile: %v", err)
	}
	return loaded
}

// basicFixtureFiles exercises black-box tests, test support, legal duplicate
// declarations, a package path ending in .test, a nested tool module,
// testdata fixtures, hard exclusions, and a production contradiction edge.
func basicFixtureFiles() map[string]string {
	return map[string]string{
		"go.mod": "module example.com/fix\n\ngo 1.26\n",
		"a/a.go": `package a

import "example.com/fix/b.test"

var _ = btest.Value
var _ = 2

func init() {}
func init() {}

type Counter struct{ total int }

func (c *Counter) Add(values []int) int {
	for _, value := range values {
		c.total += value
	}
	return c.total
}

func First[T any](values []T) T { return values[0] }

func UseFirst() int {
	values := append([]int(nil), 1, 2)
	return First(values)
}
`,
		"a/a_inpkg_test.go": `package a

func addForTest(left, right int) int { return left + right }
`,
		"a/a_bb_test.go": `package a_test

import (
	"example.com/fix/a"
	"example.com/fix/support"
)

func UseBlackBox() int {
	support.Touch()
	counter := &a.Counter{}
	return counter.Add([]int{1})
}
`,
		"support/support.go": `package support

import "os"

func Touch() { _ = os.Getpid() }
`,
		"b.test/b.go": `package btest

const Value = 7
`,
		"c/c.go": `package c

import "example.com/fix/ex"

func UseExcluded() { ex.Touch() }
`,
		"ex/ex.go": `package ex

import "net/http"

func Touch() { _ = http.MethodGet }
`,
		"_tools/gen/go.mod":  "module example.com/fix/_tools/gen\n\ngo 1.26\n",
		"_tools/gen/main.go": "package main\n\nfunc main() {}\n",
		"a/testdata/fixture.go": `package fixture
`,
	}
}

func basicFixtureProfile() map[string]any {
	return map[string]any{
		"product":       "fixture",
		"ownedRoots":    []string{"a", "b.test", "c"},
		"testOnlyRoots": []string{"support"},
		"hardExcludedRoots": map[string]any{
			"editor-service": []string{"ex"},
		},
		"toolingRoots": []string{"_tools"},
	}
}

func TestCensusFixture(t *testing.T) {
	dir, revision := writeFixtureRepo(t, basicFixtureFiles())
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	// Ambient toolchain/workspace/module state must not influence the run.
	t.Setenv("GOWORK", filepath.Join(dir, "no-such-workspace"))
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOTOOLCHAIN", "go1.1.1")

	result, err := Run(prof, dir, "fixture")
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	report := result.Report

	// Black-box test files keep their semantic package identity while being
	// owned by the package under test.
	var blackBox *FileRecord
	for i := range report.Files {
		if report.Files[i].Path == "a/a_bb_test.go" {
			blackBox = &report.Files[i]
		}
	}
	if blackBox == nil {
		t.Fatal("black-box test file missing from report")
	}
	if blackBox.Package != "example.com/fix/a_test" || blackBox.Owner != "example.com/fix/a" || blackBox.Scope != "test" {
		t.Errorf("black-box identity wrong: %+v", blackBox)
	}

	// Legal duplicate declarations receive position-qualified unique IDs.
	blanks, inits := 0, 0
	for _, d := range report.Declarations {
		if d.File == "a/a.go" && d.Name == "_" {
			blanks++
			if !strings.Contains(d.ID, "::var::_@") {
				t.Errorf("blank var ID not position-qualified: %s", d.ID)
			}
		}
		if d.File == "a/a.go" && d.Name == "init" {
			inits++
			if !strings.Contains(d.ID, "::func::init@") {
				t.Errorf("init ID not position-qualified: %s", d.ID)
			}
		}
	}
	if blanks != 2 || inits != 2 {
		t.Errorf("expected 2 blank vars and 2 inits, got %d and %d", blanks, inits)
	}

	// A legitimate package path ending in .test is ordinary owned source.
	foundBTest := false
	for _, f := range report.Files {
		if f.Package == "example.com/fix/b.test" && f.Scope == "production" {
			foundBTest = true
		}
	}
	if !foundBTest {
		t.Error("package with path ending in .test was not analyzed as production source")
	}

	// Universe: tool-module and testdata files are classified, none lost.
	universe := result.Inventory.Universe
	classes := map[string]string{}
	for _, f := range universe.OutsidePackages {
		classes[f.Path] = f.Class
	}
	if classes["_tools/gen/main.go"] != "tooling" {
		t.Errorf("nested tool module file not classified tooling: %v", classes)
	}
	if classes["a/testdata/fixture.go"] != "testdata" {
		t.Errorf("testdata file not classified: %v", classes)
	}
	if universe.TrackedGoFiles != universe.InPackages+len(universe.OutsidePackages) {
		t.Errorf("universe does not add up: %+v", universe)
	}

	// External attribution: os is a test-only dependency (via support);
	// net/http is reachable only through the hard-excluded package. The
	// -deps contract must supply transitive test-dependency evidence (io is
	// in os's closure).
	externals := map[string]*ExternalUseEvidence{}
	for i := range result.Inventory.External {
		e := &result.Inventory.External[i]
		externals[e.ImportPath] = &ExternalUseEvidence{
			Production: e.ReachableFromProduction, Test: e.ReachableFromTest, ExcludedOnly: e.ExcludedOrUnselectedOnly,
		}
	}
	if e := externals["os"]; e == nil || e.Production || !e.Test {
		t.Errorf("os should be test-only reachable: %+v", e)
	}
	if e := externals["io"]; e == nil || !e.Test {
		t.Errorf("io (transitive test dependency) missing or misattributed: %+v", e)
	}
	if e := externals["net/http"]; e == nil || !e.ExcludedOnly {
		t.Errorf("net/http should be excluded-only: %+v", e)
	}

	// The production import of a hard-excluded package is a recorded
	// contradiction, not a silent reclassification.
	foundEdge := false
	for _, edge := range report.Contradictions {
		if edge.From == "example.com/fix/c" && edge.To == "example.com/fix/ex" &&
			edge.Class == "hard-excluded" && edge.Scope == "production" && edge.File == "c/c.go" {
			foundEdge = true
		}
	}
	if !foundEdge {
		t.Errorf("expected contradiction edge for c -> ex, got %+v", report.Contradictions)
	}

	// Publication is transactional and deterministic.
	out1 := filepath.Join(t.TempDir(), "reports")
	if err := WriteReports(result, out1); err != nil {
		t.Fatalf("write reports: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out1, "manifest.json")); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	second, err := Run(prof, dir, "fixture")
	if err != nil {
		t.Fatalf("second census: %v", err)
	}
	out2 := filepath.Join(t.TempDir(), "reports")
	if err := WriteReports(second, out2); err != nil {
		t.Fatalf("write second reports: %v", err)
	}
	for _, name := range []string{"inventory.json", "census.json"} {
		first, err := os.ReadFile(filepath.Join(out1, name))
		if err != nil {
			t.Fatal(err)
		}
		again, err := os.ReadFile(filepath.Join(out2, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(first) != string(again) {
			t.Errorf("%s is not deterministic across clean runs", name)
		}
	}
}

// ExternalUseEvidence is a test-local view of external attribution.
type ExternalUseEvidence struct {
	Production   bool
	Test         bool
	ExcludedOnly bool
}

func TestCensusRejectsInjectedIgnoredFile(t *testing.T) {
	files := basicFixtureFiles()
	files[".gitignore"] = "injected.go\n"
	dir, revision := writeFixtureRepo(t, files)
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	// The file is ignored, so git status stays clean — but it is not part
	// of the pinned commit and must not be admitted.
	injected := filepath.Join(dir, "a", "injected.go")
	if err := os.WriteFile(injected, []byte("package a\n\nfunc Injected() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if clean, err := pinning.CheckClean(dir); err != nil || !clean {
		t.Fatalf("fixture precondition: checkout should look clean (clean=%v err=%v)", clean, err)
	}

	_, err := Run(prof, dir, "fixture")
	if err == nil || !strings.Contains(err.Error(), "injected.go") {
		t.Fatalf("expected failure naming injected.go, got %v", err)
	}
}

func TestCensusRejectsTamperedToolchainPin(t *testing.T) {
	dir, revision := writeFixtureRepo(t, basicFixtureFiles())
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	digest := []byte(prof.Pin.Toolchain.GoExecutableSha256)
	if digest[0] == 'a' {
		digest[0] = 'b'
	} else {
		digest[0] = 'a'
	}
	prof.Pin.Toolchain.GoExecutableSha256 = string(digest)

	_, err := Run(prof, dir, "fixture")
	if err == nil || !strings.Contains(err.Error(), "refusing to execute") {
		t.Fatalf("expected digest refusal before execution, got %v", err)
	}
}

func TestCensusRejectsOutputInsideSource(t *testing.T) {
	dir, revision := writeFixtureRepo(t, basicFixtureFiles())
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	result, err := Run(prof, dir, "fixture")
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	err = WriteReports(result, filepath.Join(dir, "reports"))
	if err == nil || !strings.Contains(err.Error(), "inside the pinned source tree") {
		t.Fatalf("expected containment refusal, got %v", err)
	}
}

func TestCensusRejectsLocalReplacement(t *testing.T) {
	files := map[string]string{
		"go.mod": "module example.com/fix\n\ngo 1.26\n\nrequire example.com/other v0.0.0\n\nreplace example.com/other => ./other\n",
		"a/a.go": `package a

import "example.com/other/lib"

func Use() int { return lib.Value }
`,
		"other/go.mod":     "module example.com/other\n\ngo 1.26\n",
		"other/lib/lib.go": "package lib\n\nconst Value = 1\n",
	}
	dir, revision := writeFixtureRepo(t, files)
	prof := writeFixtureConfig(t, revision, map[string]any{
		"product":           "fixture",
		"ownedRoots":        []string{"a"},
		"hardExcludedRoots": map[string]any{},
	})

	_, err := Run(prof, dir, "fixture")
	if err == nil || !strings.Contains(err.Error(), "unpinned local replacement") {
		t.Fatalf("expected local-replacement refusal, got %v", err)
	}
}
