package maprepresentation_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestMutationNonNominalMapInferenceCannotReplaceCanonicalContract(
	t *testing.T,
) {
	emission := compileMapOwnership(t)
	source := targetFileBySuffix(
		t,
		emission.Files(),
		"source.ts",
	).SourceFile()
	for _, functionName := range []string{
		"ReassignAggregate",
		"ReassignScalar",
	} {
		function := targetFunction(t, source, functionName)
		declaration := targetLocalDeclaration(t, function, "values")
		mapType, ok := declaration.Type().(tsgo.TypeReferenceNode)
		if !ok {
			t.Fatalf(
				"%s values type = %T, want explicit GoMapValue",
				functionName,
				declaration.Type(),
			)
		}
		name, ok := mapType.TypeName().(tsgo.Identifier)
		if !ok ||
			name.Text() != "GoMapValue" ||
			len(mapType.TypeArguments()) != 2 {
			t.Fatalf(
				"%s values type = %#v, want GoMapValue<K,V>",
				functionName,
				mapType,
			)
		}
		if declaration.Initializer() == nil ||
			declaration.Initializer().Kind() !=
				tsgo.SyntaxKindCallExpression {
			t.Fatalf(
				"%s values initializer = %T, want concrete map zero",
				functionName,
				declaration.Initializer(),
			)
		}
	}
}

func TestMutationGenericMapTransferCannotBypassProjectionCapability(
	t *testing.T,
) {
	emission := compileMapOwnership(t)
	project := targetGeneratedFunction(t, emission.Files(), "Project$kernel")
	if len(project.TypeParameters()) != 3 ||
		project.TypeParameters()[0].Name().Text() != "M" ||
		project.TypeParameters()[0].Constraint() != nil {
		t.Fatalf(
			"Project type-parameter shape = %#v, want unconstrained target M,K,V",
			project.TypeParameters(),
		)
	}
	capabilityName := ""
	capabilityCount := 0
	for _, parameter := range project.Parameters() {
		name, ok := parameter.Name().(tsgo.Identifier)
		if !ok || !strings.HasPrefix(name.Text(), "$go$convert_") {
			continue
		}
		capabilityCount++
		functionType, ok := parameter.Type().(tsgo.FunctionTypeNode)
		if !ok ||
			len(functionType.Parameters()) == 0 ||
			!typeReferenceNamed(functionType.Parameters()[0].Type(), "M") ||
			!mapContractType(functionType.Type(), "K", "V") {
			t.Fatalf(
				"Project map projection capability type = %T, want M -> GoMapValue<K,V>",
				parameter.Type(),
			)
		}
		capabilityName = name.Text()
	}
	if capabilityCount != 1 || capabilityName == "" {
		t.Fatalf(
			"Project map projection capabilities = %d, want one exact identity",
			capabilityCount,
		)
	}
	statements := project.Body().(tsgo.Block).Statements()
	result := statements[len(statements)-1].(tsgo.ReturnStatement).
		Expression()
	call, ok := result.(tsgo.CallExpression)
	if !ok {
		t.Fatalf(
			"Project return = %T, want projection capability call",
			result,
		)
	}
	callee, ok := call.Expression().(tsgo.Identifier)
	if !ok || callee.Text() != capabilityName {
		t.Fatalf(
			"Project return callee = %T/%q, want %q",
			call.Expression(),
			identifierText(call.Expression()),
			capabilityName,
		)
	}
}

func TestOpenGenericMapExportFailsWithoutAnExactSourceABI(t *testing.T) {
	loaded, err := load.One(context.Background(), load.Request{
		Directory: mapOwnershipDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), roots)
	var schedule *emit.ScheduleError
	if !errors.As(err, &schedule) ||
		schedule.Object != "Project" ||
		schedule.Reason != "export root has no exact source-facing target binding" {
		t.Fatalf("open generic map export error = %#v", err)
	}
}

func TestMapRepresentationOwnershipStrictDifferential(t *testing.T) {
	emission := compileMapOwnership(t)
	workingDirectory := t.TempDir()
	artifacts := materialize(t, emission, workingDirectory)
	sourceArtifact := readFile(t, artifacts.file(t, "source.ts"))
	t.Logf("map ownership source bytes=%d", len(sourceArtifact))
	if count := strings.Count(
		sourceArtifact,
		"let values: GoMapValue<",
	); count != 2 {
		t.Fatalf(
			"canonical local map declarations = %d, want 2:\n%s",
			count,
			sourceArtifact,
		)
	}
	for _, required := range []string{
		"export function Project$kernel<M, K, V>($go$convert_",
		"return $go$convert_",
		"Project$concrete_",
	} {
		if !strings.Contains(sourceArtifact, required) {
			t.Fatalf(
				"map ownership artifact lacks %q:\n%s",
				required,
				sourceArtifact,
			)
		}
	}
	for _, forbidden := range []string{
		"export function Project<M, K, V>",
		"M extends GoMapValue",
		"let values = $goMap_",
	} {
		if strings.Contains(sourceArtifact, forbidden) {
			t.Fatalf(
				"map ownership artifact contains %q:\n%s",
				forbidden,
				sourceArtifact,
			)
		}
	}
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import {
    DeclarationLifecycle,
    ProjectionLifecycle,
} from "`+artifacts.module(t, "source.ts")+`";

const [aggregate, scalar] = DeclarationLifecycle();
console.log(String(aggregate), String(scalar), String(ProjectionLifecycle()));
`)
	writeFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	strictTypecheckWithRunner(t, artifacts, workingDirectory, runnerPath)
	targetOutput := run(
		t,
		workingDirectory,
		"node",
		filepath.Join(workingDirectory, "out", "runner.js"),
	)
	goOutput := executeMapOwnershipGo(t, workingDirectory)
	if targetOutput != goOutput {
		t.Fatalf(
			"map ownership TypeScript = %q, Go = %q",
			targetOutput,
			goOutput,
		)
	}
}

func compileMapOwnership(t *testing.T) emit.ProgramEmission {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: mapOwnershipDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	var roots []emit.Root
	for _, name := range []string{"DeclarationLifecycle", "ProjectionLifecycle"} {
		root, rootErr := emit.NewRoot(loaded.Types().Scope().Lookup(name))
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		roots = append(roots, root)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func targetGeneratedFunction(
	t *testing.T,
	files []emit.TargetFile,
	prefix string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, file := range files {
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Name() != nil &&
				strings.HasPrefix(function.Name().Text(), prefix) {
				return function
			}
		}
	}
	t.Fatalf("target function prefix %q is absent", prefix)
	return nil
}

func targetLocalDeclaration(
	t *testing.T,
	function tsgo.FunctionDeclaration,
	name string,
) tsgo.VariableDeclaration {
	t.Helper()
	for _, statement := range function.Body().(tsgo.Block).Statements() {
		variable, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		for _, declaration := range variable.DeclarationList().Declarations() {
			identifier, ok := declaration.Name().(tsgo.Identifier)
			if ok && identifier.Text() == name {
				return declaration
			}
		}
	}
	t.Fatalf("%s local %q is absent", function.Name().Text(), name)
	return nil
}

func typeReferenceNamed(node tsgo.TypeNode, name string) bool {
	reference, ok := node.(tsgo.TypeReferenceNode)
	if !ok {
		return false
	}
	identifier, ok := reference.TypeName().(tsgo.Identifier)
	return ok && identifier.Text() == name
}

func mapContractType(
	node tsgo.TypeNode,
	keyName string,
	valueName string,
) bool {
	reference, ok := node.(tsgo.TypeReferenceNode)
	if !ok {
		return false
	}
	identifier, ok := reference.TypeName().(tsgo.Identifier)
	arguments := reference.TypeArguments()
	return ok &&
		identifier.Text() == "GoMapValue" &&
		len(arguments) == 2 &&
		typeReferenceNamed(arguments[0], keyName) &&
		typeReferenceNamed(arguments[1], valueName)
}

func identifierText(expression tsgo.Expression) string {
	identifier, _ := expression.(tsgo.Identifier)
	if identifier == nil {
		return ""
	}
	return identifier.Text()
}

func executeMapOwnershipGo(
	t *testing.T,
	workingDirectory string,
) string {
	t.Helper()
	modulePath, err := filepath.Abs(mapOwnershipDirectory())
	if err != nil {
		t.Fatal(err)
	}
	runnerDirectory := filepath.Join(workingDirectory, "go-runner")
	writeFile(t, filepath.Join(runnerDirectory, "go.mod"), fmt.Sprintf(`module example.com/runner

go 1.26.4

require example.com/mapownership v0.0.0

replace example.com/mapownership => %s
`, filepath.ToSlash(modulePath)))
	writeFile(t, filepath.Join(runnerDirectory, "main.go"), `package main

import (
	"fmt"

	values "example.com/mapownership"
)

func main() {
	aggregate, scalar := values.DeclarationLifecycle()
	fmt.Println(aggregate, scalar, values.ProjectionLifecycle())
}
`)
	return run(
		t,
		runnerDirectory,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
}

func mapOwnershipDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"map",
		"ownership",
	)
}
