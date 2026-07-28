package arrayvalue_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDefinedArrayZeroCopyConversionIndexAndAddressMatchGo(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
		suffix  string
	}{
		{name: "number", options: emit.DefaultOptions()},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
			suffix: "n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			emission := compileDefinedArrayFixture(t, testCase.options)
			workingDirectory := t.TempDir()
			target := materializeArrayProgram(t, workingDirectory, emission)
			source := target.printed[sourceOutputPath(target)]
			for _, forbidden := range []string{
				": any",
				": unknown",
				".call(",
				".apply(",
				".bind(",
			} {
				if strings.Contains(source, forbidden) {
					t.Fatalf("defined-array artifact contains %q:\n%s", forbidden, source)
				}
			}
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeFile(t, runner, definedArrayRunner(target, testCase.suffix))
			writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
			target.paths = append(target.paths, runner)
			if err := compileTypeScript(t, workingDirectory, target.paths); err != nil {
				t.Fatal(err)
			}
			typeScriptOutput := run(
				t,
				workingDirectory,
				"node",
				filepath.Join(workingDirectory, "out", "runner.js"),
			)
			goOutput := runDefinedArrayGo(t, workingDirectory)
			if typeScriptOutput != goOutput {
				t.Fatalf(
					"TypeScript output differs from Go\nTypeScript:\n%s\nGo:\n%s\nArtifact:\n%s",
					typeScriptOutput,
					goOutput,
					source,
				)
			}
		})
	}
}

func TestDefinedArrayHasOneMinimalNominalTargetClass(t *testing.T) {
	emission := compileDefinedArrayFixture(t, emit.DefaultOptions())
	classes := map[string]tsgo.ClassDeclaration{}
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			class, ok := statement.(tsgo.ClassDeclaration)
			if ok {
				classes[class.Name().Text()] = class
			}
		}
	}
	pair := classes["Pair"]
	if pair == nil || len(pair.Members()) != 4 {
		t.Fatalf(
			"addressed Pair members = %d, want brand, constructor, get, set",
			classMemberCount(pair),
		)
	}
	methods := map[string]bool{}
	for _, member := range pair.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if ok {
			methods[method.Name().(tsgo.Identifier).Text()] = true
		}
	}
	if !methods["get"] || !methods["set"] {
		t.Fatalf("addressed Pair methods = %v", methods)
	}
	other := classes["Other"]
	if other == nil || len(other.Members()) != 2 {
		t.Fatalf(
			"unaddressed Other members = %d, want brand and constructor only",
			classMemberCount(other),
		)
	}
}

func classMemberCount(class tsgo.ClassDeclaration) int {
	if class == nil {
		return 0
	}
	return len(class.Members())
}

func TestDefinedArrayNominalityRejectsSameUnderlyingAssignment(t *testing.T) {
	workingDirectory := t.TempDir()
	target := materializeArrayProgram(
		t,
		workingDirectory,
		compileDefinedArrayFixture(t, emit.DefaultOptions()),
	)
	mutation := filepath.Join(workingDirectory, "nominality.ts")
	writeFile(t, mutation, `import { ElementFromInt, NewPair, Other } from "`+target.sourceModule+`";
const pair = NewPair(ElementFromInt(1), ElementFromInt(2));
const other: Other = pair;
void other;
`)
	writeFile(t, filepath.Join(workingDirectory, "package.json"), "{\"type\":\"module\"}\n")
	target.paths = append(target.paths, mutation)
	if err := compileTypeScript(t, workingDirectory, target.paths); err == nil {
		t.Fatal("same-underlying defined arrays became assignable")
	}
	replacements := 0
	for _, path := range target.paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mutated := strings.ReplaceAll(
			string(content),
			"declare private readonly $goType: void;\n",
			"",
		)
		if mutated == string(content) {
			continue
		}
		replacements++
		writeFile(t, path, mutated)
	}
	if replacements == 0 {
		t.Fatal("nominality mutation removed no generated brand")
	}
	if err := compileTypeScript(t, workingDirectory, target.paths); err != nil {
		t.Fatalf("brand-removal foil did not expose structural identity: %v", err)
	}
}

func compileDefinedArrayFixture(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: definedArrayDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func definedArrayRunner(
	target materializedProgram,
	suffix string,
) string {
	return `import "` + target.programInit + `";
import * as values from "` + target.sourceModule + `";

const element = values.ElementFromInt;
const show = (value: values.Pair) =>
    values.PairValues(value).map(item => String(values.IntFromElement(item))).join(" ");
const pair = values.NewPair(element(3` + suffix + `), element(4` + suffix + `));
console.log(show(pair));
console.log(show(values.ZeroPair()));
const [original, copied] = values.CopyPair(pair);
console.log(show(original), show(copied));
console.log(show(values.ConvertRaw(values.ConvertPair(pair))));
console.log(show(values.AliasIdentity(pair)));
const [isolated, stored, compounded, pointed, unchanged] = values.CallIsolation();
console.log(show(isolated), show(stored));
console.log(show(compounded));
console.log(unchanged);
console.log(String(values.Length(pair)));
console.log(show(pointed));
console.log(values.ConvertOther(pair) instanceof values.Other);
const [rawOriginal, pairConverted, pairOriginal, rawConverted] = values.ConversionIsolation();
console.log(show(rawOriginal), show(pairConverted));
console.log(
    show(pairOriginal),
    values.RawValues(rawConverted).map(item => String(values.IntFromElement(item))).join(" "),
);
`
}

func runDefinedArrayGo(t *testing.T, workingDirectory string) string {
	t.Helper()
	runnerDirectory := filepath.Join(workingDirectory, "go-runner-defined-array")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), `module example.com/runner

go 1.26.4

require example.com/definedarray v0.0.0

replace example.com/definedarray => `+filepath.ToSlash(definedArrayDirectory())+"\n")
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"
	values "example.com/definedarray"
)

func show(value values.Pair) string {
	left, right := values.PairValues(value)
	return fmt.Sprint(values.IntFromElement(left), " ", values.IntFromElement(right))
}

func main() {
	element := values.ElementFromInt
	pair := values.NewPair(element(3), element(4))
	fmt.Println(show(pair))
	fmt.Println(show(values.ZeroPair()))
	original, copied := values.CopyPair(pair)
	fmt.Println(show(original), show(copied))
	fmt.Println(show(values.ConvertRaw(values.ConvertPair(pair))))
	fmt.Println(show(values.AliasIdentity(pair)))
	isolated, stored, compounded, pointed, unchanged := values.CallIsolation()
	fmt.Println(show(isolated), show(stored))
	fmt.Println(show(compounded))
	fmt.Println(unchanged)
	fmt.Println(values.Length(pair))
	fmt.Println(show(pointed))
	fmt.Println(values.ConvertOther(pair) == values.Other{element(3), element(4)})
	rawOriginal, pairConverted, pairOriginal, rawConverted := values.ConversionIsolation()
	fmt.Println(show(rawOriginal), show(pairConverted))
	rawLeft, rawRight := values.RawValues(rawConverted)
	fmt.Println(
		show(pairOriginal),
		values.IntFromElement(rawLeft),
		values.IntFromElement(rawRight),
	)
}
`)
	return run(t, runnerDirectory, "go", "run", ".")
}

func definedArrayDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"type",
		"defined-array",
	)
}
