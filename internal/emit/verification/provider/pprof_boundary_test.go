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

func TestRuntimePprofRetainsCanonicalWriterAcrossCalls(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/pprofeffects\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package pprofeffects

import (
	"runtime/pprof"
	"sync"
)

type blockingError struct { mutex *sync.Mutex }

func (failure *blockingError) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "write failed"
}

type blockingWriter struct {
	mutex *sync.Mutex
	bytes int
}

func (writer *blockingWriter) Write(source []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	if len(source) == 0 {
		return 0, &blockingError{mutex: writer.mutex}
	}
	writer.bytes += len(source)
	return len(source), nil
}

func Result() (bool, bool, bool, bool) {
	writer := &blockingWriter{mutex: new(sync.Mutex)}
	first := pprof.StartCPUProfile(writer)
	duplicate := pprof.StartCPUProfile(writer)
	pprof.StopCPUProfile()
	cpuWritten := writer.bytes > 0
	profile := pprof.Lookup("heap")
	if profile == nil {
		return first == nil, duplicate != nil, cpuWritten, false
	}
	before := writer.bytes
	failure := profile.WriteTo(writer, 0)
	return first == nil,
		duplicate != nil,
		cpuWritten,
		failure == nil && writer.bytes > before
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
		[]emit.Root{mustProviderRoot(t, scope.Lookup("Result"))},
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
			file.PackageName() == "pprofeffects" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("pprof fixture package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		`console.log((await Result()).map(String).join(" "));
`,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	fixture "example.com/pprofeffects"
)

func main() {
	a, b, c, d := fixture.Result()
	fmt.Println(a, b, c, d)
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
		t.Fatalf("execute Go pprof comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"runtime/pprof differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	for _, required := range []string{
		"PprofStartCPUProfileCanonicalAsyncWriterAsyncError",
		"PprofProfileWriteToCanonicalAsyncWriterAsyncError",
		"await pprof__from_gostdlib.StopCPUProfile()",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("runtime/pprof output lacks %q:\n%s", required, artifacts.printed)
		}
	}
}
