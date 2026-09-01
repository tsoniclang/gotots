package function_test

import (
	"context"
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDirectSeamsPrintTypecheckAndExecuteDifferentially(t *testing.T) {
	loaded := loadDirectSeamsProject(t)
	workingDirectory := t.TempDir()
	targetFile := compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
	printed := printExecutableTargetFile(t, targetFile, workingDirectory)
	for _, expected := range []string{
		"const low: int32 = 3;",
		"if (value < low)",
		"for (current = 0; current < limit; current = current + 1)",
		"for (Touch(); value < 2; Touch())",
		"let __gotots_for_first_",
		"for (;;)",
	} {
		if !strings.Contains(printed, expected) {
			t.Fatalf("printed direct-seam artifact lacks %q:\n%s", expected, printed)
		}
	}

	goOutput := executeDirectSeamsGo(t, workingDirectory)
	typeScriptOutput := executeDirectSeamsTypeScript(t, loaded, workingDirectory)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func TestExpressionlessSwitchUsesConditionalChain(t *testing.T) {
	loaded := loadDirectSeamsProject(t)
	targetFile := compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
	classify := targetFunction(t, targetFile, "Classify")
	statements := classify.Body().(tsgo.Block).Statements()
	targetLabel, ok := statements[2].(tsgo.LabeledStatement)
	if !ok {
		t.Fatalf("Classify statement 2 = %T, want LabeledStatement", statements[2])
	}
	targetBlock, ok := targetLabel.Statement().(tsgo.Block)
	if !ok || len(targetBlock.Statements()) != 1 {
		t.Fatalf("expressionless switch target = %#v, want one-statement block", targetLabel.Statement())
	}
	if _, ok := targetBlock.Statements()[0].(tsgo.IfStatement); !ok {
		t.Fatalf("expressionless switch body = %T, want IfStatement", targetBlock.Statements()[0])
	}
}

func TestDirectSeamMutationsFailAtSemanticOwners(t *testing.T) {
	t.Run("switch case loses boolean type", func(t *testing.T) {
		loaded := loadDirectSeamsProject(t)
		classify := sourceFunction(t, loaded.Files()[0].Syntax(), "Classify")
		sourceSwitch := classify.Body.List[2].(*ast.SwitchStmt)
		clause := sourceSwitch.Body.List[0].(*ast.CaseClause)
		caseExpression := clause.List[0]
		typeAndValue := loaded.TypesInfo().Types[caseExpression]
		typeAndValue.Type = types.Typ[types.Int32]
		loaded.TypesInfo().Types[caseExpression] = typeAndValue

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleSwitchCaseExpression,
			api.CategoryExpression,
			"*ast.BinaryExpr",
		)
	})
}

func loadDirectSeamsProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directSeamsProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func executeDirectSeamsGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(directSeamsProjectDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/directseams v0.0.0

replace example.com/directseams => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	seams "example.com/directseams"
)

func main() {
	fmt.Println(seams.Classify(1))
	fmt.Println(seams.Classify(5))
	fmt.Println(seams.Classify(9))
	fmt.Println(seams.ClosureCondition(true))
	fmt.Println(seams.ClosureCondition(false))
	fmt.Println(seams.Constants())
	fmt.Println(seams.Scoped(-2))
	fmt.Println(seams.Scoped(4))
	fmt.Println(seams.ScopedParallel(3))
	fmt.Println(seams.Loop(5))
	fmt.Println(seams.CallClauses(0))
	fmt.Println(seams.IncInitializers(0))
	fmt.Println(seams.ParallelInitializer())
	fmt.Println(seams.ParallelPost(7))
	fmt.Println(seams.ConditionPrerequisite(4))
}
`)
	return run(t, runnerDirectory, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
}

func executeDirectSeamsTypeScript(
	t *testing.T,
	loaded *load.Package,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
	    CallClauses,
	    Classify,
	    ClosureCondition,
	    Constants,
	    IncInitializers,
	    Loop,
	    ParallelInitializer,
	    ParallelPost,
	    ConditionPrerequisite,
	    Scoped,
	    ScopedParallel,
	} from "`+artifacts.module(t, "source.ts")+`";

console.log(Classify(1));
console.log(Classify(5));
console.log(Classify(9));
console.log(ClosureCondition(true));
console.log(ClosureCondition(false));
console.log(Constants());
console.log(Scoped(-2));
console.log(Scoped(4));
console.log(ScopedParallel(3));
console.log(Loop(5));
console.log(CallClauses(0));
console.log(IncInitializers(0));
console.log(ParallelInitializer());
console.log(ParallelPost(7));
console.log(ConditionPrerequisite(4));
`)
	return executeMaterializedTypeScript(t, workingDirectory, artifacts, runnerPath)
}

func directSeamsProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"statement",
		"direct-seams",
	)
}
