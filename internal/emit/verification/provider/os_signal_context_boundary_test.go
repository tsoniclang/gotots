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

func TestNotifyContextPreservesCanonicalContextAndError(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/signalcontext\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package signalcontext

import (
	"context"
	"os"
	"os/signal"
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

func Result() (bool, string, string) {
	parent := &fixedContext{
		failure: &blockingFailure{mutex: new(sync.Mutex)},
	}
	parentMessage := parent.Err().Error()
	child, stop := signal.NotifyContext(parent, os.Interrupt)
	stop()
	childMessage := ""
	if failure := child.Err(); failure != nil {
		childMessage = failure.Error()
	}
	return child != nil, parentMessage, childMessage
}

var _ context.Context = (*fixedContext)(nil)
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
			file.PackageName() == "signalcontext" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("signal context fixture package assembly is absent")
	}
	targetOutput := executeProviderTypeScript(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Result"},
		`const [present, parent, child] = await Result();
console.log(present + " " + JSON.stringify(parent) + " " + JSON.stringify(child));
`,
	)
	runnerDirectory := filepath.Join(project, "cmd", "compare")
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	fixture "example.com/signalcontext"
)

func main() {
	present, parent, child := fixture.Result()
	fmt.Printf("%t %q %q\n", present, parent, child)
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
		t.Fatalf("execute Go signal.NotifyContext comparison: %v\n%s", err, sourceOutput)
	}
	if targetOutput != string(sourceOutput) {
		t.Fatalf(
			"signal.NotifyContext differential:\nGo:\n%s\nTypeScript:\n%s",
			sourceOutput,
			targetOutput,
		)
	}
	if !strings.Contains(
		artifacts.printed,
		"OsSignalNotifyContextCanonicalContext<",
	) {
		t.Fatalf("NotifyContext output lacks context profile:\n%s", artifacts.printed)
	}
	if strings.Contains(
		artifacts.printed,
		"OsSignalNotifyContextCanonicalContextSignal",
	) {
		t.Fatalf("NotifyContext output canonicalized an exact signal:\n%s", artifacts.printed)
	}
}
