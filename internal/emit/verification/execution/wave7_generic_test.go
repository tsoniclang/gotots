package emit_test

import (
	"context"
	"go/ast"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestWaveSevenGenericFoundationCompilesThroughPublicPipeline(t *testing.T) {
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
				Directory: waveSevenGenericDirectory(),
				Pattern:   ".",
			})
			if err != nil {
				t.Fatal(err)
			}
			root, err := emit.NewRoot(
				program.Roots()[0].Types().Scope().Lookup("AuditFunctions"),
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
			artifacts := materializeArtifacts(
				t,
				emission,
				workingDirectory,
			)
			assertWaveSevenGenericFoundationShape(t, artifacts.printed)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { AuditFunctions } from "`+artifacts.sourceModule+`";

const values = AuditFunctions();
const output: string[] = [];
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
}
console.log(output.join(" "));
`)
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			paths := append(artifacts.paths, runner)
			waveThreeTypecheck(t, workingDirectory, paths)
			goOutput := executeWaveSevenGenericGo(
				t,
				workingDirectory,
				"AuditFunctions",
			)
			requireNativeGoEvidence(t, goOutput)
		})
	}
}

func TestInferredGenericFunctionValueUsesIdentifierInstanceEvidence(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveSevenGenericDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	sourcePackage := program.Roots()[0]
	sourceFunction := waveFourFunction(t, sourcePackage, "InferredGenericFunctionValue")
	var inferred *ast.Ident
	ast.Inspect(sourceFunction.Body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && identifier.Name == "Equal" {
			inferred = identifier
		}
		return true
	})
	if inferred == nil {
		t.Fatal("inferred generic function identifier is absent")
	}
	if _, instantiated := sourcePackage.TypesInfo().Instances[inferred]; !instantiated {
		t.Fatal("go/types did not record the inferred identifier instance")
	}
	root, err := emit.NewRoot(sourcePackage.Types().Scope().Lookup("AuditFunctions"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	var concreteNames []string
	for _, name := range []string{"InferredGenericFunctionValue", "ExplicitGenericFunctionValue"} {
		wrapper := genericFunctionValueWrapper(t, waveFourTargetFunction(t, emission, name))
		block, ok := wrapper.Body().(tsgo.Block)
		if !ok || len(block.Statements()) != 1 {
			t.Fatalf("%s wrapper body = %T, want one-statement block", name, wrapper.Body())
		}
		returned, ok := block.Statements()[0].(tsgo.ReturnStatement)
		if !ok {
			t.Fatalf("%s wrapper statement = %T, want return", name, block.Statements()[0])
		}
		body, ok := returned.Expression().(tsgo.CallExpression)
		if !ok {
			t.Fatalf("%s wrapper return = %T, want call", name, returned.Expression())
		}
		if len(body.Arguments()) != len(wrapper.Parameters()) {
			t.Fatalf("%s wrapper changes source argument cardinality", name)
		}
		callee, ok := body.Expression().(tsgo.Identifier)
		if !ok || callee.Text() != "Equal$int32" {
			t.Fatalf("%s wrapper does not call one exact concretization", name)
		}
		concreteNames = append(concreteNames, callee.Text())
	}
	if concreteNames[0] != concreteNames[1] {
		t.Fatalf("explicit/inferred concretizations differ: %v", concreteNames)
	}
}

func genericFunctionValueWrapper(t *testing.T, function tsgo.FunctionDeclaration) tsgo.ArrowFunction {
	t.Helper()
	for _, statement := range function.Body().(tsgo.Block).Statements() {
		returned, ok := statement.(tsgo.ReturnStatement)
		if !ok {
			continue
		}
		call, ok := returned.Expression().(tsgo.CallExpression)
		if !ok {
			continue
		}
		for _, argument := range call.Arguments() {
			ordinary, ordinaryOK := argument.(tsgo.ArrowFunction)
			if ordinaryOK {
				return ordinary
			}
			registration, ok := argument.(tsgo.CallExpression)
			if !ok || len(registration.Arguments()) != 2 {
				continue
			}
			_, registrationOrdinary :=
				registration.Arguments()[0].(tsgo.ArrowFunction)
			_, registrationDeferred :=
				registration.Arguments()[1].(tsgo.ArrowFunction)
			if !registrationOrdinary || !registrationDeferred {
				continue
			}
			t.Fatalf(
				"%s non-recovering generic function value acquired a deferred registry entry",
				function.Name().Text(),
			)
		}
	}
	t.Fatalf("%s lacks a generic function-value wrapper", function.Name().Text())
	return nil
}

func TestWaveSevenGenericNamedTypesCompileThroughPublicPipeline(t *testing.T) {
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
				Directory: waveSevenGenericDirectory(),
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
			local := targetFunctionText(
				t,
				artifacts.printed,
				"LocalTypeCapability",
			)
			if !strings.Contains(
				local,
				"function GenericMapValue$Named_entry",
			) || strings.Contains(
				local,
				"export function GenericMapValue$Named_entry",
			) || !strings.Contains(local, "): GoMapValue<entry, int32> =>") {
				t.Fatalf(
					"local-type concretization is not lexical, unexported, and inline:\n%s",
					local,
				)
			}
			sourceModule := sourceModuleForExport(
				t,
				artifacts,
				workingDirectory,
				"Audit",
			)
			runner := filepath.Join(workingDirectory, "runner.ts")
			writeProgramFile(t, runner, `import "./program.js";
import { Audit } from "`+sourceModule+`";

const values = Audit();
const output: string[] = [];
for (let index = 0; index < values.length; index++) {
    output.push(String(values.get(index)));
}
console.log(output.join(" "));
`)
			writeProgramFile(
				t,
				filepath.Join(workingDirectory, "package.json"),
				"{\"type\":\"module\"}\n",
			)
			paths := append(artifacts.paths, runner)
			waveThreeTypecheck(t, workingDirectory, paths)
			goOutput := executeWaveSevenGenericGo(
				t,
				workingDirectory,
				"Audit",
			)
			requireNativeGoEvidence(t, goOutput)
		})
	}
}
