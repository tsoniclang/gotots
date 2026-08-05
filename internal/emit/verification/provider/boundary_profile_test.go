package provider_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestStatefulProviderProfilePreservesRetainedInterfaceABI(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/statefulproviderprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package statefulproviderprofile

import (
	"bufio"
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
}

func (reader *blockingReader) Read(target []byte) (int, error) {
	if reader.offset == len(reader.data) {
		return 0, &blockingFailure{mutex: reader.mutex}
	}
	count := copy(target, reader.data[reader.offset:])
	reader.offset += count
	return count, nil
}

type holder struct { reader *bufio.Reader }

func CompositeOnly(source io.Reader) *bufio.Reader {
	return (&holder{reader: bufio.NewReader(source)}).reader
}

func NewBuffered(mutex *sync.Mutex, text string) *bufio.Reader {
	return bufio.NewReader(&blockingReader{mutex: mutex, data: []byte(text)})
}

func ReadLine(mutex *sync.Mutex, text string) (string, error) {
	selected := holder{reader: NewBuffered(mutex, text)}
	var asReader io.Reader = selected.reader
	_ = asReader
	line, failure := selected.reader.ReadBytes('\n')
	return string(line), failure
}

func Run(text string) (string, string) {
	line, failure := ReadLine(&sync.Mutex{}, text)
	if failure == nil {
		return line, ""
	}
	return line, failure.Error()
}

type stalledReader struct{}

func (*stalledReader) Read([]byte) (int, error) { return 0, nil }

func NoProgress() bool {
	_, failure := bufio.NewReader(&stalledReader{}).ReadByte()
	return failure == io.ErrNoProgress
}

func NilConstructed() bool { return bufio.NewReader(nil) != nil }
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
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(t, scope.Lookup("CompositeOnly")),
			mustProviderRoot(t, scope.Lookup("Run")),
			mustProviderRoot(t, scope.Lookup("NoProgress")),
			mustProviderRoot(t, scope.Lookup("NilConstructed")),
		},
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
			file.PackageName() == "statefulproviderprofile" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("stateful provider package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"NilConstructed", "NoProgress", "Run"},
		`for (const input of ["alpha\nrest", "tail"]) {
  const [line, failure] = await Run(input);
  console.log(JSON.stringify(line) + " " + JSON.stringify(failure));
}
console.log(await NoProgress());
console.log(await NilConstructed());
`,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	stateful "example.com/statefulproviderprofile"
)

func main() {
	for _, input := range []string{"alpha\nrest", "tail"} {
		line, failure := stateful.Run(input)
		fmt.Printf("%q %q\n", line, failure)
	}
	fmt.Println(stateful.NoProgress())
	fmt.Println(stateful.NilConstructed())
}
`)
	sourceContext, sourceCancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer sourceCancel()
	command := exec.CommandContext(sourceContext, "go", "run", ".")
	command.Dir = runnerDirectory
	sourceOutput, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("execute Go provider comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"stateful provider differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	for _, required := range []string{
		"CanonicalBufioReader",
		"await",
		"ReadBytes",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("stateful profile output lacks %q:\n%s", required, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "CanonicalReaderSync") ||
		strings.Contains(artifacts.printed, "CanonicalReaderAsync") {
		t.Fatalf("cooperative profile retained a reader permutation:\n%s", artifacts.printed)
	}
	if strings.Contains(artifacts.printed, "BufioReaderRead") {
		t.Fatal("stateful reader adapter retained the ordinary recovery target")
	}
}

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

type blockingFileError struct { mutex *sync.Mutex }

func (failure *blockingFileError) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "read failed"
}

func (file *blockingFile) Close() error {
	file.mutex.Lock()
	file.mutex.Unlock()
	return nil
}

func (file *blockingFile) Read([]byte) (int, error) {
	return 0, &blockingFileError{mutex: file.mutex}
}
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

func Metadata(fileSystem fs.FS) (fs.FileInfo, error) {
	return fs.Stat(fileSystem, "input")
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
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(t, scope.Lookup("Read")),
			mustProviderRoot(t, scope.Lookup("Metadata")),
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
	if !strings.Contains(
		artifacts.printed,
		"strings.Reader.Read(this.$go$value, $argument0)",
	) || !strings.Contains(artifacts.printed, ".$from(__gotots_results_") {
		t.Fatalf(
			"provider method adapter did not preserve the public call and bridge its result:\n%s",
			artifacts.printed,
		)
	}
	if strings.Contains(artifacts.printed, "strings.Reader.Read$deferred") {
		t.Fatalf(
			"private recovery mechanics leaked into the provider method surface:\n%s",
			artifacts.printed,
		)
	}
	if !strings.Contains(artifacts.printed, "IoFsReadFileCanonical") {
		t.Fatalf("canonical fs.ReadFile boundary is absent:\n%s", artifacts.printed)
	}
	if !strings.Contains(artifacts.printed, "IoFsStatCanonical") {
		t.Fatalf("canonical fs.Stat boundary is absent:\n%s", artifacts.printed)
	}
}

func TestProviderCallableProfileWithoutInterfaceContract(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/callableprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package callableprofile

import (
	"strings"
	"sync"
)

func Transform(input string) string {
	var mutex sync.Mutex
	return strings.Map(func(value rune) rune {
		mutex.Lock()
		mutex.Unlock()
		return value + 1
	}, input)
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
			program.Roots()[0].Types().Scope().Lookup("Transform"),
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
			file.PackageName() == "callableprofile" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("callable-profile package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Transform"},
		`console.log(JSON.stringify(await Transform("ab")));
`,
	)
	if targetOutput != "\"bc\"\n" {
		t.Fatalf("callable-profile output = %q, want %q", targetOutput, "\"bc\"\n")
	}
	if !strings.Contains(artifacts.printed, "StringsMapCanonical") ||
		!strings.Contains(artifacts.printed, "await") {
		t.Fatalf("callable-only provider profile was not selected:\n%s", artifacts.printed)
	}
}
