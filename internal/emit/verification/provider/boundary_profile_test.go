package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

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
	typecheckProviderRunner(
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
	"encoding/binary"
	"io"
	"io/fs"
	"os"
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

func NewSignal(mutex *sync.Mutex) os.Signal {
	return blockingSignal{mutex: mutex}
}

func Kill(process *os.Process, value os.Signal) error {
	return process.Signal(value)
}

// Decode crosses a provider strings.Reader through the canonical io.Reader
// boundary; encoding/binary.Read itself remains a certified placeholder
// proven by its own closure-gate test until the reflection family lands.
func Decode() (int, error) {
	buffer := make([]byte, 2)
	return io.ReadFull(strings.NewReader("\x00\x01"), buffer)
}

func NewOrder(mutex *sync.Mutex) binary.ByteOrder {
	return blockingOrder{mutex: mutex}
}

func RootFactory() func(string) fs.FS { return os.DirFS }

func Visit(fileSystem fs.FS, callback fs.WalkDirFunc) error {
	return fs.WalkDir(fileSystem, ".", callback)
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
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(t, scope.Lookup("Read")),
			mustProviderRoot(t, scope.Lookup("Metadata")),
			mustProviderRoot(t, scope.Lookup("NewFileSystem")),
			mustProviderRoot(t, scope.Lookup("NewSignal")),
			mustProviderRoot(t, scope.Lookup("Kill")),
			mustProviderRoot(t, scope.Lookup("Decode")),
			mustProviderRoot(t, scope.Lookup("NewOrder")),
			mustProviderRoot(t, scope.Lookup("RootFactory")),
			mustProviderRoot(t, scope.Lookup("Visit")),
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
		"this.$go$value.Read($argument0)",
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
	if !strings.Contains(artifacts.printed, "IoFsWalkDirCanonical") ||
		strings.Contains(artifacts.printed, "fs__from_gostdlib.WalkDirFunc") {
		t.Fatalf("canonical fs.WalkDir callable boundary is absent:\n%s", artifacts.printed)
	}
	if !strings.Contains(artifacts.printed, "new RuntimeSliceProjection<") ||
		!strings.Contains(artifacts.printed, "this.$go$value.ReadDir(") {
		t.Fatalf(
			"recursive fs.DirEntry slice boundary is not projected at the provider bridge:\n%s",
			artifacts.printed,
		)
	}
}

func TestDisabledRequiredProviderProfileSelectsDirectCarrier(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directrequiredprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directrequiredprofile

import "errors"

type leaf struct{}

func (leaf) Error() string { return "leaf" }

type wrapper struct {
	cause error
}

func (wrapper) Error() string { return "wrapper" }
func (value wrapper) Unwrap() error { return value.cause }

func Cause() error {
	return errors.Unwrap(wrapper{cause: leaf{}})
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
	options.ConcurrencySemantics = emit.ConcurrencySemanticsDisabled
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Cause"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	if !strings.Contains(artifacts.printed, "ErrorsUnwrapDirect") ||
		strings.Contains(artifacts.printed, "ErrorsUnwrapCanonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled required profile is not direct:\n%s", artifacts.printed)
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
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Transform"},
		`console.log(JSON.stringify(await Transform("ab")));
`,
	)
	if !strings.Contains(artifacts.printed, "StringsMapCanonical") ||
		!strings.Contains(artifacts.printed, "await") {
		t.Fatalf("callable-only provider profile was not selected:\n%s", artifacts.printed)
	}
}

func TestProviderGenericCallableAndTupleBoundary(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/genericproviderboundary\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package genericproviderboundary

import (
	"cmp"
	"slices"
	"strings"
)

func Search(values []string, target string) (int, bool) {
	return slices.BinarySearchFunc(values, target, strings.Compare)
}

func FirstInt() int {
	return cmp.Or(0, 7, 9)
}

func FirstInt64() int64 {
	return cmp.Or(int64(0), int64(11), int64(13))
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
		[]emit.Root{
			mustProviderRoot(
				t,
				program.Roots()[0].Types().Scope().Lookup("Search"),
			),
			mustProviderRoot(
				t,
				program.Roots()[0].Types().Scope().Lookup("FirstInt"),
			),
			mustProviderRoot(
				t,
				program.Roots()[0].Types().Scope().Lookup("FirstInt64"),
			),
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
			file.PackageName() == "genericproviderboundary" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("generic-provider package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"FirstInt", "FirstInt64"},
		`console.log((await FirstInt()) + "|" + (await FirstInt64()));
`,
	)
	for _, required := range []string{
		"satisfies",
		"Number(",
		"BigInt.asIntN",
		"CmpOrKernel<int, int>(",
		"CmpOrKernel<int64, int64>(",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("generic provider boundary lacks %q:\n%s", required, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "$providerStorage") {
		t.Fatalf("generic provider boundary projected caller-owned storage:\n%s", artifacts.printed)
	}
}
