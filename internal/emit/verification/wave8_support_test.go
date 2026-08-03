package emit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestFallthroughReturnHasExactUnreachableEndGuard(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	guarded, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("FallthroughReturn"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ordinary, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("FallthroughControl"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{guarded, ordinary})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	target := targetFunctionText(t, artifacts.printed, "FallthroughReturn")
	if strings.Count(target, "unreachable Go function end") != 1 ||
		!strings.Contains(
			target,
			`throw new globalThis.Error("unreachable Go function end");`,
		) {
		t.Fatalf("result guard is not exact:\n%s", target)
	}
	ordinaryTarget := targetFunctionText(
		t,
		artifacts.printed,
		"FallthroughControl",
	)
	if strings.Contains(ordinaryTarget, "unreachable Go function end") {
		t.Fatalf("direct result gained an unreachable guard:\n%s", ordinaryTarget)
	}
}

func waveEightControlDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"control",
		"wave8",
	)
}
