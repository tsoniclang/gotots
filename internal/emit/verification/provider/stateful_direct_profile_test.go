package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDisabledStatefulProfileSelectsDirectCarrier(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directstatefulprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directstatefulprofile

import "bufio"

type sink struct { data []byte }

func (target *sink) Write(source []byte) (int, error) {
	target.data = append(target.data, source...)
	return len(source), nil
}

func Render(text string) string {
	target := &sink{}
	writer := bufio.NewWriter(target)
	_, _ = writer.Write([]byte(text))
	_ = writer.WriteByte('!')
	_ = writer.Flush()
	return string(target.data)
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
			program.Roots()[0].Types().Scope().Lookup("Render"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	if !strings.Contains(artifacts.printed, "DirectBufioWriter") ||
		!strings.Contains(artifacts.printed, "NewWriterDirect") ||
		strings.Contains(artifacts.printed, "CanonicalBufioWriter") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled stateful profile is not direct:\n%s", artifacts.printed)
	}
}
