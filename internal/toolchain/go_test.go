package toolchain

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

func TestResolveGoProducesRelocatableSemanticIdentity(t *testing.T) {
	path, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	first, err := ResolveGo(path, sharedRealToolCache(t))
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), filepath.Base(path))
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	second, err := ResolveGo(link, sharedRealToolCache(t))
	if err != nil {
		t.Fatal(err)
	}

	if first.Identity() != second.Identity() {
		t.Fatalf("relocation changed Go identity: %#v != %#v", first.Identity(), second.Identity())
	}
	identity := first.Identity().String()
	if strings.Contains(identity, first.Path()) || strings.Contains(identity, first.Root()) {
		t.Fatalf("semantic identity contains a machine path: %q", identity)
	}
	if first.Version() == "" || first.Root() == "" || first.Path() == "" || !first.Valid() {
		t.Fatalf("resolved Go tool is incomplete: %#v", first)
	}
}

func TestResolveGoRejectsInvalidReportedVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(t.TempDir(), "selected-go")
	source := "#!/bin/sh\n" +
		"if [ \"$1\" = env ]; then printf '{\"GOROOT\":\"/invalid\",\"GOVERSION\":\"not-go\",\"GOOS\":\"linux\",\"GOARCH\":\"amd64\"}\\n'; exit 0; fi\n" +
		"exec " + shellQuote(realGo) + " \"$@\"\n"
	if err := os.WriteFile(fixture, []byte(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveGo(fixture, testCacheRoot(t)); err == nil {
		t.Fatal("invalid reported Go version was accepted")
	}
}

func TestResolveGoRejectsToolOutsideCompilerFrontendVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	directory := t.TempDir()
	root := writeGoRoot(t, filepath.Join(directory, "root"), "selected")
	fixture := filepath.Join(directory, "selected-go")
	source := `#!/bin/sh
if [ "$1" = env ]; then
  printf '{"GOROOT":"%s","GOVERSION":"go1.25.0","GOOS":"linux","GOARCH":"amd64","GOTOOLDIR":"%s/pkg/tool/linux_amd64"}\n' "$GOTOTS_TEST_GOROOT" "$GOTOTS_TEST_GOROOT"
  exit 0
fi
exit 99
`
	if err := os.WriteFile(fixture, []byte(source), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOTOTS_TEST_GOROOT", root)
	if _, err := ResolveGo(fixture, testCacheRoot(t)); err == nil ||
		!strings.Contains(err.Error(), "compiler frontend version") {
		t.Fatalf("frontend/tool version mismatch error = %v", err)
	}
}

func TestResolveGoRejectsExecutableMutationDuringIdentityDiscovery(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("self-mutating shell fixture is Unix-only")
	}
	fixture := filepath.Join(t.TempDir(), "selected-go")
	source := `#!/bin/sh
if [ "$1" = env ]; then
  printf '\n# mutated during identity discovery\n' >> "$0"
  printf '{"GOROOT":"/not-used","GOVERSION":"go1.26.4","GOOS":"linux","GOARCH":"amd64","GOTOOLDIR":"/not-used/pkg/tool/linux_amd64"}\n'
  exit 0
fi
exit 99
`
	if err := os.WriteFile(fixture, []byte(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveGo(fixture, testCacheRoot(t)); err == nil {
		t.Fatal("Go executable mutation during identity discovery was accepted")
	}
}

func TestResolvedGoFailsClosedAfterExecutableDrift(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(t.TempDir(), "selected-go")
	source := "#!/bin/sh\nexec " + shellQuote(realGo) + " \"$@\"\n"
	if err := os.WriteFile(fixture, []byte(source), 0o755); err != nil {
		t.Fatal(err)
	}
	selected, err := ResolveGo(fixture, sharedRealToolCache(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.WriteFile(selected.executable.sealedPath, []byte(source), 0o555); err != nil {
			t.Errorf("restore sealed fixture: %v", err)
		}
		if err := os.Chmod(selected.executable.sealedPath, 0o555); err != nil {
			t.Errorf("restore sealed fixture mode: %v", err)
		}
	})
	if err := os.Chmod(selected.executable.sealedPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(selected.executable.sealedPath, append([]byte(source), '#', '\n'), 0o755); err != nil {
		t.Fatal(err)
	}
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		selected.Version(),
		selected.DefaultGOOS(),
		selected.DefaultGOARCH(),
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selected.Output(context.Background(), t.TempDir(), profile, "version"); err == nil {
		t.Fatal("mutated Go executable was executed")
	}
}

func TestResolvedGoSealsSelectedRootContract(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	directory := t.TempDir()
	wrapper := writeIdentityGo(t, directory)
	firstRoot := writeGoRoot(t, filepath.Join(directory, "first"), "first")
	secondRoot := writeGoRoot(t, filepath.Join(directory, "second"), "second")
	t.Setenv("GOTOTS_TEST_GOROOT", firstRoot)
	first := resolveGoForTest(t, wrapper)
	t.Setenv("GOTOTS_TEST_GOROOT", secondRoot)
	second := resolveGoForTest(t, wrapper)
	if first.Identity() == second.Identity() {
		t.Fatal("different selected GOROOT contracts share one semantic identity")
	}
	if first.Root() == firstRoot || withinRoot(firstRoot, first.Root()) {
		t.Fatalf("resolved Go retained mutable selected root %q", first.Root())
	}
	relative := filepath.Join("lib", "time", "zoneinfo.zip")
	if err := os.WriteFile(filepath.Join(firstRoot, relative), []byte("mutated"), 0o644); err != nil {
		t.Fatal(err)
	}
	sealed, err := os.ReadFile(filepath.Join(first.Root(), relative))
	if err != nil {
		t.Fatal(err)
	}
	if string(sealed) != "first" {
		t.Fatalf("sealed Go root changed with source root: %q", sealed)
	}
	if err := first.VerifyComplete(); err != nil {
		t.Fatalf("source-root drift affected sealed root: %v", err)
	}
}

func TestResolveGoRejectsCorruptPreexistingRootSnapshot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	directory := t.TempDir()
	cacheRoot := filepath.Join(directory, ".temp", "cache", "toolchain")
	wrapper := writeIdentityGo(t, directory)
	sourceRoot := writeGoRoot(t, filepath.Join(directory, "source"), "selected")
	t.Setenv("GOTOTS_TEST_GOROOT", sourceRoot)
	selected, err := ResolveGo(wrapper, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(selected.Root(), "VERSION"), []byte("corrupt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveGo(wrapper, cacheRoot); err == nil {
		t.Fatal("corrupt preexisting Go root snapshot was reopened")
	}
}

func TestRootSnapshotRejectsCandidateMutationBeforePublication(t *testing.T) {
	directory := t.TempDir()
	sourceRoot := writeGoRoot(t, filepath.Join(directory, "source"), "selected")
	toolDirectory := filepath.Join(sourceRoot, "pkg", "tool", "linux_amd64")
	cacheRoot := filepath.Join(directory, ".temp", "cache", "toolchain")
	_, err := sealRootContractWithHook(
		sourceRoot,
		toolDirectory,
		cacheRoot,
		func(candidateRoot string) error {
			return os.WriteFile(filepath.Join(candidateRoot, "VERSION"), []byte("corrupt\n"), 0o644)
		},
	)
	if err == nil {
		t.Fatal("mutated Go root candidate was published")
	}
	entries, readErr := os.ReadDir(filepath.Join(cacheRoot, "go-roots"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("failed Go root candidate left published entries: %v", entries)
	}
}

func TestRootSnapshotRejectsEscapingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link fixture is Unix-only")
	}
	directory := t.TempDir()
	root := writeGoRoot(t, filepath.Join(directory, "root"), "selected")
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "src", "escape")); err != nil {
		t.Fatal(err)
	}
	_, err := sealRootContract(
		root,
		filepath.Join(root, "pkg", "tool", "linux_amd64"),
		filepath.Join(directory, ".temp", "cache"),
	)
	if err == nil {
		t.Fatal("escaping Go root symbolic link was accepted")
	}
}

func TestResolvedGoIdentityNormalizesRelocatedAbsoluteRootSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("absolute symlink fixture is Unix-only")
	}
	directory := t.TempDir()
	wrapper := writeIdentityGo(t, directory)
	firstRoot := writeGoRoot(t, filepath.Join(directory, "first"), "same")
	secondRoot := writeGoRoot(t, filepath.Join(directory, "second"), "same")
	for _, root := range []string{firstRoot, secondRoot} {
		target := filepath.Join(root, "lib", "time", "zoneinfo.zip")
		link := filepath.Join(root, "src", "runtime", "zoneinfo.link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOTOTS_TEST_GOROOT", firstRoot)
	first := resolveGoForTest(t, wrapper)
	t.Setenv("GOTOTS_TEST_GOROOT", secondRoot)
	second := resolveGoForTest(t, wrapper)
	if first.Identity() != second.Identity() {
		t.Fatalf(
			"relocated equivalent roots changed semantic identity: %#v != %#v",
			first.Identity(),
			second.Identity(),
		)
	}
}

func TestResolvedExecutableIsSealedUnderSelectedCacheRoot(t *testing.T) {
	path, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := sharedRealToolCache(t)
	selected, err := ResolveGo(path, cacheRoot)
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cacheRoot, selected.executable.SealedPath())
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatalf("sealed executable %q is outside selected cache %q", selected.executable.SealedPath(), cacheRoot)
	}
}

func TestSelectedGoDoesNotInheritAmbientPathOrAdmitCgo(t *testing.T) {
	path, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := ResolveGo(path, sharedRealToolCache(t))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		selected.Version(), selected.DefaultGOOS(), selected.DefaultGOARCH(), false, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "/ambient/tool/directory")
	t.Setenv("TMPDIR", "/ambient/temp/directory")
	environment := selected.Environment(profile)
	for _, entry := range environment {
		if strings.HasPrefix(entry, "PATH=") && entry != "PATH="+selected.executable.Directory() {
			t.Fatalf("selected PATH inherited ambient tools: %q", entry)
		}
		if strings.HasPrefix(entry, "TMPDIR=") && entry != "TMPDIR="+selected.temporaryRoot {
			t.Fatalf("selected temporary root inherited ambient state: %q", entry)
		}
	}
	cgoProfile, err := environmentcontract.NewBuildProfileForToolchain(
		selected.Version(), selected.DefaultGOOS(), selected.DefaultGOARCH(), true, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := selected.ValidateProfile(cgoProfile); err == nil {
		t.Fatal("cgo profile without an external-tool contract was accepted")
	}
}

func TestGoCommandsDoNotReinspectCompleteRoot(t *testing.T) {
	path, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := ResolveGo(path, sharedRealToolCache(t))
	if err != nil {
		t.Fatal(err)
	}
	before := selected.FullRootInspectionCount()
	for range 3 {
		if _, err := selected.HostOutput(context.Background(), t.TempDir(), "version"); err != nil {
			t.Fatal(err)
		}
	}
	if after := selected.FullRootInspectionCount(); after != before {
		t.Fatalf("three Go commands added full-root walks: before=%d after=%d", before, after)
	}
	if err := selected.VerifyComplete(); err != nil {
		t.Fatal(err)
	}
	if after := selected.FullRootInspectionCount(); after != before+1 {
		t.Fatalf("one final verification added %d full-root walks, want 1", after-before)
	}
}

func TestSealExecutableRejectsHashCopyDrift(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "selected-tool")
	if err := os.WriteFile(source, []byte("first"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := fileDigest(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("second"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := sealExecutable(
		filepath.Join(root, ".temp", "cache"), source, digest, "selected-tool",
	); err == nil {
		t.Fatal("source bytes changed between hashing and sealing were accepted")
	}
}

func resolveGoForTest(t *testing.T, path string) Go {
	t.Helper()
	selected, err := ResolveGo(path, testCacheRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return selected
}

func testCacheRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), ".temp", "cache", "toolchain")
}

func sharedRealToolCache(t *testing.T) string {
	t.Helper()
	if selected := os.Getenv("GOTOTS_TEST_REAL_TOOL_CACHE"); selected != "" {
		return selected
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, ".temp", "cache", "toolchain-tests")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func writeIdentityGo(t *testing.T, directory string) string {
	t.Helper()
	path := filepath.Join(directory, "selected-go")
	source := `#!/bin/sh
if [ "$1" = env ]; then
  printf '{"GOROOT":"%s","GOVERSION":"go1.26.4","GOOS":"linux","GOARCH":"amd64","GOTOOLDIR":"%s/pkg/tool/linux_amd64"}\n' "$GOTOTS_TEST_GOROOT" "$GOTOTS_TEST_GOROOT"
  exit 0
fi
exit 99
`
	if err := os.WriteFile(path, []byte(source), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeGoRoot(t *testing.T, root string, sourceValue string) string {
	t.Helper()
	for _, directory := range []string{
		filepath.Join(root, "src", "runtime"),
		filepath.Join(root, "pkg", "tool", "linux_amd64"),
		filepath.Join(root, "lib", "time"),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, content := range map[string]string{
		filepath.Join(root, "src", "go.mod"):                         "module std\n\ngo 1.26.4\n",
		filepath.Join(root, "src", "runtime", "runtime.go"):          "package runtime\n",
		filepath.Join(root, "lib", "time", "zoneinfo.zip"):           sourceValue,
		filepath.Join(root, "go.env"):                                "GOTOOLCHAIN=local\n",
		filepath.Join(root, "VERSION"):                               "go1.26.4\n",
		filepath.Join(root, "pkg", "tool", "linux_amd64", "compile"): "#!/bin/sh\nexit 0\n",
		filepath.Join(root, "pkg", "tool", "linux_amd64", "link"):    "#!/bin/sh\nexit 0\n",
	} {
		mode := os.FileMode(0o644)
		if filepath.Dir(path) == filepath.Join(root, "pkg", "tool", "linux_amd64") {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
