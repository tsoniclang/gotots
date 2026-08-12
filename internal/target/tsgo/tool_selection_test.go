package tsgo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestResolveToolUsesSelectedGoAndProducesRelocatableIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is Unix-only")
	}
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	logPath := filepath.Join(directory, "selected-go.log")
	t.Setenv("GOTOTS_TEST_GO_LOG", logPath)
	wrapper := filepath.Join(directory, "selected-go")
	source := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$GOTOTS_TEST_GO_LOG\"\nexec " +
		tsgoShellQuote(goPath) + " \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(source), 0o755); err != nil {
		t.Fatal(err)
	}
	selectedGo, err := toolchain.ResolveGo(wrapper, toolSelectionCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	defaultTool, err := ResolveTool(selectedGo, repositoryRoot(), "")
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), filepath.Base(defaultTool.Path()))
	if err := os.Symlink(defaultTool.Path(), link); err != nil {
		t.Fatal(err)
	}
	explicitTool, err := ResolveTool(selectedGo, repositoryRoot(), link)
	if err != nil {
		t.Fatal(err)
	}
	if defaultTool.Identity() != explicitTool.Identity() {
		t.Fatalf("relocation changed TS-Go identity: %#v != %#v", defaultTool.Identity(), explicitTool.Identity())
	}
	identity := defaultTool.Identity().String()
	if strings.Contains(identity, defaultTool.Path()) || !defaultTool.Valid() {
		t.Fatalf("TS-Go semantic identity is invalid: %q", identity)
	}
	invocations, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(invocations), "tool -n tsgo") {
		t.Fatalf("default TS-Go did not use selected Go executable:\n%s", invocations)
	}
}

func TestResolveToolRejectsForeignExplicitExecutable(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := toolchain.ResolveGo(goPath, toolSelectionCache(t))
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveTool(selectedGo, repositoryRoot(), foreign); err == nil {
		t.Fatal("foreign explicit TS-Go executable was accepted")
	}
}

func TestPinnedToolAcceptsOnlyCertifiedModuleOrSourceBuild(t *testing.T) {
	module := &debug.BuildInfo{
		GoVersion: "go1.26.4",
		Path:      pinnedToolPackage,
		Main: debug.Module{
			Path: pinnedToolModule, Version: pinnedToolVersion, Sum: pinnedToolSum,
		},
	}
	classified, err := classifyPinnedBuild("module-tsgo", module)
	if err != nil || classified.form != ToolFormModule {
		t.Fatalf("pinned module build = %#v, %v", classified, err)
	}
	source := &debug.BuildInfo{
		GoVersion: "go1.26.4",
		Path:      pinnedToolPackage,
		Main:      debug.Module{Path: pinnedToolModule, Version: "(devel)"},
		Settings: []debug.BuildSetting{
			{Key: "vcs", Value: "git"},
			{Key: "vcs.revision", Value: pinnedSchemaRevision},
			{Key: "vcs.modified", Value: "false"},
		},
	}
	classified, err = classifyPinnedBuild("source-tsgo", source)
	if err != nil || classified.form != ToolFormSource {
		t.Fatalf("clean source build = %#v, %v", classified, err)
	}
	for name, mutate := range map[string]func(*debug.BuildInfo){
		"wrong module sum": func(info *debug.BuildInfo) {
			info.Main.Version = pinnedToolVersion
			info.Main.Sum = "h1:wrong"
		},
		"wrong revision": func(info *debug.BuildInfo) {
			info.Settings[1].Value = strings.Repeat("0", 40)
		},
		"modified": func(info *debug.BuildInfo) {
			info.Settings[2].Value = "true"
		},
		"missing vcs": func(info *debug.BuildInfo) {
			info.Settings = info.Settings[1:]
		},
		"replacement": func(info *debug.BuildInfo) {
			info.Main.Replace = &debug.Module{Path: pinnedToolModule, Version: "v0.0.0"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := *source
			candidate.Settings = append([]debug.BuildSetting(nil), source.Settings...)
			mutate(&candidate)
			if _, err := classifyPinnedBuild("source-tsgo", &candidate); err == nil {
				t.Fatal("uncertified development build was accepted")
			}
		})
	}
}

func TestResolvedToolFailsClosedAfterSealedExecutableDrift(t *testing.T) {
	selectedGo, err := toolchain.ResolveGo("", toolSelectionCache(t))
	if err != nil {
		t.Fatal(err)
	}
	original, err := ResolveTool(selectedGo, repositoryRoot(), "")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(original.Path())
	if err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(t.TempDir(), "tsgo-copy")
	payload = append(payload, 0)
	if err := os.WriteFile(copyPath, payload, 0o755); err != nil {
		t.Fatal(err)
	}
	selected, err := ResolveTool(selectedGo, repositoryRoot(), copyPath)
	if err != nil {
		t.Fatal(err)
	}
	sealedPath := selected.executable.SealedPath()
	t.Cleanup(func() {
		if err := os.WriteFile(sealedPath, payload, 0o555); err != nil {
			t.Errorf("restore sealed TS-Go fixture: %v", err)
		}
		if err := os.Chmod(sealedPath, 0o555); err != nil {
			t.Errorf("restore sealed TS-Go fixture mode: %v", err)
		}
	})
	if err := os.Chmod(sealedPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sealedPath, append(payload, 1), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := selected.command(context.Background(), "--version"); err == nil {
		t.Fatal("mutated TS-Go executable was accepted")
	}
}

func selectedTool(t *testing.T) Tool {
	t.Helper()
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := toolchain.ResolveGo(goPath, toolSelectionCache(t))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := ResolveTool(selectedGo, repositoryRoot(), "")
	if err != nil {
		t.Fatal(err)
	}
	return selected
}

func tsgoShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func toolSelectionCache(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(repositoryRoot())
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, ".temp", "cache", "toolchain-tests")
}
