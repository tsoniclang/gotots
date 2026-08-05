package provider_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"ReadGenerated", "ReadProvider"},
		`const [fallback, fast, generatedFailure] = await ReadGenerated();
console.log(fallback + "|" + fast + "|" + (generatedFailure === undefined));
const [raw, rawFailure] = await ReadProvider(`+strconv.Quote(project)+`);
console.log(raw + "|" + (rawFailure === undefined));
`,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	capability "example.com/iofscapability"
)

func main() {
	fallback, fast, generatedFailure := capability.ReadGenerated()
	fmt.Printf("%s|%s|%t\n", fallback, fast, generatedFailure == nil)
	raw, rawFailure := capability.ReadProvider(`+strconv.Quote(project)+`)
	fmt.Printf("%s|%t\n", raw, rawFailure == nil)
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
		t.Fatalf("execute Go io/fs capability comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"io/fs capability differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	for _, required := range []string{
		"IoFsReadFileCanonical",
		"$Capability$",
		"$raw",
		"instanceof",
		"$go$generated",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("io/fs capability output lacks %q:\n%s", required, artifacts.printed)
		}
	}
}
