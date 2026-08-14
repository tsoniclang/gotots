package structvalue_test

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

const lexicalArtifactSource = `package boundary

var PackageValue = func() int32 {
	type Local int32
	item := struct{ Value Local }{Value: Local(3)}
	return int32(item.Value)
}()

var BlockValue = func() int32 {
	if true {
		type Local int32
		item := struct{ Value Local }{Value: Local(4)}
		return int32(item.Value)
	}
	return 0
}()

var _ = func() int32 {
	type Local int32
	item := struct{ Value Local }{Value: Local(6)}
	return int32(item.Value)
}()

func PackageInitializerResult() int32 { return PackageValue }
func BlockInitializerResult() int32 { return BlockValue }

func NestedLiteralResult() int32 {
	compute := func() int32 {
		type Local int32
		item := struct{ Value Local }{Value: Local(5)}
		return int32(item.Value)
	}
	return compute()
}

func GroupedAnchorResult() int32 {
	type (
		Before int32
		Local int32
		After int32
	)
	item := struct{ Value Local }{Value: Local(7)}
	return int32(Before(1)) + int32(item.Value) + int32(After(1))
}
`

func TestAnonymousStructLexicalPlacementAcrossSourceArtifacts(
	t *testing.T,
) {
	projectDirectory, emission := compileLexicalArtifactProgram(t)
	for _, file := range emission.Files() {
		if file.OutputPath() == "support/anonymous-structs.ts" {
			t.Fatal("local-component anonymous struct escaped lexical ownership")
		}
	}
	source := structTargetSource(t, emission)
	nested := targetFunctionByName(t, source, "NestedLiteralResult")
	inner := nestedFunctionExpression(t, nested)
	assertImmediateLexicalDefinitionPair(t, inner.Body().(tsgo.Block))
	assertGroupedLexicalAnchor(t, source)
	assertPackageInitializerLexicalPlacement(t, emission)

	workingDirectory := t.TempDir()
	targetPaths, module := materializeStructProgramWithGolden(
		t,
		workingDirectory,
		emission,
		false,
	)
	runner := filepath.Join(workingDirectory, "runner.ts")
	writeProgramFile(t, runner, `import "./program.js";
import {
	BlockInitializerResult,
	GroupedAnchorResult,
	NestedLiteralResult,
	PackageInitializerResult,
} from "`+module+`";
console.log(PackageInitializerResult());
console.log(BlockInitializerResult());
console.log(NestedLiteralResult());
console.log(GroupedAnchorResult());
`)
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	targetPaths = append(targetPaths, runner)
	compileStructTypeScript(t, workingDirectory, targetPaths)
	targetOutput := runProgram(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeLexicalArtifactGo(
		t,
		projectDirectory,
		workingDirectory,
	)
	if targetOutput != goOutput || goOutput != "3\n4\n5\n9\n" {
		t.Fatalf(
			"lexical artifact TypeScript/Go output = %q/%q",
			targetOutput,
			goOutput,
		)
	}
}

func TestAnonymousStructFunctionTopPlacementMutationFailsScopeGate(
	t *testing.T,
) {
	_, emission := compileLexicalArtifactProgram(t)
	mutatedPath, mutatedSource :=
		mutateNestedAnonymousClassToFunctionTop(t, emission)
	directory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	var paths []string
	for _, file := range emission.Files() {
		source := file.SourceFile()
		if file.OutputPath() == mutatedPath {
			source = mutatedSource
		}
		printed, err := client.PrintNode(source, tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, filepath.FromSlash(file.OutputPath()))
		writeProgramFile(t, path, printed)
		paths = append(paths, path)
	}
	writeProgramFile(
		t,
		filepath.Join(directory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	err = typecheckStructuralFiles(directory, paths)
	if err == nil || !strings.Contains(err.Error(), "Local") {
		t.Fatalf("function-top placement mutation did not fail local scope: %v", err)
	}
}

func compileLexicalArtifactProgram(
	t *testing.T,
) (string, emit.ProgramEmission) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/boundary\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(directory, "source.go"),
		lexicalArtifactSource,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
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
	return directory, emission
}

func nestedFunctionExpression(
	t *testing.T,
	outer tsgo.FunctionDeclaration,
) tsgo.ArrowFunction {
	t.Helper()
	body := outer.Body().(tsgo.Block).Statements()
	if len(body) == 0 {
		t.Fatal("outer function body is empty")
	}
	statement, ok := body[0].(tsgo.VariableStatement)
	if !ok {
		t.Fatalf("nested function owner = %T", body[0])
	}
	declarations := statement.DeclarationList().Declarations()
	if len(declarations) != 1 {
		t.Fatalf("nested function declarations = %d", len(declarations))
	}
	inner, ok := declarations[0].Initializer().(tsgo.ArrowFunction)
	if !ok {
		t.Fatalf("nested function initializer = %T", declarations[0].Initializer())
	}
	return inner
}

func assertImmediateLexicalDefinitionPair(t *testing.T, block tsgo.Block) {
	t.Helper()
	statements := block.Statements()
	if len(statements) < 2 {
		t.Fatalf("lexical block statements = %d", len(statements))
	}
	local, localOK := statements[0].(tsgo.TypeAliasDeclaration)
	anonymous, anonymousOK := statements[1].(tsgo.ClassDeclaration)
	if !localOK ||
		!anonymousOK ||
		strings.HasPrefix(local.Name().Text(), "$goStruct$") ||
		!strings.HasPrefix(anonymous.Name().Text(), "$goStruct$") {
		t.Fatalf(
			"lexical definition pair = %T/%T",
			statements[0],
			statements[1],
		)
	}
}

func assertGroupedLexicalAnchor(t *testing.T, source tsgo.SourceFile) {
	t.Helper()
	function := targetFunctionByName(t, source, "GroupedAnchorResult")
	statements := function.Body().(tsgo.Block).Statements()
	if len(statements) < 4 {
		t.Fatalf("grouped lexical statements = %d", len(statements))
	}
	before, beforeOK := statements[0].(tsgo.TypeAliasDeclaration)
	local, localOK := statements[1].(tsgo.TypeAliasDeclaration)
	anonymous, anonymousOK := statements[2].(tsgo.ClassDeclaration)
	after, afterOK := statements[3].(tsgo.TypeAliasDeclaration)
	if !beforeOK || !localOK || !anonymousOK || !afterOK {
		t.Fatalf(
			"grouped lexical declarations = %T/%T/%T/%T",
			statements[0], statements[1], statements[2], statements[3],
		)
	}
	names := []string{
		before.Name().Text(),
		local.Name().Text(),
		anonymous.Name().Text(),
		after.Name().Text(),
	}
	if !strings.HasPrefix(names[0], "Before") ||
		!strings.HasPrefix(names[1], "Local") ||
		!strings.HasPrefix(names[2], "$goStruct$") ||
		!strings.HasPrefix(names[3], "After") {
		t.Fatalf("grouped lexical order = %q", names)
	}
}

func assertPackageInitializerLexicalPlacement(
	t *testing.T,
	emission emit.ProgramEmission,
) {
	t.Helper()
	var initializers []tsgo.ArrowFunction
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFilePackageAssembly {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name().Text() != "$initialize" {
				continue
			}
			for _, candidate := range function.Body().(tsgo.Block).Statements() {
				initializers = append(
					initializers,
					initializerFunctionsInStatement(candidate)...,
				)
			}
		}
	}
	if len(initializers) != 3 {
		t.Fatalf("package initializer function literals = %d, want three", len(initializers))
	}
	assertImmediateLexicalDefinitionPair(
		t,
		initializers[0].Body().(tsgo.Block),
	)
	blockBody := initializers[1].Body().(tsgo.Block).Statements()
	if len(blockBody) == 0 {
		t.Fatal("nested-block package initializer is empty")
	}
	branch, ok := blockBody[0].(tsgo.IfStatement)
	if !ok {
		t.Fatalf("nested-block package initializer begins with %T", blockBody[0])
	}
	assertImmediateLexicalDefinitionPair(
		t,
		branch.ThenStatement().(tsgo.Block),
	)
	assertImmediateLexicalDefinitionPair(
		t,
		initializers[2].Body().(tsgo.Block),
	)
}

func initializerFunctionsInStatement(
	statement tsgo.Statement,
) []tsgo.ArrowFunction {
	switch statement := statement.(type) {
	case tsgo.ExpressionStatement:
		initializer := initializerFunctionExpression(statement.Expression())
		if initializer == nil {
			return nil
		}
		return []tsgo.ArrowFunction{initializer}
	case tsgo.Block:
		var result []tsgo.ArrowFunction
		for _, child := range statement.Statements() {
			result = append(result, initializerFunctionsInStatement(child)...)
		}
		return result
	default:
		return nil
	}
}

func initializerFunctionExpression(
	expression tsgo.Expression,
) tsgo.ArrowFunction {
	switch expression := expression.(type) {
	case tsgo.ArrowFunction:
		return expression
	case tsgo.BinaryExpression:
		return initializerFunctionExpression(expression.Right())
	case tsgo.CallExpression:
		return initializerFunctionExpression(expression.Expression())
	case tsgo.ParenthesizedExpression:
		return initializerFunctionExpression(expression.Expression())
	default:
		return nil
	}
}

func executeLexicalArtifactGo(
	t *testing.T,
	projectDirectory string,
	workingDirectory string,
) string {
	t.Helper()
	runner := filepath.Join(workingDirectory, "go-runner")
	writeProgramFile(t, filepath.Join(runner, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/boundary v0.0.0

replace example.com/boundary => %s
`, filepath.ToSlash(projectDirectory)))
	writeProgramFile(t, filepath.Join(runner, "main.go"), `package main

import (
	"fmt"
	boundary "example.com/boundary"
)

func main() {
	fmt.Println(boundary.PackageInitializerResult())
	fmt.Println(boundary.BlockInitializerResult())
	fmt.Println(boundary.NestedLiteralResult())
	fmt.Println(boundary.GroupedAnchorResult())
}
`)
	return runProgram(
		t,
		runner,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func mutateNestedAnonymousClassToFunctionTop(
	t *testing.T,
	emission emit.ProgramEmission,
) (string, tsgo.SourceFile) {
	t.Helper()
	factory := tsgo.NewFactory()
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		source := file.SourceFile()
		statements := source.Statements()
		for index, statement := range statements {
			outer, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || outer.Name().Text() != "NestedLiteralResult" {
				continue
			}
			outerBody := outer.Body().(tsgo.Block)
			outerStatements := outerBody.Statements()
			variable := outerStatements[0].(tsgo.VariableStatement)
			declaration := variable.DeclarationList().Declarations()[0]
			inner := declaration.Initializer().(tsgo.ArrowFunction)
			innerBody := inner.Body().(tsgo.Block)
			innerStatements := innerBody.Statements()
			anonymous := innerStatements[1].(tsgo.ClassDeclaration)
			inner = factory.ArrowFunction(
				inner.Modifiers(),
				inner.TypeParameters(),
				inner.Parameters(),
				inner.Type(),
				inner.EqualsGreaterThanToken(),
				factory.Block(
					append(
						[]tsgo.Statement{innerStatements[0]},
						innerStatements[2:]...,
					),
					innerBody.MultiLine(),
				),
			)
			declaration = factory.VariableDeclaration(
				declaration.Name(),
				declaration.ExclamationToken(),
				declaration.Type(),
				inner,
			)
			variable = factory.VariableStatement(
				variable.Modifiers(),
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{declaration},
					variable.DeclarationList().Flags(),
				),
			)
			outerStatements[0] = variable
			outerStatements = append(
				[]tsgo.Statement{anonymous},
				outerStatements...,
			)
			statements[index] = factory.FunctionDeclaration(
				outer.Modifiers(),
				outer.AsteriskToken(),
				outer.Name(),
				outer.TypeParameters(),
				outer.Parameters(),
				outer.Type(),
				factory.Block(outerStatements, outerBody.MultiLine()),
			)
			return file.OutputPath(), factory.SourceFile(
				statements,
				source.EndOfFileToken(),
				source.SourceData(),
			)
		}
	}
	t.Fatal("NestedLiteralResult target function is absent")
	return "", nil
}
