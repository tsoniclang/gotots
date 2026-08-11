package verify

import (
	"context"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/tools/go/packages"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestSelectedToolOperationsDoNotReinspectCompleteGoRoot(t *testing.T) {
	repository := repositoryRoot(t)
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	selectedGo, err := toolchain.ResolveGo(
		goPath,
		filepath.Join(repository, ".temp", "cache", "toolchain-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		selectedGo.Version(),
		selectedGo.DefaultGOOS(),
		selectedGo.DefaultGOARCH(),
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":   "module example.com/exact\n\ngo 1.26.4\n",
		"main.go":  "package main\nfunc main() {}\n",
		"input.ts": "export const answer: number = 42;\n",
	} {
		if err := os.WriteFile(filepath.Join(project, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before := selectedGo.FullRootInspectionCount()
	if _, err := selectedGo.HostOutput(context.Background(), project, "version"); err != nil {
		t.Fatal(err)
	}
	if _, err := load.GoPackages(
		selectedGo,
		profile,
		load.PackageRequest{
			Context: context.Background(), Directory: project,
			FileSet: token.NewFileSet(), Mode: packages.NeedName | packages.NeedFiles,
		},
		"./...",
	); err != nil {
		t.Fatal(err)
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.CompileWithTool(
		ctx,
		selectedTSGo,
		project,
		[]string{"--noEmit", "--strict", filepath.Join(project, "input.ts")},
	); err != nil {
		t.Fatal(err)
	}
	if after := selectedGo.FullRootInspectionCount(); after != before {
		t.Fatalf(
			"Go, packages-driver, and TS-Go operations added full-root walks: before=%d after=%d",
			before,
			after,
		)
	}
	if err := selectedGo.VerifyComplete(); err != nil {
		t.Fatal(err)
	}
	if after := selectedGo.FullRootInspectionCount(); after != before+1 {
		t.Fatalf("final boundary added %d full-root walks, want 1", after-before)
	}
}
