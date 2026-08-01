package provider_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestProviderCallableProfilesPreserveCanonicalInterfaceABI(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providerprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providerprofile

import (
	"context"
	"encoding/binary"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"sync"
)

type blockingFile struct { mutex *sync.Mutex }

func (file *blockingFile) Close() error {
	file.mutex.Lock()
	file.mutex.Unlock()
	return nil
}

func (file *blockingFile) Read([]byte) (int, error) { return 0, nil }
func (file *blockingFile) Stat() (fs.FileInfo, error) { return nil, nil }

type blockingFS struct { mutex *sync.Mutex }

func (fileSystem *blockingFS) Open(string) (fs.File, error) {
	return &blockingFile{mutex: fileSystem.mutex}, nil
}

type blockingSignal struct { mutex *sync.Mutex }

func (value blockingSignal) Signal() {}

func (value blockingSignal) String() string {
	value.mutex.Lock()
	value.mutex.Unlock()
	return "interrupt"
}

type blockingOrder struct { mutex *sync.Mutex }

func (value blockingOrder) String() string {
	value.mutex.Lock()
	value.mutex.Unlock()
	return "blocking"
}

func (blockingOrder) Uint16([]byte) uint16 { return 0 }
func (blockingOrder) Uint32([]byte) uint32 { return 0 }
func (blockingOrder) Uint64([]byte) uint64 { return 0 }
func (blockingOrder) PutUint16([]byte, uint16) {}
func (blockingOrder) PutUint32([]byte, uint32) {}
func (blockingOrder) PutUint64([]byte, uint64) {}

func Read(fileSystem fs.FS) ([]byte, error) {
	return fs.ReadFile(fileSystem, "input")
}

func NewFileSystem(mutex *sync.Mutex) fs.FS {
	return &blockingFS{mutex: mutex}
}

func Notify(parent context.Context, value os.Signal) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, value, os.Interrupt)
}

func NewSignal(mutex *sync.Mutex) os.Signal {
	return blockingSignal{mutex: mutex}
}

func Kill(process *os.Process, value os.Signal) error {
	return process.Signal(value)
}

func Decode(order binary.ByteOrder) error {
	var value uint16
	return binary.Read(strings.NewReader("\x00\x01"), order, &value)
}

func NewOrder(mutex *sync.Mutex) binary.ByteOrder {
	return blockingOrder{mutex: mutex}
}

func RootFactory() func(string) fs.FS { return os.DirFS }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(t, scope.Lookup("Read")),
			mustProviderRoot(t, scope.Lookup("NewFileSystem")),
			mustProviderRoot(t, scope.Lookup("Notify")),
			mustProviderRoot(t, scope.Lookup("NewSignal")),
			mustProviderRoot(t, scope.Lookup("Kill")),
			mustProviderRoot(t, scope.Lookup("Decode")),
			mustProviderRoot(t, scope.Lookup("NewOrder")),
			mustProviderRoot(t, scope.Lookup("RootFactory")),
		},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
}
