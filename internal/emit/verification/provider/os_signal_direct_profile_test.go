package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDisabledNotifyContextSelectsDirectProfile(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directsignalcontext\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directsignalcontext

import (
	"context"
	"os/signal"
)

func Result() bool {
	child, stop := signal.NotifyContext(context.Background())
	stop()
	return child != nil
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
	if !strings.Contains(artifacts.printed, "OsSignalNotifyContextDirect") ||
		strings.Contains(artifacts.printed, "OsSignalNotifyContextCanonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled signal context profile is not direct:\n%s", artifacts.printed)
	}
}
