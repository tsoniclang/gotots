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
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		"const [missing, message] = await Result("+
			strconv.Quote(missing)+
			");\nconsole.log(missing + \" \" + message);\n",
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	fixture "example.com/providerunwrap"
)

func main() {
	missing, message := fixture.Result(`+strconv.Quote(missing)+`)
	fmt.Println(missing, message)
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
		t.Fatalf("execute Go provider-interface comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"provider dynamic-interface differential:\nGo:\n%s\nTypeScript:\n%s\nArtifacts:\n%s",
			sourceOutput,
			targetOutput,
			artifacts.printed,
		)
	}
}
