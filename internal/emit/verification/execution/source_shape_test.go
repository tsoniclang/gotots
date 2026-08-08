package emit_test

import (
	"context"
	"fmt"
	"go/types"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSourceFunctionDeclarationsPreserveGoParameterArity(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	sourcePackage := program.Roots()[0]
	root, err := emit.NewRoot(
		sourcePackage.Types().Scope().Lookup("RecoveryCallableForms"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}

	expected := make(map[string]int)
	for _, name := range sourcePackage.Types().Scope().Names() {
		function, ok := sourcePackage.Types().Scope().Lookup(name).(*types.Func)
		if !ok {
			continue
		}
		signature, ok := function.Type().(*types.Signature)
		if !ok || signature.Recv() != nil {
			continue
		}
		expected[name] = signature.Params().Len()
	}

	checked := 0
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource ||
			file.PackageName() != sourcePackage.Types().Name() {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name() == nil {
				continue
			}
			want, sourceOwned := expected[function.Name().Text()]
			if !sourceOwned {
				continue
			}
			checked++
			if got := len(function.Parameters()); got != want {
				t.Errorf(
					"%s target parameters = %d, Go value parameters = %d",
					function.Name().Text(),
					got,
					want,
				)
			}
		}
	}
	if checked == 0 {
		t.Fatal("source-shape gate inspected no source function declarations")
	}
}

func TestTransportedCallableTypeHasNoRecoveryParameter(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("RecoveryCallableForms"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	for _, sourceFragment := range []string{
		"public Call:",
		"Pointer<",
		"RuntimeSlice.literal<",
	} {
		index := strings.Index(artifacts.printed, sourceFragment)
		if index < 0 {
			t.Fatalf("fixture lacks transported callable fragment %q", sourceFragment)
		}
		end := strings.IndexByte(artifacts.printed[index:], '\n')
		if end < 0 {
			end = len(artifacts.printed) - index
		}
		line := artifacts.printed[index : index+end]
		if strings.Contains(line, "$go$recovery") ||
			strings.Contains(line, "GoRecovery") {
			t.Errorf("source callable type leaks recovery authority: %s", line)
		}
	}
}

func TestRecoveringFunctionHasSeparatePrivateDeferredEntry(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	sourcePackage := program.Roots()[0]
	root, err := emit.NewRoot(
		sourcePackage.Types().Scope().Lookup("RecoveryCallableForms"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]int{
		"recoverFunction":          1,
		"recoverFunction$deferred": 2,
	}
	seen := make(map[string]bool)
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource ||
			file.PackageName() != sourcePackage.Types().Name() {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name() == nil {
				continue
			}
			name := function.Name().Text()
			parameterCount, selected := want[name]
			if !selected {
				continue
			}
			seen[name] = true
			if got := len(function.Parameters()); got != parameterCount {
				t.Errorf("%s parameters = %d, want %d", name, got, parameterCount)
			}
			if name == "recoverFunction$deferred" {
				identifier, ok := function.Parameters()[0].Name().(tsgo.Identifier)
				if !ok || identifier.Text() != "$go$recovery" {
					t.Errorf("private deferred entry does not own recovery first")
				}
			}
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("missing callable %s", name)
		}
	}
}

func TestSourceMethodsAndInterfacesExposeNoDeferredEntry(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("RecoveryCallableForms"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}

	classes := 0
	interfaces := 0
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			switch statement := statement.(type) {
			case tsgo.ClassDeclaration:
				classes++
				for _, member := range statement.Members() {
					method, ok := member.(tsgo.MethodDeclaration)
					if !ok {
						continue
					}
					name, ok := method.Name().(tsgo.Identifier)
					if ok && strings.HasSuffix(name.Text(), "$deferred") {
						t.Errorf(
							"source class %s exposes private recovery member %s",
							statement.Name().Text(),
							name.Text(),
						)
					}
				}
			case tsgo.InterfaceDeclaration:
				interfaces++
				for _, member := range statement.Members() {
					method, ok := member.(tsgo.MethodSignatureDeclaration)
					if !ok {
						continue
					}
					name, ok := method.Name().(tsgo.Identifier)
					if ok && strings.HasSuffix(name.Text(), "$deferred") {
						t.Errorf(
							"source interface %s exposes private recovery member %s",
							statement.Name().Text(),
							name.Text(),
						)
					}
				}
			}
		}
	}
	if classes == 0 || interfaces == 0 {
		t.Fatalf(
			"source-shape gate inspected %d classes and %d interfaces",
			classes,
			interfaces,
		)
	}
}

func TestRecoveryCallableFormsCanonicalizeWithNativeEvidence(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("RecoveryCallableForms"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{"registerMethod(", "resolveMethod("} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("private deferred dispatch lacks %q", required)
		}
	}
	if count := strings.Count(
		artifacts.printed,
		"export class GoDeferredRegistry<",
	); count != 1 {
		t.Fatalf("deferred runtime implementation count = %d, want one", count)
	}
	if instances := strings.Count(
		artifacts.printed,
		"export const $goDeferred_",
	); instances == 0 {
		t.Fatal("typed deferred registry instances are absent")
	}
	if strings.Contains(artifacts.printed, "export class $goDeferred_") {
		t.Fatal("callable signatures duplicate the deferred registry implementation")
	}
	runner := filepath.Join(workingDirectory, "runner.ts")
	sourceModule := sourceModuleForExport(
		t,
		artifacts,
		workingDirectory,
		"RecoveryCallableForms",
	)
	writeProgramFile(t, runner, `import "./program.js";
import { RecoveryCallableForms } from "`+sourceModule+`";

console.log(RecoveryCallableForms());
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
	goOutput := executeRecoveryCallableFormsGo(t, workingDirectory)
	requireNativeGoEvidence(t, goOutput)
}

func executeRecoveryCallableFormsGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveEightControlDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-recovery-runner")
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			"module example.com/recoveryrunner\n\ngo 1.26.4\n\n"+
				"require example.com/wave8control v0.0.0\n\n"+
				"replace example.com/wave8control => %s\n",
			filepath.ToSlash(modulePath),
		),
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "main.go"),
		`package main

import (
	"fmt"

	values "example.com/wave8control"
)

func main() {
	fmt.Println(values.RecoveryCallableForms())
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
