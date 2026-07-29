package emit_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveNineRequiresExplicitCooperativeProfile(t *testing.T) {
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/concurrencyprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(directory, "source.go"), `package concurrencyprofile

func Run() int32 {
	values := make(chan int32, 1)
	values <- 1
	return <-values
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Run"),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(program, []emit.Root{root})
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.CallExpr" ||
		unsupported.Role != api.RoleLocalValue {
		t.Fatalf(
			"default concurrency error = %#v, want make-expression UnsupportedError",
			err,
		)
	}
	if _, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		waveNineOptions(),
	); err != nil {
		t.Fatalf("explicit cooperative profile: %v", err)
	}
}

func waveNineOptions() emit.Options {
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	return options
}

func TestConcurrencySemanticsSelectionIsClosed(t *testing.T) {
	if emit.DefaultOptions().ConcurrencySemantics !=
		emit.ConcurrencySemanticsDisabled {
		t.Fatal("default options silently select cooperative concurrency")
	}
	for source, want := range map[string]emit.ConcurrencySemantics{
		"disabled":    emit.ConcurrencySemanticsDisabled,
		"cooperative": emit.ConcurrencySemanticsCooperative,
	} {
		got, err := emit.ParseConcurrencySemantics(source)
		if err != nil || got != want {
			t.Fatalf("parse %q = %s, %v; want %s", source, got, err, want)
		}
	}
	got, err := emit.ParseConcurrencySemantics("preemptive")
	if err == nil || got != emit.ConcurrencySemanticsInvalid {
		t.Fatalf("parse preemptive = %s, %v; want typed rejection", got, err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsInvalid
	_, err = emit.CompileWithOptions(nil, nil, options)
	var optionsError *emit.OptionsError
	if !errors.As(err, &optionsError) ||
		optionsError.Field != "concurrency semantics" {
		t.Fatalf("invalid concurrency option error = %#v", err)
	}
}

func TestWaveNinePreemptionBoundaryAddsNoYieldHeuristic(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("RequiresPreemption"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	target := waveNineFunctionText(t, artifacts.printed, "RequiresPreemption")
	if !strings.Contains(target, "GoScheduler.spawn") ||
		!strings.Contains(target, "for (;;)") {
		t.Fatalf("preemption boundary shape changed:\n%s", target)
	}
	for _, forbidden := range []string{
		"setTimeout",
		"queueMicrotask",
		"setImmediate",
		"yield",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf(
				"preemption boundary acquired %q heuristic:\n%s",
				forbidden,
				target,
			)
		}
	}
}
