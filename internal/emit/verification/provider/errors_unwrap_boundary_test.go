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

func TestErrorsUnwrapPreservesCanonicalOptionalProtocol(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/errorsunwrap\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package errorsunwrap

import (
	"errors"
	"sync"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "cause"
}

type blockingWrapper struct {
	mutex *sync.Mutex
	cause error
}

func (wrapper *blockingWrapper) Error() string {
	wrapper.mutex.Lock()
	wrapper.mutex.Unlock()
	return "wrapper"
}

func (wrapper *blockingWrapper) Unwrap() error {
	wrapper.mutex.Lock()
	wrapper.mutex.Unlock()
	return wrapper.cause
}

func Result(wrapped bool) (string, bool) {
	mutex := new(sync.Mutex)
	cause := &blockingFailure{mutex: mutex}
	var source error = cause
	if wrapped {
		source = &blockingWrapper{mutex: mutex, cause: cause}
	}
	selected := errors.Unwrap(source)
	if selected == nil {
		return "", false
	}
	return selected.Error(), true
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
			file.PackageName() == "errorsunwrap" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("errors.Unwrap fixture package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		`for (const wrapped of [true, false]) {
  const [message, ok] = await Result(wrapped);
  console.log(JSON.stringify(message) + " " + ok);
}
`,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	fixture "example.com/errorsunwrap"
)

func main() {
	for _, wrapped := range []bool{true, false} {
		message, ok := fixture.Result(wrapped)
		fmt.Printf("%q %t\n", message, ok)
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
		t.Fatalf("execute Go errors.Unwrap comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"errors.Unwrap differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	if !strings.Contains(
		artifacts.printed,
		"ErrorsUnwrapCanonicalAsyncErrorAsync",
	) {
		t.Fatalf("errors.Unwrap output lacks exact async profile:\n%s", artifacts.printed)
	}
	for _, rejected := range []string{
		"ErrorsUnwrapCanonicalSync",
		"ErrorsUnwrapCanonicalAsyncErrorSync",
	} {
		if strings.Contains(artifacts.printed, rejected) {
			t.Fatalf("errors.Unwrap output selected %q:\n%s", rejected, artifacts.printed)
		}
	}
}
