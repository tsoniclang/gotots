package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestIoReadFullPreservesDirectReaderAndErrorBehavior(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/ioreadprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package ioreadprofile

import (
	"io"
	"sync"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "read failed"
}

type blockingReader struct {
	mutex *sync.Mutex
	data []byte
	offset int
	custom bool
}

func (reader *blockingReader) Read(target []byte) (int, error) {
	reader.mutex.Lock()
	defer reader.mutex.Unlock()
	if reader.offset == len(reader.data) {
		if reader.custom {
			return 0, &blockingFailure{mutex: reader.mutex}
		}
		return 0, io.EOF
	}
	count := copy(target, reader.data[reader.offset:])
	reader.offset += count
	return count, nil
}

func Result(text string, custom bool) (int, string) {
	target := make([]byte, 4)
	count, failure := io.ReadFull(&blockingReader{
		mutex: new(sync.Mutex),
		data: []byte(text),
		custom: custom,
	}, target)
	if failure != nil {
		return count, failure.Error()
	}
	return count, string(target)
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
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "ioreadprofile" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("io ReadFull fixture package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		`for (const [count, result] of [
  Result("four", false),
  Result("abc", false),
  Result("abc", true),
]) {
  console.log(count + " " + JSON.stringify(result));
}
`,
	)
	if !strings.Contains(artifacts.printed, "IoReadFullDirect") {
		t.Fatalf("io.ReadFull output lacks direct boundary:\n%s", artifacts.printed)
	}
	if strings.Contains(artifacts.printed, "IoReadFullCanonicalSync") ||
		strings.Contains(artifacts.printed, "IoReadFullCanonicalAsync") {
		t.Fatalf("io.ReadFull output retained a profile variant:\n%s", artifacts.printed)
	}
	for _, forbidden := range []string{"async ", "await ", "Promise<", "Awaitable<"} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("io.ReadFull output contains %q", forbidden)
		}
	}
}
