package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestBase64EncoderPreservesCanonicalWriterAndErrorBehavior(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/base64encoderprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package base64encoderprofile

import (
	"encoding/base64"
	"sync"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "write failed"
}

type blockingWriter struct {
	mutex *sync.Mutex
	failAt int
	calls []int
	bytes []byte
}

func (writer *blockingWriter) Write(source []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	writer.calls = append(writer.calls, len(source))
	if len(writer.calls) == writer.failAt {
		return 0, &blockingFailure{mutex: writer.mutex}
	}
	writer.bytes = append(writer.bytes, source...)
	return len(source), nil
}

func Result(failAt int) (int, int, int, int, int, int, int, string, string) {
	source := make([]byte, 1540)
	for index := range source {
		source[index] = byte(index)
	}
	writer := &blockingWriter{mutex: new(sync.Mutex), failAt: failAt}
	encoder := base64.NewEncoder(base64.StdEncoding, writer)
	count, writeFailure := encoder.Write(source)
	closeFailure := encoder.Close()
	lengths := [4]int{}
	copy(lengths[:], writer.calls)
	writeMessage := ""
	if writeFailure != nil {
		writeMessage = writeFailure.Error()
	}
	closeMessage := ""
	if closeFailure != nil {
		closeMessage = closeFailure.Error()
	}
	return count, len(writer.bytes), len(writer.calls),
		lengths[0], lengths[1], lengths[2], lengths[3],
		writeMessage, closeMessage
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
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
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
			file.PackageName() == "base64encoderprofile" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("base64 encoder fixture package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		`for (const failAt of [0, 2]) {
  console.log(JSON.stringify(await Result(failAt)));
}
`,
	)
	if !strings.Contains(
		artifacts.printed,
		"Base64NewEncoderCanonical<",
	) {
		t.Fatalf("base64 output lacks canonical boundary:\n%s", artifacts.printed)
	}
	for _, obsolete := range []string{
		"Base64NewEncoderCanonicalSync",
		"Base64NewEncoderCanonicalAsync",
	} {
		if strings.Contains(artifacts.printed, obsolete) {
			t.Fatalf("base64 output retained profile variant %q:\n%s", obsolete, artifacts.printed)
		}
	}
}
