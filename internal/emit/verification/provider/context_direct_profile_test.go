package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDisabledContextCallablesSelectDirectProfiles(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directcontext\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directcontext

import (
	"context"
	"time"
)

func Result() bool {
	valued := context.WithValue(context.Background(), "key", "value")
	canceled, cancel := context.WithCancel(valued)
	cancel()
	caused, cancelCause := context.WithCancelCause(canceled)
	cancelCause(nil)
	timed, cancelTimeout := context.WithTimeout(caused, time.Duration(1))
	cancelTimeout()
	stop := context.AfterFunc(timed, func() {})
	_ = stop()
	return context.Cause(timed) != nil
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
	options := emit.DefaultOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Result"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	for _, expected := range []string{
		"ContextWithValueDirect",
		"ContextWithCancelDirect",
		"ContextWithCancelCauseDirect",
		"ContextWithTimeoutDirect",
		"ContextAfterFuncDirect",
		"ContextCauseDirect",
	} {
		if !strings.Contains(artifacts.printed, expected) {
			t.Fatalf("disabled context profile omits %s:\n%s", expected, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "Canonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled context profiles are not direct:\n%s", artifacts.printed)
	}
}
