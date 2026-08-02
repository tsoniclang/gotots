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

func TestContextProviderProfileConstructsExactInterfaceValue(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/contextprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package contextprofile

import (
	"context"
	"sync"
	"time"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "parent failed"
}

type fixedContext struct { failure error }

func (*fixedContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*fixedContext) Done() <-chan struct{} { return nil }
func (source *fixedContext) Err() error { return source.failure }
func (*fixedContext) Value(any) any { return nil }

func Run(key string, value string) (string, bool, string) {
	background := context.Background()
	todo := context.TODO()
	parent := &fixedContext{failure: &blockingFailure{mutex: &sync.Mutex{}}}
	child := context.WithValue(parent, key, value)
	selected, ok := child.Value(key).(string)
	identities := ok && child.Value("missing") == nil &&
		background == context.Background() && todo == context.TODO() &&
		background != todo
	return selected, identities, child.Err().Error()
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
			program.Roots()[0].Types().Scope().Lookup("Run"),
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
			file.PackageName() == "contextprofile" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("context profile package assembly is absent")
	}
	if !strings.Contains(artifacts.printed, "export class RuntimeSlice") {
		t.Fatalf("context provider runtime closure lacks RuntimeSlice:\n%s", artifacts.printed)
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Run"},
		`const [value, identities, failure] = await Run("request", "alpha");
console.log(JSON.stringify(value) + " " + identities + " " + JSON.stringify(failure));
`,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	profile "example.com/contextprofile"
)

func main() {
	value, identities, failure := profile.Run("request", "alpha")
	fmt.Printf("%q %t %q\n", value, identities, failure)
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
		t.Fatalf("execute Go context comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"context provider differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	for _, required := range []string{
		"ContextWithValueCanonicalSync",
		"context__from_gostdlib.Background",
		"context__from_gostdlib.TODO",
		"$contract);",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("context provider output lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"$from(provider_context.ContextWithValueCanonicalSync",
		" as any",
		" as unknown",
		".apply(",
		".call(",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("context provider output contains %q:\n%s", forbidden, artifacts.printed)
		}
	}
}
