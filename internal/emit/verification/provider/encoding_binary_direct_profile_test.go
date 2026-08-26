package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

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
