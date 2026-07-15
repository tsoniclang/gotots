// Shared fixture-repository helpers for the census contract tests.
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
		"schemaVersion": 3,
		"upstream":      "fixture",
		"goModule":      "example.com/fix",
		"revision":      revision,
		"toolchain": map[string]any{
			"version":            toolchain.Version,
			"goos":               toolchain.GOOS,
			"goarch":             toolchain.GOARCH,
			"goExecutableSha256": toolchain.GoExecutableSha256,
			"gorootSrcDigest":    toolchain.GorootSrcDigest,
			"gorootToolDigest":   toolchain.GorootToolDigest,
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
