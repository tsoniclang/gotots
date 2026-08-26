package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDisabledBase64EncoderSelectsDirectProfile(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directbase64\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directbase64

import "encoding/base64"

type writer struct{}

func (*writer) Write(source []byte) (int, error) { return len(source), nil }

func Result() error {
	encoder := base64.NewEncoder(base64.StdEncoding, &writer{})
	if _, failure := encoder.Write([]byte("source")); failure != nil {
		return failure
	}
	return encoder.Close()
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
	if !strings.Contains(artifacts.printed, "Base64NewEncoderDirect") ||
		strings.Contains(artifacts.printed, "Base64NewEncoderCanonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled base64 encoder profile is not direct:\n%s", artifacts.printed)
	}
}

func TestDisabledEncodingBinaryCallablesSelectDirectProfiles(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directbinary\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directbinary

import "encoding/binary"

type buffer struct {
	data []byte
}

func (target *buffer) Write(source []byte) (int, error) {
	target.data = append(target.data, source...)
	return len(source), nil
}

func (target *buffer) Read(destination []byte) (int, error) {
	return copy(destination, target.data), nil
}

func Result(value *uint32) error {
	target := &buffer{}
	if failure := binary.Write(target, binary.LittleEndian, *value); failure != nil {
		return failure
	}
	return binary.Read(target, binary.LittleEndian, value)
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
		"EncodingBinaryReadDirect",
		"EncodingBinaryWriteDirect",
	} {
		if !strings.Contains(artifacts.printed, expected) {
			t.Fatalf("disabled encoding/binary profile omits %s:\n%s", expected, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "Canonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled encoding/binary profiles are not direct:\n%s", artifacts.printed)
	}
}

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
