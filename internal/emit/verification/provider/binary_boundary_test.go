package provider_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

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
