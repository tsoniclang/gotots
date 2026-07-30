package emit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestClassMembersPreserveReceiverSemanticsDifferentially(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: classMemberMethodDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Audit"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	assertClassMemberMethodAST(t, emission)
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, forbidden := range []string{
		"export function Counter_Bump",
		"export function Counter_Read",
		"export function Counter_Reset",
		".bind(",
		".call(",
		".apply(",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("class-member artifact contains %q", forbidden)
		}
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	sourceModule := sourceModuleForExport(
		t,
		artifacts,
		workingDirectory,
		"Audit",
	)
	writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+sourceModule+`";

const values = Audit();
console.log("[" + Array.from({ length: values.length }, (_, index) =>
    String(values.get(index))).join(" ") + "]");
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(
		t,
		workingDirectory,
		append(artifacts.paths, runner),
	)
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeClassMemberMethodGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"class-member output differs\nTypeScript:\n%s\nGo:\n%s\nArtifacts:\n%s",
			targetOutput,
			goOutput,
			artifacts.printed,
		)
	}
}

func assertClassMemberMethodAST(
	t *testing.T,
	emission emit.ProgramEmission,
) {
	t.Helper()
	var counter tsgo.ClassDeclaration
	topLevelMethods := 0
	for _, file := range emission.Files() {
		for _, statement := range file.SourceFile().Statements() {
			switch statement := statement.(type) {
			case tsgo.ClassDeclaration:
				if statement.Name().Text() == "Counter" {
					if counter != nil {
						t.Fatal("Counter class is duplicated")
					}
					counter = statement
				}
			case tsgo.FunctionDeclaration:
				switch statement.Name().Text() {
				case "Counter_Bump", "Counter_Read", "Counter_Reset":
					topLevelMethods++
				}
			}
		}
	}
	if counter == nil || topLevelMethods != 0 {
		t.Fatalf(
			"Counter class = %T, top-level receiver methods = %d",
			counter,
			topLevelMethods,
		)
	}
	methods := make(map[string]tsgo.MethodDeclaration)
	for _, member := range counter.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok {
			continue
		}
		name, ok := method.Name().(tsgo.Identifier)
		if ok {
			methods[name.Text()] = method
		}
	}
	for _, name := range []string{"Bump", "Read", "Reset"} {
		if methods[name] == nil {
			t.Fatalf("Counter.%s is absent", name)
		}
	}
	for name := range methods {
		if strings.HasPrefix(name, "$go$private_") {
			t.Fatalf("unreached private method %q entered Counter", name)
		}
	}
	bumpBody := methods["Bump"].Body().(tsgo.Block).Statements()
	readBody := methods["Read"].Body().(tsgo.Block).Statements()
	if _, ok := bumpBody[0].(tsgo.VariableStatement); !ok {
		t.Fatalf("Counter.Bump first statement = %T, want receiver copy", bumpBody[0])
	}
	if _, ok := readBody[0].(tsgo.ReturnStatement); !ok {
		t.Fatalf("Counter.Read first statement = %T, want direct this read", readBody[0])
	}
	if hasStaticModifier(methods["Bump"]) ||
		hasStaticModifier(methods["Read"]) ||
		!hasStaticModifier(methods["Reset"]) {
		t.Fatal("class receiver method staticness differs")
	}
}

func hasStaticModifier(method tsgo.MethodDeclaration) bool {
	for _, modifier := range method.Modifiers() {
		if modifier.Kind() == tsgo.SyntaxKindStaticKeyword {
			return true
		}
	}
	return false
}

func executeClassMemberMethodGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(classMemberMethodDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-class-members",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			"module example.com/runner\n\ngo 1.26.4\n\n"+
				"require example.com/classmembers v0.0.0\n\n"+
				"replace example.com/classmembers => %s\n",
			filepath.ToSlash(modulePath),
		),
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "main.go"),
		`package main

import (
	"fmt"

	values "example.com/classmembers"
)

func main() {
	fmt.Println(values.Audit())
}
`,
	)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func classMemberMethodDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"method",
		"class-members",
	)
}
