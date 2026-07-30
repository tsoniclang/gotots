package emit_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBlankIdentifierDispositionsPrintTypecheckAndExecuteDifferentially(
	t *testing.T,
) {
	projectDirectory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "go.mod"),
		"module example.com/blankidentifier\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "source.go"),
		`package blankidentifier

const (
	_ int32 = iota
	one
)

const _ = 1 << 10
type _ int32
func _() {}

type Item struct{}

func (Item) _() {}
func (_ Item) Value() int32 { return 3 }
func (_ Item) Speak(_ int32) int32 { return 4 }

type Speaker interface {
	Speak(_ int32) int32
}

func zero() (_ int32) {
	defer func() {}()
	return
}
func ignore(_ int32, _ int32) int32 { return 5 }
func generic[_ any](value int32) int32 { return value }

func Run() int32 {
	const _ = 5
	type _ int32
	literal := func(_ int32) int32 { return 6 }
	var speaker Speaker = Item{}
	return one +
		Item{}.Value() +
		speaker.Speak(0) +
		zero() +
		ignore(0, 0) +
		generic[string](7) +
		literal(0)
}
`,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	rootObject := program.Roots()[0].Types().Scope().Lookup("Run")
	root, err := emit.NewRoot(rootObject)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}

	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	blankDefinition := regexp.MustCompile(
		`\b(?:function|const|let|class|interface|type)\s+_\b`,
	)
	if blankDefinition.MatchString(artifacts.printed) {
		t.Fatalf("blank source declaration leaked into TypeScript:\n%s", artifacts.printed)
	}
	for _, expected := range []string{"$0", "$1", "$T0", "$result0"} {
		if !strings.Contains(artifacts.printed, expected) {
			t.Fatalf(
				"target-only blank slot %q is absent:\n%s",
				expected,
				artifacts.printed,
			)
		}
	}

	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(
		t,
		runnerPath,
		`import "./program.js";
import { Run } from "`+artifacts.sourceModule+`";

console.log(Run());
`,
	)
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, artifacts.paths...)
	arguments = append(arguments, runnerPath)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
	if targetOutput != "26\n" {
		t.Fatalf("TypeScript output = %q, want Go-equivalent 26", targetOutput)
	}

	goTest := filepath.Join(projectDirectory, "source_test.go")
	writeProgramFile(
		t,
		goTest,
		`package blankidentifier

import "testing"

func TestRun(t *testing.T) {
	if got := Run(); got != 26 {
		t.Fatalf("Run() = %d, want 26", got)
	}
}
`,
	)
	commandContext, commandCancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer commandCancel()
	command := exec.CommandContext(
		commandContext,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"test",
		".",
	)
	command.Dir = projectDirectory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Go differential: %v\n%s", err, output)
	}
}
