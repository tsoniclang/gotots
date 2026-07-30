package emit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestEnvironmentGenericCallableProfileIsTypedAndBodyless(
	t *testing.T,
) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/environmentgeneric\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package environmentgeneric

import "slices"

func Sum(values []int32, input <-chan int32) int32 {
	var total int32
	for value := range slices.Values(values) {
		total += value + <-input
	}
	return total
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Sum"),
	)
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	baseCount := 0
	profileCount := 0
	profileName := ""
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileEnvironmentContract {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name() == nil {
				continue
			}
			name := function.Name().Text()
			switch {
			case name == "Values":
				baseCount++
			case strings.HasPrefix(name, "Values$cooperative_"):
				profileCount++
				profileName = name
				if function.Body() != nil {
					t.Fatal("environment callable profile acquired a body")
				}
			}
		}
	}
	if baseCount != 1 || profileCount != 1 {
		t.Fatalf(
			"environment Values declarations base=%d profile=%d, want 1/1:\n%s",
			baseCount,
			profileCount,
			artifacts.printed,
		)
	}
	if !strings.Contains(artifacts.printed, profileName) {
		t.Fatalf(
			"environment callable profile is absent:\n%s",
			artifacts.printed,
		)
	}
	if !strings.Contains(artifacts.printed, "Promise<bool>") {
		t.Fatalf(
			"environment profile declaration lacks cooperative yield ABI:\n%s",
			artifacts.printed,
		)
	}
	base := environmentDeclarationLine(
		t,
		artifacts.printed,
		"export declare function Values<",
	)
	if strings.Contains(base, "Promise<") {
		t.Fatalf(
			"environment base declaration was widened:\n%s",
			base,
		)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		append(
			[]string{
				"--target", "es2022",
				"--module", "nodenext",
				"--moduleResolution", "nodenext",
				"--strict",
				"--noEmit",
			},
			artifacts.paths...,
		),
	); err != nil {
		t.Fatal(err)
	}
}

func environmentDeclarationLine(
	t *testing.T,
	printed string,
	prefix string,
) string {
	t.Helper()
	start := strings.Index(printed, prefix)
	if start < 0 {
		t.Fatalf("environment declaration lacks %q:\n%s", prefix, printed)
	}
	end := strings.IndexByte(printed[start:], '\n')
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+end]
}
