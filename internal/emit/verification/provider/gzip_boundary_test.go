package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestGzipStatefulProfilePreservesFieldsAndCooperativeReader(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/gzipprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package gzipprofile

import (
	"compress/gzip"
	"io"
	"sync"
)

type blockingReader struct {
	mutex *sync.Mutex
	data []byte
	offset int
	fail bool
}

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "source failed"
}

func (reader *blockingReader) Read(target []byte) (int, error) {
	reader.mutex.Lock()
	defer reader.mutex.Unlock()
	if reader.offset == len(reader.data) {
		if reader.fail {
			return 0, &blockingFailure{mutex: reader.mutex}
		}
		return 0, io.EOF
	}
	count := copy(target, reader.data[reader.offset:])
	reader.offset += count
	return count, nil
}

func (reader *blockingReader) ReadByte() (byte, error) {
	reader.mutex.Lock()
	defer reader.mutex.Unlock()
	if reader.offset == len(reader.data) {
		if reader.fail {
			return 0, &blockingFailure{mutex: reader.mutex}
		}
		return 0, io.EOF
	}
	result := reader.data[reader.offset]
	reader.offset++
	return result, nil
}

func Result() string {
	encoded := []byte{31,139,8,0,0,0,0,0,0,3,203,72,205,201,201,87,72,175,202,44,0,0,25,106,210,223,10,0,0,0}
	reader, failure := gzip.NewReader(&blockingReader{
		mutex: new(sync.Mutex),
		data: encoded,
	})
	if failure != nil {
		return failure.Error()
	}
	defer reader.Close()
	result := reader.Header.Name + "|" + reader.Comment + "|"
	buffer := make([]byte, 4)
	for {
		count, readFailure := reader.Read(buffer)
		result += string(buffer[:count])
		if readFailure == io.EOF {
			return result
		}
		if readFailure != nil {
			return readFailure.Error()
		}
	}
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
			file.PackageName() == "gzipprofile" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("gzip fixture package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		`console.log(JSON.stringify(await Result()));
`,
	)
	for _, required := range []string{
		"GzipNewReaderCanonical",
		"CanonicalGzipReader",
		"bindPointer<",
		".Header.Name",
		".Header.Comment",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("gzip profile output lacks %q:\n%s", required, artifacts.printed)
		}
	}
}
