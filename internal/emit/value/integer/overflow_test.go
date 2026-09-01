package integer_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func assertBigIntDivisionUsesRuntime(
	t *testing.T,
	emission emit.ProgramEmission,
) {
	t.Helper()
	function := targetFunction(
		t,
		integerFamilySourceFile(t, emission),
		"BigSigned",
	)
	returned := function.Body().(tsgo.Block).
		Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.ArrayLiteralExpression).
		Elements()
	for index, expected := range []string{
		"goIntegerDivide",
		"goIntegerRemainder",
	} {
		normalization, ok := returned[index].(tsgo.CallExpression)
		if !ok || len(normalization.Arguments()) != 1 {
			t.Fatalf(
				"BigSigned result %d = %T, want fixed-width normalization",
				index,
				returned[index],
			)
		}
		call, ok := normalization.Arguments()[0].(tsgo.CallExpression)
		if !ok {
			t.Fatalf(
				"BigSigned result %d operand = %T, want %s runtime call",
				index,
				normalization.Arguments()[0],
				expected,
			)
		}
		callee, calleeOK := call.Expression().(tsgo.Identifier)
		if !calleeOK || callee.Text() != expected {
			t.Fatalf(
				"BigSigned result %d = %T, want %s runtime call",
				index,
				call.Expression(),
				expected,
			)
		}
	}
}

func TestIntegerBigIntCarrierProfilesWrapFixedWidthOperationsDifferentially(t *testing.T) {
	loaded := loadIntegerFamily(t)
	for _, test := range []struct {
		name           string
		representation emit.IntegerRepresentation
		nativeAlias    string
	}{
		{"fixed64-bigint", emit.IntegerRepresentationFixed64BigInt, "export type int = number;"},
		{"bigint", emit.IntegerRepresentationBigInt, "export type int = $go$core$int64;"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := integerOptions(test.representation)
			emission := compileIntegerFamily(
				t,
				loaded,
				options,
				"BigOverflowBinary",
				"BigOverflowUpdate",
				"NativeInt",
				"WideHash",
			)
			printed := printIntegerFamily(t, emission)
			for _, required := range []string{
				"goUint64(",
				"goInt64(",
				test.nativeAlias,
			} {
				if !strings.Contains(printed, required) {
					t.Fatalf("fixed-width artifact lacks %q:\n%s", required, printed)
				}
			}
			for _, definition := range []string{
				"export function goInt64(",
				"export function goUint64(",
			} {
				if count := strings.Count(printed, definition); count != 1 {
					t.Fatalf("fixed-width artifact has %d %q definitions, want one", count, definition)
				}
			}

			workingDirectory := t.TempDir()
			goOutput := executeIntegerOverflowGo(t, workingDirectory)
			targetOutput := executeIntegerOverflowTS(t, emission, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf("fixed-width TypeScript output = %q, Go output = %q", targetOutput, goOutput)
			}
		})
	}
}

func executeIntegerOverflowTS(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeIntegerFamily(t, emission, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import * as values from "`+
		artifacts.module(t, "source.ts")+`";

const row = (values: readonly bigint[]): string => values.map(String).join(" ");
console.log(row(values.BigOverflowBinary(18446744073709551615n, -9223372036854775808n)));
console.log(row(values.BigOverflowUpdate(18446744073709551615n)));
console.log(values.WideHash("a").toString(), values.WideHash("b").toString(), values.WideHash("cache-key").toString());
`)
	return executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
}

func executeIntegerOverflowGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(integerFamilyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "overflow-go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/overflowrunner

go 1.26.4

require example.com/integerfamily v0.0.0

replace example.com/integerfamily => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/integerfamily"
)

func main() {
	fmt.Println(values.BigOverflowBinary(^uint64(0), -1<<63))
	fmt.Println(values.BigOverflowUpdate(^uint64(0)))
	fmt.Println(values.WideHash("a"), values.WideHash("b"), values.WideHash("cache-key"))
}
`)
	return run(t, runnerDirectory, "go", "run", ".")
}

func TestIntegerNumberProfileRecordsWideHashBoundary(t *testing.T) {
	loaded := loadIntegerFamily(t)
	emission := compileIntegerFamily(
		t,
		loaded,
		integerOptions(emit.IntegerRepresentationNumber),
		"WideHash",
	)
	workingDirectory := t.TempDir()
	goOutput := executeWideHashGo(t, workingDirectory)
	targetOutput := executeWideHashTS(t, emission, workingDirectory)
	if targetOutput == goOutput {
		t.Fatalf("number profile unexpectedly claimed exact wide hash %q", targetOutput)
	}
}

func executeWideHashTS(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) string {
	t.Helper()
	artifacts := materializeIntegerFamily(t, emission, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import * as values from "`+
		artifacts.module(t, "source.ts")+`";

console.log(values.WideHash("a"), values.WideHash("b"), values.WideHash("cache-key"));
`)
	return executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
}

func executeWideHashGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(integerFamilyDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "hash-go-runner")
	if err := os.MkdirAll(runnerDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/hashrunner

go 1.26.4

require example.com/integerfamily v0.0.0

replace example.com/integerfamily => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/integerfamily"
)

func main() {
	fmt.Println(values.WideHash("a"), values.WideHash("b"), values.WideHash("cache-key"))
}
`)
	return run(t, runnerDirectory, "go", "run", ".")
}
