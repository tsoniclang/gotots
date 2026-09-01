package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

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
	options := providerNumberOptions()
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
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Results"},
		`console.log(Results().map(String).join(" "));
`,
	)
	for _, required := range []string{
		"ErrorsIsDirect",
		"readonly comparable: boolean",
		"$is",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("errors.Is output lacks %q:\n%s", required, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "ErrorsIsCanonical(") {
		t.Fatalf("direct errors.Is output selected a retired boundary:\n%s", artifacts.printed)
	}
	for _, view := range []string{
		"provider_error.AsProviderErrorIsDirect(value)",
		"provider_error.AsProviderErrorUnwrapDirect(value)",
		"provider_error.AsProviderErrorUnwrapManyDirect(value)",
	} {
		if count := strings.Count(artifacts.printed, view); count != 1 {
			t.Fatalf("direct provider capability view %q calls = %d, want 1:\n%s", view, count, artifacts.printed)
		}
	}
	for _, retiredView := range []string{
		"provider_error.AsProviderErrorIs(value)",
		"provider_error.AsProviderErrorUnwrap(value)",
		"provider_error.AsProviderErrorUnwrapMany(value)",
	} {
		if strings.Contains(artifacts.printed, retiredView) {
			t.Fatalf("direct provider bridge selected retired view %q:\n%s", retiredView, artifacts.printed)
		}
	}
}

func TestErrorsIsSelectsSynchronousOptionalProtocol(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/errorprotocolmethod\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package errorprotocolmethod

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
	options := providerNumberOptions()
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
			file.PackageName() == "errorprotocolmethod" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("synchronous error package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		"console.log(Result());\n",
	)
	for _, required := range []string{
		"ErrorsIsDirect",
		"Is(",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("synchronous errors.Is output lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"ErrorsIsCanonicalAsyncIs(",
		"ErrorsIsCanonicalAsyncUnwrap(",
		"ErrorsIsCanonicalAsyncUnwrapMany(",
		"async ",
		"await ",
		"Promise<",
		"Awaitable<",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("synchronous errors.Is retained %q:\n%s", forbidden, artifacts.printed)
		}
	}
}

func TestErrorsIsUsesSynchronousProtocolWithoutInvokingError(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/errorstringmethod\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package errorstringmethod

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
	options := providerNumberOptions()
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
			file.PackageName() == "errorstringmethod" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("synchronous Error package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		"console.log(Result());\n",
	)
	for _, required := range []string{
		"ErrorsIsDirect",
		"Error(",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("synchronous Error output lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{"async ", "await ", "Promise<", "Awaitable<"} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("synchronous Error output contains %q", forbidden)
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
	options := providerNumberOptions()
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
		"Error(",
		"errors__from_gostdlib.New(\"provider\")",
		"extends GoProviderInterfaceBridge<GoError> implements GoInterface",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("provider error artifact lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{"async ", "await ", "Promise<", "Awaitable<"} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("provider error artifact contains %q", forbidden)
		}
	}
}
