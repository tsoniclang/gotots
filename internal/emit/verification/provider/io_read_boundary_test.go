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

func TestIoReadFullPreservesCanonicalReaderAndErrorBehavior(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/ioreadprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package ioreadprofile

import (
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
	custom bool
}

func (reader *blockingReader) Read(target []byte) (int, error) {
	reader.mutex.Lock()
	defer reader.mutex.Unlock()
	if reader.offset == len(reader.data) {
		if reader.custom {
			return 0, &blockingFailure{mutex: reader.mutex}
		}
		return 0, io.EOF
	}
	count := copy(target, reader.data[reader.offset:])
	reader.offset += count
	return count, nil
}

func Result(text string, custom bool) (int, string) {
	target := make([]byte, 4)
	count, failure := io.ReadFull(&blockingReader{
		mutex: new(sync.Mutex),
		data: []byte(text),
		custom: custom,
	}, target)
	if failure != nil {
		return count, failure.Error()
	}
	return count, string(target)
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
			file.PackageName() == "ioreadprofile" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("io ReadFull fixture package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		`for (const [count, result] of [
  await Result("four", false),
  await Result("abc", false),
  await Result("abc", true),
]) {
  console.log(count + " " + JSON.stringify(result));
}
`,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	fixture "example.com/ioreadprofile"
)

func main() {
	for _, test := range []struct {
		text string
		custom bool
	}{{"four", false}, {"abc", false}, {"abc", true}} {
		count, result := fixture.Result(test.text, test.custom)
		fmt.Printf("%d %q\n", count, result)
	}
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
		t.Fatalf("execute Go io.ReadFull comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"io.ReadFull differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	if !strings.Contains(artifacts.printed, "IoReadFullCanonical") {
		t.Fatalf("io.ReadFull output lacks canonical boundary:\n%s", artifacts.printed)
	}
	if strings.Contains(artifacts.printed, "IoReadFullCanonicalSync") ||
		strings.Contains(artifacts.printed, "IoReadFullCanonicalAsync") {
		t.Fatalf("io.ReadFull output retained a profile variant:\n%s", artifacts.printed)
	}
}
