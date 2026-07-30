package emit

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDemandSchedulerExactJoinsIndependentReferenceClosure(t *testing.T) {
	program := loadSchedulerFixture(t)
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	expected := independentReferenceClosure(t, program, roots)
	actual := emittedObjectCounts(t, program, roots)
	if err := verifyObjectMultiset(expected, actual); err != nil {
		t.Fatal(err)
	}
	if labels := objectLabels(expected); strings.Join(labels, ",") !=
		"example.com/demand/api.Compute,"+
			"example.com/demand/api.Run,"+
			"example.com/demand/mathx.Even,"+
			"example.com/demand/mathx.Odd,"+
			"example.com/demand/mathx.Offset,"+
			"example.com/demand/mathx.unsupportedValue,"+
			"example.com/demand/service.Compute" {
		t.Fatalf("independent reachable closure = %v", labels)
	}
}

func TestDemandSchedulerJoinMutationControls(t *testing.T) {
	program := loadSchedulerFixture(t)
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	expected := independentReferenceClosure(t, program, roots)
	actual := emittedObjectCounts(t, program, roots)
	target := objectByLabel(t, expected, "example.com/demand/mathx.Odd")

	t.Run("omitted enqueue", func(t *testing.T) {
		mutated := cloneObjectCounts(actual)
		delete(mutated, target)
		if err := verifyObjectMultiset(expected, mutated); err == nil {
			t.Fatal("omitted reachable object passed the exact scheduler join")
		}
	})
	t.Run("duplicate owner", func(t *testing.T) {
		mutated := cloneObjectCounts(actual)
		mutated[target]++
		if err := verifyObjectMultiset(expected, mutated); err == nil {
			t.Fatal("duplicate target owner passed the exact scheduler join")
		}
	})
}

func TestFunctionValueReferenceSchedulesExactCrossPackageDeclaration(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: filepath.Join(
			root,
			"testdata",
			"constructs",
			"expression",
			"function-value",
			"cross-package",
		),
		Pattern: "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	expected := independentReferenceClosure(t, program, roots)
	actual := emittedObjectCounts(t, program, roots)
	if err := verifyObjectMultiset(expected, actual); err != nil {
		t.Fatal(err)
	}
	labels := strings.Join(objectLabels(actual), ",")
	if labels != "example.com/callbackdemand/api.Apply,"+
		"example.com/callbackdemand/api.Run,"+
		"example.com/callbackdemand/worker.Double" {
		t.Fatalf("callable demand closure = %s", labels)
	}
}

func loadSchedulerFixture(t *testing.T) *load.Program {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: filepath.Join(root, "testdata", "projects", "demand-program"),
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func independentReferenceClosure(
	t *testing.T,
	program *load.Program,
	roots []Root,
) map[types.Object]int {
	t.Helper()
	declarations := make(map[types.Object]ast.Decl)
	infoByObject := make(map[types.Object]*types.Info)
	packageVariables := make(map[*types.Package][]types.Object)
	for _, sourcePackage := range program.Packages() {
		info := sourcePackage.TypesInfo()
		for _, sourceFile := range sourcePackage.Files() {
			for _, declaration := range sourceFile.Syntax().Decls {
				switch declaration := declaration.(type) {
				case *ast.FuncDecl:
					if declaration.Recv != nil || declaration.Name.Name == "init" {
						continue
					}
					if object, ok := info.Defs[declaration.Name].(*types.Func); ok {
						declarations[object] = declaration
						infoByObject[object] = info
					}
				case *ast.GenDecl:
					if declaration.Tok != token.CONST &&
						declaration.Tok != token.VAR {
						continue
					}
					for _, sourceSpec := range declaration.Specs {
						valueSpec, ok := sourceSpec.(*ast.ValueSpec)
						if !ok {
							t.Fatalf("constant declaration contains %T", sourceSpec)
						}
						for _, name := range valueSpec.Names {
							object := info.Defs[name]
							switch object := object.(type) {
							case *types.Const:
								declarations[object] = declaration
								infoByObject[object] = info
							case *types.Var:
								if object.Name() == "_" ||
									object.Parent() != sourcePackage.Types().Scope() {
									continue
								}
								declarations[object] = declaration
								infoByObject[object] = info
								packageVariables[sourcePackage.Types()] = append(
									packageVariables[sourcePackage.Types()],
									object,
								)
							}
						}
					}
				}
			}
		}
	}

	pending := make([]types.Object, 0, len(roots))
	for _, root := range roots {
		pending = append(pending, root.object)
	}
	expected := make(map[types.Object]int)
	reachedPackages := make(map[*types.Package]struct{})
	var reachPackage func(*types.Package)
	reachPackage = func(sourcePackage *types.Package) {
		if sourcePackage == nil {
			return
		}
		if _, reached := reachedPackages[sourcePackage]; reached {
			return
		}
		reachedPackages[sourcePackage] = struct{}{}
		pending = append(pending, packageVariables[sourcePackage]...)
		imports := sourcePackage.Imports()
		sort.Slice(imports, func(left, right int) bool {
			return imports[left].Path() < imports[right].Path()
		})
		for _, imported := range imports {
			reachPackage(imported)
		}
	}
	for len(pending) != 0 {
		sort.Slice(pending, func(left, right int) bool {
			return emitordering.CompareObjects(
				pending[left],
				pending[right],
			) < 0
		})
		object := pending[0]
		pending = pending[1:]
		if expected[object] != 0 {
			continue
		}
		reachPackage(object.Pkg())
		declaration := declarations[object]
		if declaration == nil {
			t.Fatalf("root/reference %s has no independently derived declaration", objectLabel(object))
		}
		expected[object] = 1
		info := infoByObject[object]
		ast.Inspect(declaration, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			referenced := info.Uses[identifier]
			if declarations[referenced] != nil && expected[referenced] == 0 {
				pending = append(pending, referenced)
			}
			return true
		})
	}
	return expected
}

func emittedObjectCounts(
	t *testing.T,
	program *load.Program,
	roots []Root,
) map[types.Object]int {
	t.Helper()
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range roots {
		if err := session.require(root.object); err != nil {
			t.Fatal(err)
		}
	}
	for {
		if object, ok := session.scheduler.next(); ok {
			if err := session.emit(object); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if requirements, ok := session.requirements.nextBatch(); ok {
			if err := session.applyDeclarationRequirements(requirements); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if object, ok := session.artifacts.NextDirty(); ok {
			if err := session.reconstructScheduledArtifact(object); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if sourcePackage, ok := session.packageInitializations.next(); ok {
			if err := session.emitPackageInitialization(sourcePackage); err != nil {
				t.Fatal(err)
			}
			continue
		}
		break
	}
	actual := make(map[types.Object]int)
	for _, builder := range session.builders {
		for owner := range builder.byOwner {
			object, sourceOwned := owner.Source()
			if sourceOwned {
				actual[object]++
			}
		}
	}
	for _, builder := range session.packageBuilders {
		for object := range builder.storageByObject {
			actual[object]++
		}
	}
	for object := range session.scheduler.emitted {
		if actual[object] != 1 {
			t.Fatalf(
				"scheduled object %s owns %d target declarations",
				objectLabel(object),
				actual[object],
			)
		}
	}
	return actual
}

func verifyObjectMultiset(
	expected map[types.Object]int,
	actual map[types.Object]int,
) error {
	var differences []string
	for object, expectedCount := range expected {
		if actual[object] != expectedCount {
			differences = append(
				differences,
				fmt.Sprintf(
					"%s expected %d actual %d",
					objectLabel(object),
					expectedCount,
					actual[object],
				),
			)
		}
	}
	for object, actualCount := range actual {
		if expected[object] == 0 {
			differences = append(
				differences,
				fmt.Sprintf("%s expected 0 actual %d", objectLabel(object), actualCount),
			)
		}
	}
	sort.Strings(differences)
	if len(differences) != 0 {
		return fmt.Errorf("scheduler exact join failed: %s", strings.Join(differences, "; "))
	}
	return nil
}

func cloneObjectCounts(source map[types.Object]int) map[types.Object]int {
	result := make(map[types.Object]int, len(source))
	for object, count := range source {
		result[object] = count
	}
	return result
}

func objectByLabel(
	t *testing.T,
	objects map[types.Object]int,
	label string,
) types.Object {
	t.Helper()
	for object := range objects {
		if objectLabel(object) == label {
			return object
		}
	}
	t.Fatalf("object %s is absent", label)
	return nil
}

func objectLabels(objects map[types.Object]int) []string {
	result := make([]string, 0, len(objects))
	for object := range objects {
		result = append(result, objectLabel(object))
	}
	sort.Strings(result)
	return result
}

func objectLabel(object types.Object) string {
	if object == nil {
		return "<nil>"
	}
	if object.Pkg() == nil {
		return object.Name()
	}
	return object.Pkg().Path() + "." + object.Name()
}

func TestRuntimeDefinitionsExactJoinRequestedSymbols(t *testing.T) {
	factory := tsgo.NewFactory()
	index := runtimeDefinition(t, factory, api.RuntimeStringIndex)
	slice := runtimeDefinition(t, factory, api.RuntimeStringSlice)
	statements, err := exactRuntimeDefinitions(
		api.RuntimeModuleString,
		[]api.RuntimeSymbol{api.RuntimeStringIndex, api.RuntimeStringSlice},
		[]runtimeemission.Definition{slice, index},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 2 ||
		statements[0] != index.Statement() ||
		statements[1] != slice.Statement() {
		t.Fatalf("runtime statements = %#v", statements)
	}
}

func TestRuntimeDefinitionsRejectJoinMutations(t *testing.T) {
	factory := tsgo.NewFactory()
	index := runtimeDefinition(t, factory, api.RuntimeStringIndex)
	slice := runtimeDefinition(t, factory, api.RuntimeStringSlice)
	pointer := runtimeDefinition(t, factory, api.RuntimePointer)
	tests := []struct {
		name        string
		requested   []api.RuntimeSymbol
		definitions []runtimeemission.Definition
	}{
		{"missing", []api.RuntimeSymbol{api.RuntimeStringIndex, api.RuntimeStringSlice}, []runtimeemission.Definition{index}},
		{"duplicate", []api.RuntimeSymbol{api.RuntimeStringIndex}, []runtimeemission.Definition{index, index}},
		{"extra", []api.RuntimeSymbol{api.RuntimeStringIndex}, []runtimeemission.Definition{index, slice}},
		{"wrong module", []api.RuntimeSymbol{api.RuntimePointer}, []runtimeemission.Definition{pointer}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := exactRuntimeDefinitions(
				api.RuntimeModuleString,
				test.requested,
				test.definitions,
			)
			var assemblyError *runtimeemission.AssemblyError
			if !errors.As(err, &assemblyError) {
				t.Fatalf("error = %v, want runtime assembly error", err)
			}
		})
	}
}

func TestRuntimeDependencyClosureIncludesEveryTransitiveOwner(t *testing.T) {
	closure, err := runtimeDependencyClosure(map[api.RuntimeSymbol]struct{}{
		api.RuntimeArray:         {},
		api.RuntimeIntegerDivide: {},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[api.RuntimeSymbol]struct{}{
		api.RuntimeArray:             {},
		api.RuntimeIntegerDivide:     {},
		api.RuntimePanic:             {},
		api.RuntimePanicValue:        {},
		api.RuntimeInterfaceValue:    {},
		api.RuntimeErrorMethodToken:  {},
		api.RuntimeRuntimeErrorToken: {},
	}
	if len(closure) != len(want) {
		t.Fatalf("runtime closure = %v, want %v", closure, want)
	}
	for symbol := range want {
		if _, ok := closure[symbol]; !ok {
			t.Fatalf("runtime closure omits symbol %d", symbol)
		}
	}
}

func TestRuntimeModuleImportsExactDependencyContract(t *testing.T) {
	session := &programSession{factory: tsgo.NewFactory()}
	imports, err := session.runtimeModuleImports(
		"runtime/array.ts",
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeArray},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 1 {
		t.Fatalf("array runtime imports = %d, want one", len(imports))
	}
	declaration := imports[0].(tsgo.ImportDeclaration)
	module := declaration.ModuleSpecifier().(tsgo.StringLiteral)
	if module.Text() != "./panic.js" {
		t.Fatalf("array runtime dependency = %q, want ./panic.js", module.Text())
	}
	bindings := declaration.ImportClause().NamedBindings().(tsgo.NamedImports).
		Elements()
	if len(bindings) != 1 ||
		bindings[0].Name().Text() != "GoPanic" ||
		bindings[0].PropertyName() != nil {
		t.Fatalf("array runtime bindings = %#v, want direct GoPanic", bindings)
	}
}

func runtimeDefinition(
	t *testing.T,
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
) runtimeemission.Definition {
	t.Helper()
	definition, err := runtimeemission.NewDefinition(
		symbol,
		factory.EmptyStatement(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}
