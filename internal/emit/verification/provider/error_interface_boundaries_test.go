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

func TestProviderErrorPreservesDynamicUnwrapInterface(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/providerunwrap\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package providerunwrap

import "os"

func Result(path string) (bool, string) {
	_, failure := os.Stat(path)
	return os.IsNotExist(failure), failure.Error()
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
	for _, view := range []string{
		"provider_error.AsProviderErrorIs(value)",
		"provider_error.AsProviderErrorUnwrap(value)",
		"provider_error.AsProviderErrorUnwrapMany(value)",
	} {
		if count := strings.Count(artifacts.printed, view); count != 2 {
			t.Fatalf("canonical provider capability view %q calls = %d, want 2:\n%s", view, count, artifacts.printed)
		}
	}
	for _, providerInternal := range []string{
		"AsProviderErrorIsDirect",
		"AsProviderErrorUnwrapDirect",
		"AsProviderErrorUnwrapManyDirect",
	} {
		if strings.Contains(artifacts.printed, providerInternal) {
			t.Fatalf("provider-internal capability %q escaped into generated output:\n%s", providerInternal, artifacts.printed)
		}
	}
	if count := strings.Count(artifacts.printed, "Unwrap(): Promise<"); count != 9 {
		t.Fatalf("Unwrap overload/implementation declarations = %d, want 9:\n%s", count, artifacts.printed)
	}
	for _, dynamicProbe := range []string{
		`["Is"]`,
		`["Unwrap"]`,
		`"Is" in `,
		`"Unwrap" in `,
	} {
		if strings.Contains(artifacts.printed, dynamicProbe) {
			t.Fatalf("provider bridge contains dynamic capability probe %q:\n%s", dynamicProbe, artifacts.printed)
		}
	}
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "providerunwrap" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("provider dynamic-interface package assembly is absent")
	}
	missing := filepath.Join(project, "absent")
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		"const [missing, message] = await Result("+
			strconv.Quote(missing)+
			");\nconsole.log(missing + \" \" + message);\n",
	)
}

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
	typecheckProviderRunner(
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
	if !strings.Contains(
		artifacts.printed,
		"ErrorsUnwrapCanonical",
	) {
		t.Fatalf("errors.Unwrap output lacks canonical boundary:\n%s", artifacts.printed)
	}
	for _, rejected := range []string{
		"ErrorsUnwrapCanonicalSync",
		"ErrorsUnwrapCanonicalAsync",
		"ErrorsUnwrapCanonicalAsyncErrorSync",
	} {
		if strings.Contains(artifacts.printed, rejected) {
			t.Fatalf("errors.Unwrap output selected %q:\n%s", rejected, artifacts.printed)
		}
	}
}
