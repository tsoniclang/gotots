package defined_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type printedDefined struct {
	paths        []string
	sourceModule string
	printed      string
}

func printDefined(
	t *testing.T,
	workingDirectory string,
	emission emit.ProgramEmission,
) printedDefined {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	var result printedDefined
	var printed strings.Builder
	for _, file := range emission.Files() {
		text, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		printed.WriteString(text)
		path := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeDefinedFile(t, path, text)
		result.paths = append(result.paths, path)
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "definedbasic" {
			result.sourceModule = "./" +
				strings.TrimSuffix(file.OutputPath(), ".ts") +
				".js"
		}
	}
	if result.sourceModule == "" {
		t.Fatal("defined-basic source module is absent")
	}
	result.printed = printed.String()
	return result
}

func runDefinedGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	modulePath, err := filepath.Abs(definedFixtureDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeDefinedFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/definedbasic v0.0.0

replace example.com/definedbasic => `+filepath.ToSlash(modulePath)+`
`)
	writeDefinedFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/definedbasic"
)

func main() {
	count := values.CountFromInt(7)
	other := values.OtherFromCount(count)
	fmt.Println(values.IntFromCount(values.CountFromOther(other)))
	fmt.Println(values.IntFromCount(values.AliasIdentity(count)))
	fmt.Println(values.IntFromCount(values.CountZero()))
	a, b, c := values.CountArithmetic(count, values.CountFromInt(3))
	fmt.Println(values.IntFromCount(a), values.IntFromCount(b), values.IntFromCount(c))
	a, b, c, d := values.CountBits(count, values.CountFromInt(3))
	fmt.Println(values.IntFromCount(a), values.IntFromCount(b), values.IntFromCount(c), values.IntFromCount(d))
	a, b, c = values.CountUnary(count)
	fmt.Println(values.IntFromCount(a), values.IntFromCount(b), values.IntFromCount(c))
	fmt.Println(values.CountOrder(count, values.CountFromInt(3)))
	left := values.LabelFromString("a")
	right := values.LabelFromString("z")
	fmt.Println(values.StringFromLabel(values.LabelJoin(left, right)))
	fmt.Println(values.LabelOrder(left, right))
	fmt.Println(values.BoolFromSwitch(values.SwitchNot(values.SwitchFromBool(true))))
	ratio := values.RatioFromFloat(7.5)
	r1, r2, r3, r4 := values.RatioArithmetic(ratio, values.RatioFromFloat(2))
	fmt.Println(values.FloatFromRatio(r1), values.FloatFromRatio(r2), values.FloatFromRatio(r3), values.FloatFromRatio(r4))
	narrow := values.NarrowAdd(values.NarrowFromFloat(0.1), values.NarrowFromFloat(0.2))
	fmt.Println(values.FloatFromNarrow(narrow))
	signal := values.SignalFromParts(1, 2)
	product := values.SignalProduct(signal, values.SignalFromParts(3, -4))
	fmt.Println(real(values.ComplexFromSignal(product)), imag(values.ComplexFromSignal(product)))
	fmt.Println(values.SignalEqual(signal, signal))
	fmt.Println(values.IntFromCount(values.ConstantValue()))
	fmt.Println(values.IntFromCount(values.UntypedConstantValue()))
	fmt.Println(values.IntFromCount(values.CountWithLiteral(values.CountFromInt(5))))
	fmt.Println(values.IntFromCount(values.FoldedCount()))
	fmt.Println(values.LocalTypes(6))
	fmt.Println(values.IntFromCount(values.CountUpdate(values.CountFromInt(4))))
	minimum, maximum, length := values.DefinedBuiltins(
		values.CountFromInt(9),
		values.CountFromInt(4),
		values.LabelFromString("hello"),
	)
	fmt.Println(values.IntFromCount(minimum), values.IntFromCount(maximum), length)
	fmt.Println(values.IntFromCount(values.CountPointer(values.CountFromInt(8))))
	fmt.Println(values.CountSwitch(values.CountFromInt(1)))
	fmt.Println(values.CountSwitch(values.CountFromInt(3)))
	fmt.Println(values.CountSwitch(values.CountFromInt(5)))
	original, copied, equal := values.CountArrayValues()
	fmt.Println(
		values.IntFromCount(original[0]),
		values.IntFromCount(copied[0]),
		equal,
	)
	slice := values.CountSliceValues()
	fmt.Println(
		values.IntFromCount(slice[0]),
		values.IntFromCount(slice[1]),
		values.IntFromCount(slice[2]),
	)
	found, missing, ok := values.CountMapValues()
	fmt.Println(
		values.StringFromLabel(found),
		values.StringFromLabel(missing),
		ok,
	)
}
`)
	return runDefinedCommand(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func runDefinedTypeScript(
	t *testing.T,
	workingDirectory string,
	artifacts printedDefined,
	suffix string,
) string {
	t.Helper()
	runner := `import * as values from "` + artifacts.sourceModule + `";
const count = values.CountFromInt(7` + suffix + `);
const other = values.OtherFromCount(count);
console.log(String(values.IntFromCount(values.CountFromOther(other))));
console.log(String(values.IntFromCount(values.AliasIdentity(count))));
console.log(String(values.IntFromCount(values.CountZero())));
console.log(values.CountArithmetic(count, values.CountFromInt(3` + suffix + `)).map(value => String(values.IntFromCount(value))).join(" "));
console.log(values.CountBits(count, values.CountFromInt(3` + suffix + `)).map(value => String(values.IntFromCount(value))).join(" "));
console.log(values.CountUnary(count).map(value => String(values.IntFromCount(value))).join(" "));
console.log(values.CountOrder(count, values.CountFromInt(3` + suffix + `)).join(" "));
const left = values.LabelFromString("a");
const right = values.LabelFromString("z");
console.log(values.StringFromLabel(values.LabelJoin(left, right)));
console.log(values.LabelOrder(left, right).join(" "));
console.log(String(values.BoolFromSwitch(values.SwitchNot(values.SwitchFromBool(true)))));
const ratio = values.RatioFromFloat(7.5);
console.log(values.RatioArithmetic(ratio, values.RatioFromFloat(2)).map(value => String(values.FloatFromRatio(value))).join(" "));
const narrow = values.NarrowAdd(values.NarrowFromFloat(0.1), values.NarrowFromFloat(0.2));
console.log(String(values.FloatFromNarrow(narrow)));
const signal = values.SignalFromParts(1, 2);
const product = values.SignalProduct(signal, values.SignalFromParts(3, -4));
const complex = values.ComplexFromSignal(product);
console.log(String(complex.real), String(complex.imag));
console.log(String(values.SignalEqual(signal, signal)));
console.log(String(values.IntFromCount(values.ConstantValue())));
console.log(String(values.IntFromCount(values.UntypedConstantValue())));
console.log(String(values.IntFromCount(values.CountWithLiteral(values.CountFromInt(5` + suffix + `)))));
console.log(String(values.IntFromCount(values.FoldedCount())));
console.log(String(values.LocalTypes(6` + suffix + `)));
console.log(String(values.IntFromCount(values.CountUpdate(values.CountFromInt(4` + suffix + `)))));
const [minimum, maximum, length] = values.DefinedBuiltins(
  values.CountFromInt(9` + suffix + `),
  values.CountFromInt(4` + suffix + `),
  values.LabelFromString("hello"),
);
console.log(String(values.IntFromCount(minimum)), String(values.IntFromCount(maximum)), String(length));
console.log(String(values.IntFromCount(values.CountPointer(values.CountFromInt(8` + suffix + `)))));
console.log(String(values.CountSwitch(values.CountFromInt(1` + suffix + `))));
console.log(String(values.CountSwitch(values.CountFromInt(3` + suffix + `))));
console.log(String(values.CountSwitch(values.CountFromInt(5` + suffix + `))));
const [original, copied, equal] = values.CountArrayValues();
console.log(
  String(values.IntFromCount(original.get(0` + suffix + `))),
  String(values.IntFromCount(copied.get(0` + suffix + `))),
  String(equal),
);
const slice = values.CountSliceValues();
console.log(
  String(values.IntFromCount(slice.get(0` + suffix + `))),
  String(values.IntFromCount(slice.get(1` + suffix + `))),
  String(values.IntFromCount(slice.get(2` + suffix + `))),
);
const [found, missing, ok] = values.CountMapValues();
console.log(values.StringFromLabel(found), values.StringFromLabel(missing), String(ok));
`
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeDefinedFile(t, runnerPath, runner)
	writeDefinedFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatalf("defined-basic program failed strict typecheck: %v", err)
	}
	return runDefinedCommand(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}

func definedFixtureDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"type",
		"defined-basic",
	)
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve defined-basic repository root")
	}
	return filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "..", ".."),
	)
}

func runDefinedCommand(
	t *testing.T,
	directory string,
	name string,
	arguments ...string,
) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf(
			"%s %s: %v\n%s",
			name,
			strings.Join(arguments, " "),
			err,
			output,
		)
	}
	return string(output)
}

func writeDefinedFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
