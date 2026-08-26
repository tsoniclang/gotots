package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDisabledRuntimePprofCallablesSelectDirectProfiles(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directpprof\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directpprof

import "runtime/pprof"

type writer struct{}

func (*writer) Write(source []byte) (int, error) { return len(source), nil }

func Result() error {
	target := &writer{}
	if failure := pprof.StartCPUProfile(target); failure != nil {
		return failure
	}
	pprof.StopCPUProfile()
	profile := pprof.Lookup("goroutine")
	if profile == nil {
		return nil
	}
	return profile.WriteTo(target, 0)
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
		"PprofStartCPUProfileDirect",
		"PprofProfileWriteToDirect",
	} {
		if !strings.Contains(artifacts.printed, expected) {
			t.Fatalf("disabled runtime/pprof profile omits %s:\n%s", expected, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "Canonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled runtime/pprof profiles are not direct:\n%s", artifacts.printed)
	}
}
