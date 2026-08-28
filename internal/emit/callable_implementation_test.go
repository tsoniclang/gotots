package emit

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	callableimplementationcontract "github.com/tsoniclang/gotots/internal/contracts/callableimplementation"
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
	"github.com/tsoniclang/gotots/internal/emit/callableimplementation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCallableImplementationsReplaceOnlyExactSelectedBodies(t *testing.T) {
	fixture := loadCallableImplementationFixture(t)
	callables := []callableimplementation.CallableDocument{
		fixture.callable(t, fixture.function("Add"), "addFast"),
		fixture.callable(t, fixture.method("Box", "Add"), "boxAddFast"),
	}
	certificate := fixture.certificate(t, callables)
	options := DefaultOptions()
	options.CallableImplementations = certificate
	emission, err := CompileWithOptions(fixture.program, fixture.roots(t), options)
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := emission.CallableImplementationPlan()
	if !ok || len(plan.Modules()) != 1 || len(plan.Targets()) != 2 {
		t.Fatal("callable implementation plan did not close exactly")
	}

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClientWithTool(
		sourceImplementationTestTool(t, repository),
		fixture.root,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var generated strings.Builder
	for _, file := range emission.Files() {
		printed, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if printErr != nil {
			t.Fatal(printErr)
		}
		generated.WriteString(printed)
	}
	actual := generated.String()
	for _, required := range []string{
		"addFast", "boxAddFast", "implementations/hot.js", "7003",
	} {
		if !strings.Contains(actual, required) {
			t.Fatalf("generated source omitted %q:\n%s", required, actual)
		}
	}
	for _, retired := range []string{"8001", "9001"} {
		if strings.Contains(actual, retired) {
			t.Fatalf("selected translated body %q survived:\n%s", retired, actual)
		}
	}
}

func TestCallableImplementationRejectsWrongVariantAndUnconsumedClaim(t *testing.T) {
	fixture := loadCallableImplementationFixture(t)
	wrongBody := fixture.callable(t, fixture.function("Add"), "addFast")
	wrongBody.SourceBodyDigest = strings.Repeat("0", 64)
	_, err := fixture.prepared(
		t,
		[]callableimplementation.CallableDocument{wrongBody},
	).Join(fixture.program)
	if err == nil || !strings.Contains(err.Error(), "selected source body differs") {
		t.Fatalf("wrong source body error = %v", err)
	}

	wrongVariant := fixture.callable(t, fixture.function("Add"), "addFast")
	wrongVariant.Variant = callableimplementation.VariantKernel
	options := DefaultOptions()
	options.CallableImplementations = fixture.certificate(
		t,
		[]callableimplementation.CallableDocument{wrongVariant},
	)
	if _, err := CompileWithOptions(fixture.program, fixture.roots(t), options); err == nil ||
		!strings.Contains(err.Error(), "kernel variant requires a generic declaration") {
		t.Fatalf("wrong callable variant error = %v", err)
	}

	options.CallableImplementations = fixture.certificate(
		t,
		[]callableimplementation.CallableDocument{
			fixture.callable(t, fixture.function("Add"), "addFast"),
			fixture.callable(t, fixture.function("dead"), "deadFast"),
		},
	)
	if _, err := CompileWithOptions(fixture.program, fixture.roots(t), options); err == nil ||
		!strings.Contains(err.Error(), "not every selected callable") {
		t.Fatalf("unconsumed callable error = %v", err)
	}
}

func TestCallableImplementationRejectsStaleProgramWithUnchangedSelectedBody(t *testing.T) {
	fixture := loadCallableImplementationFixture(t)
	claim := fixture.callable(t, fixture.function("Add"), "addFast")
	prepared := fixture.prepared(t, []callableimplementation.CallableDocument{claim})
	if err := os.WriteFile(
		filepath.Join(fixture.root, "other", "other.go"),
		[]byte("package other\nfunc Add(value int) int { return value + 7004 }\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	changed := reloadCallableImplementationFixture(t, fixture.root)
	changedClaim := changed.callable(t, changed.function("Add"), "addFast")
	if changedClaim.SourceBodyDigest != claim.SourceBodyDigest {
		t.Fatal("unrelated source mutation changed the selected callable body digest")
	}
	if _, err := prepared.Join(changed.program); err == nil ||
		!strings.Contains(err.Error(), "selected source snapshot differs") {
		t.Fatalf("stale selected-source error = %v", err)
	}
}

func TestCallableImplementationRejectsChangedGlobalOrHelperWithUnchangedSelectedBody(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		before      string
		replacement string
	}{
		{name: "referenced global", before: "const scoreBias = 3001", replacement: "const scoreBias = 3002"},
		{name: "referenced helper", before: "return value + 4001", replacement: "return value + 4002"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := loadCallableImplementationFixture(t)
			claim := fixture.callable(t, fixture.function("score"), "scoreFast")
			prepared := fixture.prepared(t, []callableimplementation.CallableDocument{claim})
			sourcePath := filepath.Join(fixture.root, "app.go")
			payload, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			changedPayload := strings.Replace(string(payload), testCase.before, testCase.replacement, 1)
			if changedPayload == string(payload) {
				t.Fatal("semantic-source mutation did not match the fixture")
			}
			if err := os.WriteFile(sourcePath, []byte(changedPayload), 0o644); err != nil {
				t.Fatal(err)
			}
			changed := reloadCallableImplementationFixture(t, fixture.root)
			changedClaim := changed.callable(t, changed.function("score"), "scoreFast")
			if changedClaim.SourceBodyDigest != claim.SourceBodyDigest {
				t.Fatal("global/helper mutation changed the selected callable body digest")
			}
			if _, err := prepared.Join(changed.program); err == nil ||
				!strings.Contains(err.Error(), "selected source snapshot differs") {
				t.Fatalf("stale selected-source error = %v", err)
			}
		})
	}
}

func TestCallableImplementationSelectsExactKernelVariant(t *testing.T) {
	fixture := loadCallableImplementationFixture(t)
	claim := fixture.callable(t, fixture.method("NumberBox", "Twice"), "numberBoxTwiceFast")
	claim.Variant = callableimplementation.VariantKernel
	options := DefaultOptions()
	options.CallableImplementations = fixture.certificate(
		t,
		[]callableimplementation.CallableDocument{claim},
	)
	emission, err := CompileWithOptions(fixture.program, fixture.roots(t), options)
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := emission.CallableImplementationPlan()
	if !ok || len(plan.Targets()) != 1 ||
		plan.Targets()[0].Variant() != callableimplementation.VariantKernel {
		t.Fatal("kernel callable implementation plan did not close exactly")
	}

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClientWithTool(
		sourceImplementationTestTool(t, repository),
		fixture.root,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var generated strings.Builder
	for _, file := range emission.Files() {
		printed, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if printErr != nil {
			t.Fatal(printErr)
		}
		generated.WriteString(printed)
	}
	actual := generated.String()
	if !strings.Contains(actual, "numberBoxTwiceFast") || strings.Contains(actual, "5009") {
		t.Fatalf("kernel body replacement is incomplete:\n%s", actual)
	}
}

type callableImplementationFixture struct {
	root    string
	program *load.Program
	source  *load.Package
}

func loadCallableImplementationFixture(t *testing.T) callableImplementationFixture {
	t.Helper()
	root := t.TempDir()
	writeSourceImplementationFixture(
		t,
		filepath.Join(root, "go.mod"),
		"module example.test/app\n\ngo 1.26.4\n",
	)
	writeSourceImplementationFixture(t, filepath.Join(root, "other", "other.go"), `package other
func Add(value int) int { return value + 7003 }
`)
	writeSourceImplementationFixture(t, filepath.Join(root, "app.go"), `package app
import "example.test/app/other"

type Box struct { Offset int }
type NumberBox[T ~int] struct{}

const scoreBias = 3001

func scoreHelper(value int) int { return value + 4001 }
func score(value int) int { return scoreHelper(value) + scoreBias }

func Add(value int) int { return value + 8001 }
func (box *Box) Add(value int) int { return value + box.Offset + 9001 }
func (box *NumberBox[T]) Twice(value T) T { if false { panic(5009) }; return value + value }
func Bind(box *Box) func(int) int { return box.Add }
func UseNumberBox(box *NumberBox[int], value int) int { return box.Twice(value) }
func UseOther(value int) int { return other.Add(value) }
func dead(value int) int { return value + 6007 }
func _() { panic("blank declarations are not callable") }
`)
	return reloadCallableImplementationFixture(t, root)
}

func reloadCallableImplementationFixture(
	t *testing.T,
	root string,
) callableImplementationFixture {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: root,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.PackageByPath("example.test/app")
	if source == nil {
		t.Fatal("source package is absent")
	}
	return callableImplementationFixture{
		root: root, program: program, source: source,
	}
}

func (f callableImplementationFixture) function(name string) *types.Func {
	function, _ := f.source.Types().Scope().Lookup(name).(*types.Func)
	return function
}

func (f callableImplementationFixture) method(
	typeName string,
	methodName string,
) *types.Func {
	declared := f.source.Types().Scope().Lookup(typeName).(*types.TypeName)
	named := types.Unalias(declared.Type()).(*types.Named)
	method, _, _ := types.LookupFieldOrMethod(
		types.NewPointer(named),
		true,
		f.source.Types(),
		methodName,
	)
	return method.(*types.Func)
}

func (f callableImplementationFixture) callable(
	t *testing.T,
	function *types.Func,
	export string,
) callableimplementation.CallableDocument {
	t.Helper()
	if function == nil {
		t.Fatal("callable fixture function is absent")
	}
	contract, err := environmentcontract.Describe(function)
	if err != nil {
		t.Fatal(err)
	}
	bodyDigest := ""
	for _, file := range f.source.Files() {
		for _, declaration := range file.Syntax().Decls {
			functionDeclaration, ok := declaration.(*ast.FuncDecl)
			if !ok || f.source.TypesInfo().Defs[functionDeclaration.Name] != function {
				continue
			}
			bodyDigest, err = callableimplementationcontract.SourceBodyDigest(
				f.source.FileSet(),
				functionDeclaration.Body,
			)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if bodyDigest == "" {
		t.Fatal("callable fixture body is absent")
	}
	return callableimplementation.CallableDocument{
		SourceIdentity:   contract.Identity(),
		SourceSignature:  contract.Signature(),
		SourceBodyDigest: bodyDigest,
		Variant:          callableimplementation.VariantSource,
		Export:           export,
	}
}

func (f callableImplementationFixture) certificate(
	t *testing.T,
	callables []callableimplementation.CallableDocument,
) *callableimplementation.Certificate {
	t.Helper()
	prepared := f.prepared(t, callables)
	certificate, err := prepared.Join(f.program)
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}

func (f callableImplementationFixture) prepared(
	t *testing.T,
	callables []callableimplementation.CallableDocument,
) *callableimplementation.Prepared {
	t.Helper()
	sort.Slice(callables, func(left, right int) bool {
		leftKey := callables[left].SourceIdentity + "\x00" + string(callables[left].Variant)
		rightKey := callables[right].SourceIdentity + "\x00" + string(callables[right].Variant)
		return leftKey < rightKey
	})
	implementationRoot := filepath.Join(f.root, "implementation")
	writeSourceImplementationFixture(
		t,
		filepath.Join(implementationRoot, "hot.ts"),
		"export function addFast(value: number): number { return value + 1; }\n"+
			"export function boxAddFast(box: object, value: number): number { return value + 2; }\n"+
			"export function deadFast(value: number): number { return value + 3; }\n"+
			"export function numberBoxTwiceFast(box: object, operation: object, value: number): number { return value; }\n",
	)
	profile := f.program.BuildProfile()
	document := callableimplementation.Document{
		SchemaVersion:       callableimplementation.SchemaVersion,
		SourceProgramDigest: f.program.SourceDigest(),
		Package: callableimplementation.PackageDocument{
			ImportPath: "example.test/app",
			ModulePath: "example.test/app",
		},
		Build: callableimplementation.BuildDocument{
			GoVersion:  profile.ToolchainVersion(),
			GOOS:       profile.GOOS(),
			GOARCH:     profile.GOARCH(),
			CGOEnabled: profile.CgoEnabled(),
			BuildTags:  profile.Tags(),
		},
		Compilation: callableimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct",
		},
		Source: "hot.ts",
		Output: "implementations/hot.ts",
		Envelope: implementationcontract.Envelope{
			Kind: implementationcontract.EnvelopeExact,
		},
		Callables: callables,
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(implementationRoot, "contract.json")
	writeSourceImplementationFixture(t, contractPath, string(payload)+"\n")
	prepared, err := callableimplementation.PrepareAll(
		callableimplementation.Config{
			ContractPaths: []string{contractPath},
			BuildProfile:  profile,
			Compilation: callableimplementation.CompilationDocument{
				Integers: "number", EvaluationOrder: "direct",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}

func (f callableImplementationFixture) roots(t *testing.T) []Root {
	t.Helper()
	roots, err := ExportedAPIRoots(f.source)
	if err != nil {
		t.Fatal(err)
	}
	return roots
}
