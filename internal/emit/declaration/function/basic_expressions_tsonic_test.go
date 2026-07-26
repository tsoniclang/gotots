package function_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBasicExpressionsExecuteDifferentiallyThroughTsonic(t *testing.T) {
	loaded := loadBasicExpressionsProject(t)
	workingDirectory := t.TempDir()
	targetFile := emitBasicExpressions(
		t,
		loaded,
		filepath.Join(workingDirectory, "index.ts"),
	)
	printed := printTargetFile(t, targetFile, workingDirectory)
	targetOutput := executeThroughTsonic(t, printed, tsonicProof{
		namespace: "GoToTS.BasicExpressions",
		assembly:  "GoToTSBasicExpressions",
		runnerSource: `Console.WriteLine(GoToTS.BasicExpressions.Index.Arithmetic(10));
Console.WriteLine(GoToTS.BasicExpressions.Index.WrapAdd(long.MaxValue));
Console.WriteLine(GoToTS.BasicExpressions.Index.WrapSubtract(long.MinValue));
Console.WriteLine(GoToTS.BasicExpressions.Index.WrapMultiply(long.MaxValue));
Console.WriteLine(GoToTS.BasicExpressions.Index.IntWrapAdd(long.MaxValue));
Console.WriteLine(GoToTS.BasicExpressions.Index.IntWrapSubtract(long.MinValue));
Console.WriteLine(GoToTS.BasicExpressions.Index.IntWrapMultiply(long.MaxValue));
Console.WriteLine(GoToTS.BasicExpressions.Index.Logic(true, false) ? "true" : "false");
Console.WriteLine(GoToTS.BasicExpressions.Index.ShortCircuitAnd() ? "true" : "false");
Console.WriteLine(GoToTS.BasicExpressions.Index.ShortCircuitOr() ? "true" : "false");
`,
		requiredTarget: []string{
			"return (value - (3)) * (2);",
			"return value + (1);",
			"return value - (1);",
			"return value * (2);",
			"return (left && !right) || (!left && right);",
			"return false && Never();",
			"return true || Never();",
		},
	})
	goOutput := executeBasicExpressionsBoundaryGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf("Tsonic/C# output = %q, Go output = %q", targetOutput, goOutput)
	}
}

func executeBasicExpressionsBoundaryGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(basicExpressionsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "boundary-go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/basicexpressions v0.0.0

replace example.com/basicexpressions => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	expressions "example.com/basicexpressions"
)

func main() {
	fmt.Println(expressions.Arithmetic(10))
	fmt.Println(expressions.WrapAdd(9223372036854775807))
	fmt.Println(expressions.WrapSubtract(-9223372036854775808))
	fmt.Println(expressions.WrapMultiply(9223372036854775807))
	fmt.Println(expressions.IntWrapAdd(9223372036854775807))
	fmt.Println(expressions.IntWrapSubtract(-9223372036854775808))
	fmt.Println(expressions.IntWrapMultiply(9223372036854775807))
	fmt.Println(expressions.Logic(true, false))
	fmt.Println(expressions.ShortCircuitAnd())
	fmt.Println(expressions.ShortCircuitOr())
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}
