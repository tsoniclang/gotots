package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDisabledFmtWriterCallablesSelectDirectProfiles(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directfmtwriter\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directfmtwriter

import "fmt"

type writer struct{}

func (*writer) Write(source []byte) (int, error) { return len(source), nil }

func Result() (int, error) {
	target := &writer{}
	first, failure := fmt.Fprint(target, "first")
	if failure != nil {
		return 0, failure
	}
	second, failure := fmt.Fprintf(target, "%s", "second")
	if failure != nil {
		return 0, failure
	}
	third, failure := fmt.Fprintln(target, "third")
	return first + second + third, failure
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
		"FprintDirect",
		"FprintfDirect",
		"FprintlnDirect",
	} {
		if !strings.Contains(artifacts.printed, expected) {
			t.Fatalf("disabled fmt writer profile omits %s:\n%s", expected, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "Canonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled fmt writer profiles are not direct:\n%s", artifacts.printed)
	}
}
