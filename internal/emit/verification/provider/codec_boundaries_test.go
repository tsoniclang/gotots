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

func TestEncodingBinaryProfilePreservesReaderAndErrorABI(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/binaryprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package binaryprofile

import (
	"encoding/binary"
	"sync"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "read failed"
}

type blockingReader struct { mutex *sync.Mutex }

func (reader *blockingReader) Read([]byte) (int, error) {
	reader.mutex.Lock()
	reader.mutex.Unlock()
	return 0, &blockingFailure{mutex: reader.mutex}
}

func Decode(mutex *sync.Mutex) error {
	var value uint16
	return binary.Read(
		&blockingReader{mutex: mutex},
		binary.LittleEndian,
		&value,
	)
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
	scope := program.Roots()[0].Types().Scope()
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	// encoding/binary.Read is implemented over the reflection value model:
	// the compilation passes the used-provider closure and preserves the
	// certified reader and error profile ABI.
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(t, scope.Lookup("Decode"))},
		options,
	)
	if err != nil {
		t.Fatalf("implemented encoding/binary.Read failed the closure gate: %v", err)
	}
	if len(emission.Files()) == 0 {
		t.Fatal("binary profile compilation emitted no target files")
	}
}

func TestEncodingBinaryProfileCombinesOrderStreamAndErrorABI(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/binarycombinedprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package binarycombinedprofile

import (
	"encoding/binary"
	"sync"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "binary failed"
}

type blockingOrder struct { mutex *sync.Mutex }

func (order *blockingOrder) PutUint16([]byte, uint16) {}
func (order *blockingOrder) PutUint32([]byte, uint32) {}
func (order *blockingOrder) PutUint64([]byte, uint64) {}
func (order *blockingOrder) Uint16([]byte) uint16 { return 0 }
func (order *blockingOrder) Uint32([]byte) uint32 { return 0 }
func (order *blockingOrder) Uint64([]byte) uint64 { return 0 }
func (order *blockingOrder) String() string {
	order.mutex.Lock()
	order.mutex.Unlock()
	return "blocking"
}

type blockingReader struct { mutex *sync.Mutex }

func (reader *blockingReader) Read([]byte) (int, error) {
	reader.mutex.Lock()
	reader.mutex.Unlock()
	return 0, &blockingFailure{mutex: reader.mutex}
}

type blockingWriter struct { mutex *sync.Mutex }

func (writer *blockingWriter) Write(buffer []byte) (int, error) {
	writer.mutex.Lock()
	writer.mutex.Unlock()
	return len(buffer), &blockingFailure{mutex: writer.mutex}
}

func Decode(mutex *sync.Mutex) error {
	var value uint16
	return binary.Read(
		&blockingReader{mutex: mutex},
		&blockingOrder{mutex: mutex},
		&value,
	)
}

func Encode(mutex *sync.Mutex) error {
	value := uint16(1)
	return binary.Write(
		&blockingWriter{mutex: mutex},
		&blockingOrder{mutex: mutex},
		&value,
	)
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
	scope := program.Roots()[0].Types().Scope()
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	// Both callables are implemented over the reflection value model: the
	// combined order, stream, and error profile ABI compiles through the
	// used-provider closure.
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(t, scope.Lookup("Decode")),
			mustProviderRoot(t, scope.Lookup("Encode")),
		},
		options,
	)
	if err != nil {
		t.Fatalf("implemented encoding/binary family failed the closure gate: %v", err)
	}
	if len(emission.Files()) == 0 {
		t.Fatal("combined binary profile compilation emitted no target files")
	}
}

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
