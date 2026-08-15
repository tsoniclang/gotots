package provider_test

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestIoFsReadFileCapabilityPreservesGeneratedAndProviderPaths(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/iofscapability\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "raw.txt"), "raw-provider")
	writeProgramFile(t, filepath.Join(project, "source.go"), `package iofscapability

import (
	"io"
	"io/fs"
	"os"
)

type memoryFile struct {
	data   []byte
	offset int
}

func (file *memoryFile) Read(target []byte) (int, error) {
	if file.offset == len(file.data) {
		return 0, io.EOF
	}
	count := copy(target, file.data[file.offset:])
	file.offset += count
	if file.offset == len(file.data) {
		return count, io.EOF
	}
	return count, nil
}

func (*memoryFile) Close() error { return nil }
func (*memoryFile) Stat() (fs.FileInfo, error) { return nil, io.ErrUnexpectedEOF }

type fallbackFS struct{}

func (*fallbackFS) Open(string) (fs.File, error) {
	return &memoryFile{data: []byte("generated-fallback")}, nil
}

type fastFS struct{}

func (*fastFS) Open(string) (fs.File, error) {
	panic("optional ReadFile capability was not selected")
}

func (*fastFS) ReadFile(name string) ([]byte, error) {
	return []byte("generated-fast:" + name), nil
}

func ReadGenerated() (string, string, error) {
	fallback, failure := fs.ReadFile(&fallbackFS{}, "input")
	if failure != nil {
		return "", "", failure
	}
	fast, failure := fs.ReadFile(&fastFS{}, "input")
	return string(fallback), string(fast), failure
}

func ReadProvider(root string) (string, error) {
	value, failure := fs.ReadFile(os.DirFS(root), "raw.txt")
	return string(value), failure
}

func ReadProviderDir(root string) (string, int64, error) {
	entries, failure := fs.ReadDir(os.DirFS(root), ".")
	if failure != nil {
		return "", 0, failure
	}
	for _, entry := range entries {
		if entry.Name() != "raw.txt" {
			continue
		}
		info, failure := entry.Info()
		if failure != nil {
			return "", 0, failure
		}
		return entry.Name(), info.Size(), nil
	}
	return "", 0, fs.ErrNotExist
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
			mustProviderRoot(t, scope.Lookup("ReadGenerated")),
			mustProviderRoot(t, scope.Lookup("ReadProvider")),
			mustProviderRoot(t, scope.Lookup("ReadProviderDir")),
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
			file.PackageName() == "iofscapability" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("io/fs capability package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReadGenerated", "ReadProvider", "ReadProviderDir"},
		`const [fallback, fast, generatedFailure] = await ReadGenerated();
console.log(fallback + "|" + fast + "|" + (generatedFailure === undefined));
const [raw, rawFailure] = await ReadProvider(`+strconv.Quote(project)+`);
console.log(raw + "|" + (rawFailure === undefined));
const [name, size, dirFailure] = await ReadProviderDir(`+strconv.Quote(project)+`);
console.log(name + "|" + size + "|" + (dirFailure === undefined));
`,
	)
	for _, required := range []string{
		"IoFsReadFileCanonical",
		"IoFsReadDirCanonical",
		"RuntimeSliceProjection",
		"$Capability$",
		"$raw",
		"instanceof",
		"$go$generated",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("io/fs capability output lacks %q:\n%s", required, artifacts.printed)
		}
	}
	assertIoFsProfileCapabilityHeritage(t, artifacts.printed)
}

func assertIoFsProfileCapabilityHeritage(t *testing.T, printed string) {
	t.Helper()
	var classHeaders []string
	for _, line := range strings.Split(printed, "\n") {
		if strings.Contains(line, "class ") &&
			strings.Contains(line, " implements ") {
			classHeaders = append(classHeaders, line)
		}
	}
	for _, contract := range []string{
		"Method_fs$ReadFile_string_to_SliceOf_byte_Named_error",
		"Method_fs$ReadDir_string_to_SliceOf_Named_fs$DirEntry_Named_error",
	} {
		found := false
		for _, line := range strings.Split(printed, "\n") {
			if strings.Contains(line, "class ") &&
				strings.Contains(line, " implements ") &&
				strings.Contains(line, contract) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf(
				"io/fs profile output has no class with exact capability heritage %q:\n%s",
				contract,
				strings.Join(classHeaders, "\n"),
			)
		}
	}
}
