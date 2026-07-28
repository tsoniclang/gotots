package function_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestFunctionLiteralAndSignatureOwnersCreateNativeTargetTree(t *testing.T) {
	loaded := loadCallableValuesProject(t)
	root, err := emit.NewRoot(loaded.Types().Scope().Lookup("Offset"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	var sourceFile tsgo.SourceFile
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource {
			sourceFile = file.SourceFile()
			break
		}
	}
	if sourceFile == nil {
		t.Fatal("Offset source target is absent")
	}
	offset := targetFunction(t, sourceFile, "Offset")
	literal, ok := targetReturn(t, offset).Expression().(tsgo.FunctionExpression)
	if !ok {
		t.Fatalf("Offset return = %T, want FunctionExpression", targetReturn(t, offset).Expression())
	}
	offsetType, ok := offset.Type().(tsgo.UnionTypeNode)
	if !ok || len(offsetType.Types()) != 2 {
		t.Fatalf("Offset result type = %T, want callable | undefined", offset.Type())
	}
	if _, ok := offsetType.Types()[0].(tsgo.FunctionTypeNode); !ok {
		t.Fatalf("Offset non-nil result type = %T, want FunctionTypeNode", offsetType.Types()[0])
	}
	if len(literal.Parameters()) != 1 {
		t.Fatalf("literal parameters = %d, want one", len(literal.Parameters()))
	}
}

func TestCallableValuesPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadCallableValuesProject(t)
	workingDirectory := t.TempDir()
	targetFile := compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
	printed := printTargetFile(t, targetFile, workingDirectory)

	for _, forbidden := range []string{".call(", ".apply(", ".bind(", "any", "unknown"} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("printed callable artifact contains %q:\n%s", forbidden, printed)
		}
	}
	if !strings.Contains(
		printed,
		"transform: (($0: int32) => int32) | undefined",
	) || !strings.Contains(
		printed,
		"return function (value: int32): int32",
	) || !strings.Contains(
		printed,
		`GoPanic.raise("call of nil function")`,
	) || !strings.Contains(
		printed,
		"return Apply(Double, value);",
	) {
		t.Fatalf("printed callable artifact is not direct:\n%s", printed)
	}
	if strings.Contains(printed, "__gotots_parameter_") {
		t.Fatalf("printed callable artifact contains wide synthetic parameter names:\n%s", printed)
	}
	ordered := printedFunction(t, printed, "OrderedCalleeAndArguments")
	calleeCapture := strings.Index(ordered, "const __gotots_callee_")
	argumentCapture := strings.Index(ordered, "const __gotots_results_")
	guard := strings.Index(ordered, "if (__gotots_callee_")
	if calleeCapture < 0 ||
		argumentCapture < 0 ||
		guard < 0 ||
		!(calleeCapture < argumentCapture && argumentCapture < guard) {
		t.Fatalf(
			"callee is not captured before argument prerequisites and guard:\n%s",
			ordered,
		)
	}

	goOutput := executeCallableValuesGo(t, workingDirectory)
	typeScriptOutput := executeCallableValuesTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestCallableValuesCreateNativeTargetTrees(t *testing.T) {
	loaded := loadCallableValuesProject(t)
	targetFile := compileSourceFile(t, loaded, loaded.Files()[0].Syntax())

	apply := targetFunction(t, targetFile, "Apply")
	statements := apply.Body().(tsgo.Block).Statements()
	if len(statements) != 4 {
		t.Fatalf("Apply statements = %d, want callee, argument, guard, return", len(statements))
	}
	result, ok := statements[3].(tsgo.ReturnStatement)
	if !ok {
		t.Fatalf("Apply final statement = %T, want return", statements[3])
	}
	call, ok := result.Expression().(tsgo.CallExpression)
	if !ok {
		t.Fatalf("Apply return = %T, want direct call", result.Expression())
	}
	callee, ok := call.Expression().(tsgo.Identifier)
	if !ok || !strings.HasPrefix(callee.Text(), "__gotots_callee_") {
		t.Fatalf("Apply callee = %T %#v, want captured callable", call.Expression(), call.Expression())
	}
	if _, ok := statements[0].(tsgo.VariableStatement); !ok {
		t.Fatalf("Apply callee capture = %T, want variable statement", statements[0])
	}
	if _, ok := statements[1].(tsgo.VariableStatement); !ok {
		t.Fatalf("Apply argument capture = %T, want variable statement", statements[1])
	}
	if _, ok := statements[2].(tsgo.IfStatement); !ok {
		t.Fatalf("Apply guard = %T, want if statement", statements[2])
	}

	offset := targetFunction(t, targetFile, "Offset")
	literal, ok := targetReturn(t, offset).Expression().(tsgo.FunctionExpression)
	if !ok {
		t.Fatalf("Offset return = %T, want FunctionExpression", targetReturn(t, offset).Expression())
	}
	if len(literal.Parameters()) != 1 || literal.Parameters()[0].Name().Kind() != tsgo.SyntaxKindIdentifier {
		t.Fatal("Offset literal parameter is not a native typed parameter")
	}

	ignore := targetFunction(t, targetFile, "Ignore")
	if len(ignore.Parameters()) != 1 {
		t.Fatalf("Ignore parameters = %d, want one synthesized target parameter", len(ignore.Parameters()))
	}
}

func TestCrossPackageFunctionValueTypechecksAndExecutesDifferentially(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: callbackDemandProjectDirectory(),
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	modulePath, err := filepath.Abs(callbackDemandProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	goRunner := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(goRunner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/callbackdemand v0.0.0

replace example.com/callbackdemand => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(goRunner, "main.go"), `package main

import (
	"fmt"

	"example.com/callbackdemand/api"
)

func main() {
	fmt.Println(api.Run(21))
}
`)
	goOutput := run(t, goRunner, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")

	artifacts := materializeExportedProgram(t, program.Roots()[0], workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import { Run } from "`+artifacts.module(t, "api.ts")+`";

console.log(Run(21));
`)
	typeScriptOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func loadCallableValuesProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: callableValuesProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func executeCallableValuesGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(callableValuesProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/callablevalues v0.0.0

replace example.com/callablevalues => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	callable "example.com/callablevalues"
)

func main() {
	fmt.Println(callable.UseNamed(4))
	fmt.Println(callable.UseClosure(5, 3))
	fmt.Println(callable.UseReturned(7, true))
	fmt.Println(callable.UseReturned(7, false))
	fmt.Println(callable.Immediate(9))
	fmt.Println(callable.Ignore(123))
	fmt.Println(callable.UseExplicit(6, -2))
	fmt.Println(callable.UseCounter(3))
	fmt.Println(callable.OrderedCalleeAndArguments())
	fmt.Println(callable.UseItem(4))
	fmt.Println(callable.UsePair(5))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeCallableValuesTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
	    Ignore,
	    Immediate,
	    OrderedCalleeAndArguments,
	    UseClosure,
    UseCounter,
	    UseExplicit,
	    UseItem,
	    UseNamed,
	    UsePair,
	    UseReturned,
} from "`+artifacts.module(t, "source.ts")+`";

console.log(UseNamed(4));
console.log(UseClosure(5, 3));
console.log(UseReturned(7, true));
console.log(UseReturned(7, false));
console.log(Immediate(9));
console.log(Ignore(123));
console.log(UseExplicit(6, -2));
console.log(UseCounter(3));
console.log(OrderedCalleeAndArguments());
console.log(UseItem(4));
console.log(...UsePair(5));
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func callableValuesProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"expression",
		"function-literal",
		"callable-values",
	)
}

func callbackDemandProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"expression",
		"function-value",
		"cross-package",
	)
}
