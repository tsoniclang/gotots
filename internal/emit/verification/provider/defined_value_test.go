package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestLinkedProviderDefinedValuesUseCertifiedRepresentation(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providerdefined\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providerdefined

import (
	"context"
	"io/fs"
	"iter"
	"time"
)

func Cancel(cancel context.CancelFunc) {
	cancel()
}

func Walk(
	callback fs.WalkDirFunc,
	path string,
	entry fs.DirEntry,
	failure error,
) error {
	return callback(path, entry, failure)
}

func DurationMath(value time.Duration) time.Duration {
	value += time.Second
	if value > time.Minute {
		return -value
	}
	return value
}

func SequenceTotal(sequence iter.Seq[int]) int {
	total := 0
	for value := range sequence {
		total += value
	}
	return total
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	printed := artifacts.printed
	for _, required := range []string{
		"TimeDurationValueOperations.$project",
		"TimeDurationValueOperations.$wrap",
		"IterSeqValueOperations.$project",
		"context__from_gostdlib.CancelFunc",
		"fs__from_gostdlib.WalkDirFunc",
		"const __gotots_callee_0 = cancel;",
		"const __gotots_callee_1 = callback;",
		"await __gotots_callee_0();",
		"return await __gotots_callee_1(",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("provider defined-value artifact lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		"cancel.$value",
		"callback.$value",
		"sequence.$value",
		"value.$value",
		"new Duration(",
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("provider defined-value artifact contains %q:\n%s", forbidden, printed)
		}
	}
	if count := strings.Count(
		printed,
		"IterSeqValueOperations.$project",
	); count != 1 {
		t.Fatalf(
			"provider iterator projection count = %d, want one:\n%s",
			count,
			printed,
		)
	}
}
