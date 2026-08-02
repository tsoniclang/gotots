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

func TestErrorsIsUsesTypedOptionalErrorProtocols(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/errorprotocols\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package errorprotocols

import "errors"

type codeError int

func (failure codeError) Error() string { return "code" }

func (failure codeError) Is(target error) bool {
	selected, ok := target.(codeError)
	return ok && failure%10 == selected%10
}

type wrappedError struct { failure error }

func (failure wrappedError) Error() string { return "wrapped" }
func (failure wrappedError) Unwrap() error { return failure.failure }

type joinedErrors []error

func (failure joinedErrors) Error() string { return "joined" }
func (failure joinedErrors) Unwrap() []error { return []error(failure) }

type sliceError []int

func (failure sliceError) Error() string { return "slice" }

func Results() (bool, bool, bool, bool, bool, bool) {
	sentinel := errors.New("sentinel")
	wrapped := wrappedError{failure: sentinel}
	joined := joinedErrors{errors.New("other"), wrapped}
	nonComparable := sliceError{1, 2, 3}
	return errors.Is(nil, nil),
		errors.Is(sentinel, sentinel),
		errors.Is(codeError(12), codeError(2)),
		errors.Is(wrapped, sentinel),
		errors.Is(joined, sentinel),
		errors.Is(nonComparable, nonComparable)
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
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(t, scope.Lookup("Results"))},
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
			file.PackageName() == "errorprotocols" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("error protocol package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Results"},
		`console.log(Results().map(String).join(" "));
`,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	protocols "example.com/errorprotocols"
)

func main() {
	a, b, c, d, e, f := protocols.Results()
	fmt.Println(a, b, c, d, e, f)
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
		t.Fatalf("execute Go error-protocol comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"errors.Is differential:\nGo:\n%s\nTypeScript:\n%s\nArtifacts:\n%s",
			sourceOutput,
			targetOutput,
			artifacts.printed,
		)
	}
	for _, required := range []string{
		"ErrorsIsCanonical",
		"readonly comparable: boolean",
		"$is",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("errors.Is output lacks %q:\n%s", required, artifacts.printed)
		}
	}
}

func TestErrorsIsSelectsCooperativeOptionalProtocol(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/asyncerrorprotocol\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package asyncerrorprotocol

import (
	"errors"
	"sync"
)

type blockingError struct {
	mutex *sync.Mutex
	code int
}

func (failure *blockingError) Error() string { return "blocking" }

func (failure *blockingError) Is(target error) bool {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	selected, ok := target.(*blockingError)
	return ok && failure.code == selected.code
}

func Result() bool {
	return errors.Is(
		&blockingError{mutex: &sync.Mutex{}, code: 42},
		&blockingError{mutex: &sync.Mutex{}, code: 42},
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
			file.PackageName() == "asyncerrorprotocol" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("cooperative error package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		"console.log(await Result());\n",
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	protocol "example.com/asyncerrorprotocol"
)

func main() { fmt.Println(protocol.Result()) }
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
		t.Fatalf("execute cooperative Go error comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"cooperative errors.Is differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	for _, required := range []string{
		"ErrorsIsCanonicalAsyncIs",
		"async Is(",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("cooperative errors.Is output lacks %q:\n%s", required, artifacts.printed)
		}
	}
}

func TestErrorsIsDoesNotInvokeCooperativeErrorMethod(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/asyncerrormethod\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package asyncerrormethod

import (
	"errors"
	"sync"
)

type blockingMessage struct {
	mutex *sync.Mutex
}

func (failure *blockingMessage) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "blocking"
}

func Result() bool {
	failure := &blockingMessage{mutex: &sync.Mutex{}}
	return errors.Is(failure, failure)
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
			file.PackageName() == "asyncerrormethod" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("cooperative Error package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		"console.log(Result());\n",
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	comparison "example.com/asyncerrormethod"
)

func main() { fmt.Println(comparison.Result()) }
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
		t.Fatalf("execute cooperative Error comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"cooperative Error errors.Is differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	for _, required := range []string{
		"ErrorsIsCanonicalAsyncErrorSync",
		"async Error(",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("cooperative Error output lacks %q:\n%s", required, artifacts.printed)
		}
	}
}

func TestProviderErrorResultUsesOneCanonicalBridge(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providererror\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providererror

import (
	"errors"
	"sync"
)

type blockingError struct { mutex *sync.Mutex }

func (failure blockingError) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "generated"
}

func ProviderError() error { return errors.New("provider") }

func GeneratedError() error {
	return blockingError{mutex: new(sync.Mutex)}
}

func Message(failure error) string { return failure.Error() }
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
			mustProviderRoot(t, scope.Lookup("ProviderError")),
			mustProviderRoot(t, scope.Lookup("GeneratedError")),
			mustProviderRoot(t, scope.Lookup("Message")),
		},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	if count := strings.Count(
		artifacts.printed,
		"class $goProviderInterfaceBridge",
	); count != 1 {
		t.Fatalf("provider error bridge definitions = %d, want 1:\n%s", count, artifacts.printed)
	}
	for _, required := range []string{
		"async Error(",
		"errors__from_gostdlib.New(\"provider\")",
		"implements $goInterface",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("provider error artifact lacks %q:\n%s", required, artifacts.printed)
		}
	}
}
