package provider_test

import (
	"context"
	"path/filepath"
	"strings"
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
	// encoding/binary.Read is a certified placeholder until the reflection
	// family lands; the used-provider closure must fail this compilation
	// before any target file is sealed. The profile-ABI artifact
	// assertions return with the implemented family.
	_, err = emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(t, scope.Lookup("Decode"))},
		options,
	)
	if err == nil {
		t.Fatal("used encoding/binary.Read placeholder passed the closure gate")
	}
	if !strings.Contains(err.Error(), "used provider placeholders") ||
		!strings.Contains(
			err.Error(),
			"encoding/binary|kind=4|receiver=|name=Read",
		) {
		t.Fatalf("closure diagnostic = %v", err)
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
	// Both selected placeholders must be named by one typed closure
	// diagnostic with identity-keyed lists; the behavior assertions return
	// with the implemented family.
	_, err = emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(t, scope.Lookup("Decode")),
			mustProviderRoot(t, scope.Lookup("Encode")),
		},
		options,
	)
	if err == nil {
		t.Fatal("used encoding/binary placeholders passed the closure gate")
	}
	for _, expected := range []string{
		"encoding/binary|kind=4|receiver=|name=Read",
		"encoding/binary|kind=4|receiver=|name=Write",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("closure diagnostic lacks %q: %v", expected, err)
		}
	}
}
