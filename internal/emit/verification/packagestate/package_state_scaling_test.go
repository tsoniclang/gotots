package packagestate_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type packageStateScale struct {
	fields              int
	assemblyAssignments int
	sourceVariables     int
	stateConstructors   int
	wireBytes           int
	printedBytes        int
}

func TestPackageStateConstructionScalesWithVariables(t *testing.T) {
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})

	counts := []int{32, 64, 128}
	measurements := make([]packageStateScale, 0, len(counts))
	for _, count := range counts {
		measurement := measurePackageStateScale(t, client, count)
		t.Logf(
			"variables=%d fields=%d assignments=%d wire=%dB printed=%dB",
			count,
			measurement.fields,
			measurement.assemblyAssignments,
			measurement.wireBytes,
			measurement.printedBytes,
		)
		if measurement.fields != count {
			t.Fatalf(
				"%d variables emitted %d state fields",
				count,
				measurement.fields,
			)
		}
		if measurement.assemblyAssignments != count*2 {
			t.Fatalf(
				"%d variables emitted %d zero/initializer assignments",
				count,
				measurement.assemblyAssignments,
			)
		}
		if measurement.sourceVariables != 0 {
			t.Fatalf(
				"%d package variables leaked into source modules",
				measurement.sourceVariables,
			)
		}
		if measurement.stateConstructors != 1 {
			t.Fatalf(
				"state constructors = %d, want one",
				measurement.stateConstructors,
			)
		}
		measurements = append(measurements, measurement)
	}
	assertLinearGrowth(t, "encoded TS-Go bytes", measurements, func(
		value packageStateScale,
	) int {
		return value.wireBytes
	})
	assertLinearGrowth(t, "printed TypeScript bytes", measurements, func(
		value packageStateScale,
	) int {
		return value.printedBytes
	})
}

func TestBlankOnlyInitializationHasAssemblyWithoutState(t *testing.T) {
	projectDirectory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "go.mod"),
		"module example.com/blank-only\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "sample", "sample.go"),
		`package sample

var _ = touch()

func touch() int32 { return 1 }
func init() { touch() }
func Run() int32 { return 2 }
`,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./sample",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Run"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	var assemblies int
	var programs int
	var programCalls int
	var states int
	var initFunctions int
	for _, file := range emission.Files() {
		switch file.Kind() {
		case emit.TargetFilePackageAssembly:
			assemblies++
		case emit.TargetFileProgramInitialization:
			programs++
			for _, statement := range file.SourceFile().Statements() {
				if _, ok := statement.(tsgo.ExpressionStatement); ok {
					programCalls++
				}
			}
		case emit.TargetFilePackageState:
			states++
		case emit.TargetFileSource:
			for _, statement := range file.SourceFile().Statements() {
				function, ok := statement.(tsgo.FunctionDeclaration)
				if ok && function.Name().Text() == "init" {
					initFunctions++
				}
			}
		}
	}
	if assemblies != 1 ||
		programs != 1 ||
		programCalls != 1 ||
		states != 0 ||
		initFunctions != 1 {
		t.Fatalf(
			"assembly/program/calls/state/init files = %d/%d/%d/%d/%d, want 1/1/1/0/1",
			assemblies,
			programs,
			programCalls,
			states,
			initFunctions,
		)
	}
}

func measurePackageStateScale(
	t *testing.T,
	client *tsgo.Client,
	count int,
) packageStateScale {
	t.Helper()
	projectDirectory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "go.mod"),
		"module example.com/package-state-scaling\n\ngo 1.26.4\n",
	)
	var source strings.Builder
	source.WriteString("package sample\n\n")
	for index := range count {
		fmt.Fprintf(&source, "var v%03d int32 = 1\n", index)
	}
	source.WriteString("\nfunc Run() int32 { return v000 }\n")
	writeProgramFile(
		t,
		filepath.Join(projectDirectory, "sample", "sample.go"),
		source.String(),
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./sample",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := emit.NewRoot(
		program.Roots()[0].Types().Scope().Lookup("Run"),
	)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(emission.Files()) != 5 {
		t.Fatalf(
			"target files = %d, want source/state/assembly/program/support",
			len(emission.Files()),
		)
	}
	var result packageStateScale
	for _, file := range emission.Files() {
		encoded, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		result.wireBytes += len(encoded)
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		result.printedBytes += len(printed)
		for _, statement := range file.SourceFile().Statements() {
			switch file.Kind() {
			case emit.TargetFileSource:
				if _, ok := statement.(tsgo.VariableStatement); ok {
					result.sourceVariables++
				}
			case emit.TargetFilePackageAssembly:
				function, ok := statement.(tsgo.FunctionDeclaration)
				if !ok || function.Name().Text() != "$initialize" {
					continue
				}
				for _, bodyStatement := range function.Body().(tsgo.Block).Statements() {
					if _, ok := bodyStatement.(tsgo.ExpressionStatement); ok {
						result.assemblyAssignments++
					}
				}
			case emit.TargetFilePackageState:
				inspectPackageStateStatement(t, statement, &result)
			}
		}
	}
	return result
}

func inspectPackageStateStatement(
	t *testing.T,
	statement tsgo.Statement,
	result *packageStateScale,
) {
	t.Helper()
	switch statement := statement.(type) {
	case tsgo.ClassDeclaration:
		for _, member := range statement.Members() {
			if _, ok := member.(tsgo.PropertyDeclaration); !ok {
				t.Fatalf("package state member is %T, want property", member)
			}
			result.fields++
		}
	case tsgo.VariableStatement:
		for _, declaration := range statement.DeclarationList().Declarations() {
			name, ok := declaration.Name().(tsgo.Identifier)
			if !ok || name.Text() != "$state" {
				continue
			}
			if _, ok := declaration.Initializer().(tsgo.NewExpression); !ok {
				t.Fatalf(
					"package state initializer is %T, want new expression",
					declaration.Initializer(),
				)
			}
			result.stateConstructors++
		}
	}
}

func assertLinearGrowth(
	t *testing.T,
	label string,
	measurements []packageStateScale,
	selectValue func(packageStateScale) int,
) {
	t.Helper()
	firstDelta := selectValue(measurements[1]) - selectValue(measurements[0])
	secondDelta := selectValue(measurements[2]) - selectValue(measurements[1])
	if firstDelta <= 0 ||
		secondDelta*10 < firstDelta*19 ||
		secondDelta*10 > firstDelta*21 {
		t.Fatalf(
			"%s deltas = %d, %d; want doubling within 5%%",
			label,
			firstDelta,
			secondDelta,
		)
	}
}

func TestProgramInitializationMatchesGoGlobalPackageOrder(t *testing.T) {
	projectDirectory := filepath.Join(
		repositoryRoot(),
		"testdata",
		"projects",
		"package-global-order",
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	assertGlobalInitializationArtifact(t, emission)
	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var assemblyPath string
	var targetPaths []string
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if file.Kind() == emit.TargetFileProgramInitialization {
			expected, err := os.ReadFile(
				filepath.Join(projectDirectory, "expected-program.ts"),
			)
			if err != nil {
				t.Fatal(err)
			}
			if printed != string(expected) {
				t.Fatalf(
					"program initialization TypeScript:\n%s\nwant:\n%s",
					printed,
					expected,
				)
			}
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeProgramFile(t, targetPath, printed)
		targetPaths = append(targetPaths, targetPath)
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "api" {
			assemblyPath = file.OutputPath()
		}
	}
	if assemblyPath == "" {
		t.Fatal("api package assembly is absent")
	}
	goOutput := executeGlobalOrderGo(t, projectDirectory, workingDirectory)
	targetOutput := executeGlobalOrderTypeScript(
		t,
		workingDirectory,
		targetPaths,
		assemblyPath,
	)
	if goOutput != "1234\n" || targetOutput != goOutput {
		t.Fatalf(
			"TypeScript/Go global initialization = %q/%q, want 1234",
			targetOutput,
			goOutput,
		)
	}
}

func assertGlobalInitializationArtifact(
	t *testing.T,
	emission emit.ProgramEmission,
) {
	t.Helper()
	expected := []string{
		"$initialize__registry",
		"$initialize____u3c0_",
		"$initialize__b",
		"$initialize____u3c0___package_1",
		"$initialize__a",
		"$initialize__api",
	}
	var actual []string
	programs := 0
	for _, file := range emission.Files() {
		switch file.Kind() {
		case emit.TargetFilePackageAssembly:
			for _, statement := range file.SourceFile().Statements() {
				if _, effect := statement.(tsgo.ExpressionStatement); effect {
					t.Fatalf(
						"passive package assembly %s has a top-level effect",
						file.OutputPath(),
					)
				}
			}
		case emit.TargetFileProgramInitialization:
			programs++
			for _, statement := range file.SourceFile().Statements() {
				expression, ok := statement.(tsgo.ExpressionStatement)
				if !ok {
					continue
				}
				call, ok := expression.Expression().(tsgo.CallExpression)
				if !ok {
					t.Fatalf("program effect is %T, want direct call", expression.Expression())
				}
				identifier, ok := call.Expression().(tsgo.Identifier)
				if !ok {
					t.Fatalf("program call target is %T, want identifier", call.Expression())
				}
				actual = append(actual, identifier.Text())
			}
		}
	}
	if programs != 1 {
		t.Fatalf("program initialization artifacts = %d, want one", programs)
	}
	if err := verifyInitializationSequence(expected, actual); err != nil {
		t.Fatal(err)
	}
	for name, mutated := range map[string][]string{
		"omitted":    actual[:len(actual)-1],
		"duplicated": append(append([]string{}, actual...), actual[len(actual)-1]),
		"reordered": {
			actual[0],
			actual[2],
			actual[1],
			actual[3],
			actual[4],
			actual[5],
		},
	} {
		if err := verifyInitializationSequence(expected, mutated); err == nil {
			t.Fatalf("%s program-call mutation passed the exact order join", name)
		}
	}
}

func verifyInitializationSequence(expected []string, actual []string) error {
	if len(actual) != len(expected) {
		return fmt.Errorf(
			"program initialization calls = %v, want %v",
			actual,
			expected,
		)
	}
	for index := range expected {
		if actual[index] != expected[index] {
			return fmt.Errorf(
				"program initialization call %d = %q, want %q",
				index,
				actual[index],
				expected[index],
			)
		}
	}
	return nil
}

func executeGlobalOrderGo(
	t *testing.T,
	projectDirectory string,
	workingDirectory string,
) string {
	t.Helper()
	absoluteProject, err := filepath.Abs(projectDirectory)
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-global-order")
	writeProgramFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/global-order-runner

go 1.26.4

require example.com/package-global-order v0.0.0

replace example.com/package-global-order => %s
`, filepath.ToSlash(absoluteProject)))
	writeProgramFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	"example.com/package-global-order/api"
)

func main() {
	fmt.Println(api.Run())
}
`)
	return runProgram(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func executeGlobalOrderTypeScript(
	t *testing.T,
	workingDirectory string,
	targetPaths []string,
	assemblyPath string,
) string {
	t.Helper()
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	modulePath := "./" + strings.TrimSuffix(assemblyPath, ".ts") + ".js"
	writeProgramFile(t, runnerPath, `import "./program.js";
import { Run } from "`+modulePath+`";

console.log(Run());
`)
	outputDirectory := filepath.Join(workingDirectory, "out")
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", outputDirectory,
	}
	arguments = append(arguments, targetPaths...)
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
	return runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(outputDirectory, "runner.js"),
	)
}
