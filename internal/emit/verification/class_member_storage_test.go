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

func TestClassMembersPreserveReceiverSemantics(t *testing.T) {
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
		".bind(",
		".call(",
		".apply(",
		"Counter.Reset(Counter.$copy(",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("class-member artifact contains %q", forbidden)
		}
	}
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(
		t,
		workingDirectory,
		artifacts.paths,
	)
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
		if strings.HasPrefix(name, "$go$private$") {
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

func TestLocalNamedStructPointerStaysDirectAndLexical(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			program, err := load.Load(context.Background(), load.Request{
				Directory: localStructStorageDirectory(),
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
			emission, err := emit.CompileWithOptions(
				program,
				[]emit.Root{root},
				testCase.options,
			)
			if err != nil {
				t.Fatal(err)
			}
			workingDirectory := t.TempDir()
			artifacts := materializeArtifacts(t, emission, workingDirectory)
			for _, required := range []string{
				"class record",
				"let value: Pointer<record> | undefined = allocatePointer<record>(",
				"loadPointer<record>(",
			} {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf(
						"local struct storage artifact lacks %q:\n%s",
						required,
						artifacts.printed,
					)
				}
			}
			for _, forbidden := range []string{
				"export class record",
				"record$Storage",
				"GoPointer",
				"runtime/pointer",
			} {
				if !strings.Contains(artifacts.printed, forbidden) {
					continue
				}
				t.Fatalf(
					"direct local struct contains %q:\n%s",
					forbidden,
					artifacts.printed,
				)
			}
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			waveThreeTypecheck(
				t,
				workingDirectory,
				artifacts.paths,
			)
		})
	}
}

func executeLocalStructStorageGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(localStructStorageDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-local-struct-storage",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/localstructstorage v0.0.0

replace example.com/localstructstorage => %s
`,
			filepath.ToSlash(modulePath),
		),
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "main.go"),
		`package main

import (
	"fmt"

	values "example.com/localstructstorage"
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

func localStructStorageDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"type",
		"local-struct-storage",
	)
}

func TestGenericInterfaceCallableFamilyConverges(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveNineConcurrencyDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().
			Lookup("GenericInterfaceAudit"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{root},
		waveNineOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	for _, required := range []string{
		"export interface GenericValue<T>",
		"export interface IntValue",
		"Value(): T;",
		"Value(): int32;",
		"export function GenericInterfaceAudit(): int32",
		"const __gotots_argument_0 = goInterfaceNonNil<GenericValue<T>>(__gotots_receiver_0).Value();",
		"return $go$copy$",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"generic interface callable family lacks %q:\n%s",
				required,
				artifacts.printed,
			)
		}
	}
	for _, forbidden := range []string{"async ", "await ", "Promise<", "Awaitable<"} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("generic interface callable family contains %q", forbidden)
		}
	}
	if methods := strings.Count(artifacts.printed, "Value$deferred"); methods != 0 {
		t.Fatalf(
			"non-recovering generic interface adapters have %d recovery entries, want zero:\n%s",
			methods,
			artifacts.printed,
		)
	}
	if tokens := strings.Count(
		artifacts.printed,
		"export const $goInterfaceMethod$",
	); tokens != 2 {
		t.Fatalf(
			"generic interface runtime tokens = %d, want two closed signatures",
			tokens,
		)
	}
	if strings.Contains(artifacts.printed, "$goInterfaceCallable$") {
		t.Fatal("contract-only interface callable leaked into TypeScript output")
	}
	for _, forbidden := range []string{"Value(): Promise<T>;", "Value(): Awaitable<T>;"} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatal("generic interface retained an asynchronous callable contract")
		}
	}
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	waveThreeTypecheck(
		t,
		workingDirectory,
		artifacts.paths,
	)
}

func executeGenericInterfaceCallableGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(waveNineConcurrencyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(
		workingDirectory,
		"go-runner-generic-interface-callable",
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "go.mod"),
		fmt.Sprintf(
			`module example.com/runner

go 1.26.4

require example.com/wave9concurrency v0.0.0

replace example.com/wave9concurrency => %s
`,
			filepath.ToSlash(modulePath),
		),
	)
	writeProgramFile(
		t,
		filepath.Join(runnerDirectory, "main.go"),
		`package main

import (
	"fmt"

	values "example.com/wave9concurrency"
)

func main() {
	fmt.Println(values.GenericInterfaceAudit())
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
