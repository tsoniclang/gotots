package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDisabledGzipReaderSelectsDirectProfile(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directgzip\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directgzip

import "compress/gzip"

type reader struct {
	data []byte
}

func (source *reader) Read(target []byte) (int, error) {
	count := copy(target, source.data)
	source.data = source.data[count:]
	return count, nil
}

func Result(source []byte) error {
	reader, failure := gzip.NewReader(&reader{data: source})
	if failure != nil {
		return failure
	}
	return reader.Close()
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
	if !strings.Contains(artifacts.printed, "GzipNewReaderDirect") ||
		strings.Contains(artifacts.printed, "GzipNewReaderCanonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled gzip reader profile is not direct:\n%s", artifacts.printed)
	}
}
