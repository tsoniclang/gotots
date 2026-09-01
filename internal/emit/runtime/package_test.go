package runtime

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAssemblePackageOwnsExactGeneratedRuntimeSurface(t *testing.T) {
	assembled, err := AssemblePackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationNumber),
		map[api.RuntimeSymbol]struct{}{
			api.RuntimeStringIndex: {},
		},
		[]api.PrimitiveAlias{api.PrimitiveInt32},
	)
	if err != nil {
		t.Fatal(err)
	}
	if assembled.Name() != "@gotots/runtime" ||
		assembled.RootPath() != "runtime" ||
		assembled.ManifestPath() != "runtime/package.json" ||
		assembled.Profile() != api.IntegerRepresentationNumber {
		t.Fatalf(
			"runtime package identity = %q/%q/%q/%s",
			assembled.Name(),
			assembled.RootPath(),
			assembled.ManifestPath(),
			assembled.Profile(),
		)
	}
	paths := make([]string, 0)
	for _, file := range assembled.Files() {
		paths = append(paths, file.OutputPath())
	}
	wantPaths := []string{
		"runtime/interface-value.ts",
		"runtime/panic.ts",
		"runtime/scalars.ts",
		"runtime/source-fact.ts",
		"runtime/string.ts",
	}
	if !slices.Equal(paths, wantPaths) {
		t.Fatalf("runtime files = %v, want %v", paths, wantPaths)
	}
	var manifest struct {
		Name    string `json:"name"`
		Private bool   `json:"private"`
		Type    string `json:"type"`
		GoToTS  struct {
			IntegerRepresentation string `json:"integerRepresentation"`
			NativeIntegerBits     uint8  `json:"nativeIntegerBits"`
		} `json:"gotots"`
		Exports map[string]string `json:"exports"`
	}
	if err := json.Unmarshal(assembled.Manifest(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "@gotots/runtime" ||
		!manifest.Private ||
		manifest.Type != "module" ||
		manifest.GoToTS.IntegerRepresentation != "number" {
		t.Fatalf("runtime manifest metadata = %#v", manifest)
	}
	if manifest.GoToTS.NativeIntegerBits != 64 {
		t.Fatalf("runtime native integer width = %d, want 64", manifest.GoToTS.NativeIntegerBits)
	}
	if len(manifest.Exports) != len(wantPaths) {
		t.Fatalf(
			"runtime manifest exports = %d, want %d",
			len(manifest.Exports),
			len(wantPaths),
		)
	}
	scalar := manifest.Exports["./scalars.js"]
	if scalar != "./scalars.js" {
		t.Fatalf("scalar package export = %#v", scalar)
	}
	if assembled.Fingerprint() == "" {
		t.Fatal("runtime package fingerprint is empty")
	}
}

func TestAssemblePackageRejectsDuplicateAliases(t *testing.T) {
	_, err := AssemblePackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationNumber),
		nil,
		[]api.PrimitiveAlias{
			api.PrimitiveInt32,
			api.PrimitiveInt32,
		},
	)
	if err == nil {
		t.Fatal("duplicate primitive alias was accepted")
	}
}

func TestSameModuleRuntimeDependenciesPrecedeConsumers(t *testing.T) {
	assembled, err := AssemblePackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationNumber),
		map[api.RuntimeSymbol]struct{}{api.RuntimeMap: {}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	var classes []string
	for _, file := range assembled.Files() {
		if file.OutputPath() != "runtime/map.ts" {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			if class, ok := statement.(tsgo.ClassDeclaration); ok {
				classes = append(classes, class.Name().Text())
			}
		}
	}
	if !slices.Equal(classes, []string{"GoMapValue", "GoMap"}) {
		t.Fatalf("map class order = %v, want nominal base before consumer", classes)
	}
}

func TestProviderCertificationRuntimeHasOneExactSynchronousContract(t *testing.T) {
	assembled, err := AssembleProviderCertificationPackage(
		tsgo.NewFactory(),
		testScalarABI(t, api.IntegerRepresentationNumber),
		map[api.RuntimeSymbol]struct{}{
			api.RuntimeBuiltinErrorType: {},
			api.RuntimeReceiveChannel:   {},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		GoToTS struct {
			IntegerRepresentation string `json:"integerRepresentation"`
			NativeIntegerBits     uint8  `json:"nativeIntegerBits"`
		} `json:"gotots"`
	}
	if err := json.Unmarshal(assembled.Manifest(), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.GoToTS.IntegerRepresentation != "number" ||
		manifest.GoToTS.NativeIntegerBits != 64 {
		t.Fatalf(
			"provider certification scalar ABI = %q/%d",
			manifest.GoToTS.IntegerRepresentation,
			manifest.GoToTS.NativeIntegerBits,
		)
	}

	var errorResult tsgo.TypeNode
	var receiveResult tsgo.TypeNode
	for _, file := range assembled.Files() {
		for _, statement := range file.SourceFile().Statements() {
			switch declaration := statement.(type) {
			case tsgo.TypeAliasDeclaration:
				if declaration.Name().Text() == "Awaitable" {
					t.Fatal("provider certification runtime retained Awaitable")
				}
			case tsgo.InterfaceDeclaration:
				for _, member := range declaration.Members() {
					method, ok := member.(tsgo.MethodSignatureDeclaration)
					if !ok {
						continue
					}
					name, ok := method.Name().(tsgo.Identifier)
					if !ok {
						continue
					}
					switch {
					case declaration.Name().Text() == "GoError" && name.Text() == "Error":
						errorResult = method.Type()
					case declaration.Name().Text() == "GoReceiveChannel" && name.Text() == "receive":
						receiveResult = method.Type()
					}
				}
			}
		}
	}
	if errorResult == nil ||
		errorResult.Kind() != tsgo.SyntaxKindStringKeyword {
		t.Fatalf("provider GoError result = %T", errorResult)
	}
	if _, ok := receiveResult.(tsgo.TupleTypeNode); !ok {
		t.Fatalf("provider receive result = %T", receiveResult)
	}
}

func testScalarABI(
	t *testing.T,
	profile api.IntegerRepresentation,
) api.ScalarABI {
	t.Helper()
	abi, err := api.NewScalarABI(profile, api.NativeIntegerWidth64)
	if err != nil {
		t.Fatal(err)
	}
	return abi
}

func TestDefinitionsExactJoinRequestedSymbols(t *testing.T) {
	factory := tsgo.NewFactory()
	index := runtimeDefinition(t, factory, api.RuntimeStringIndex)
	slice := runtimeDefinition(t, factory, api.RuntimeStringSlice)
	statements, err := exactDefinitions(
		api.RuntimeModuleString,
		[]api.RuntimeSymbol{api.RuntimeStringIndex, api.RuntimeStringSlice},
		[]Definition{slice, index},
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

func TestDefinitionsRejectJoinMutations(t *testing.T) {
	factory := tsgo.NewFactory()
	index := runtimeDefinition(t, factory, api.RuntimeStringIndex)
	slice := runtimeDefinition(t, factory, api.RuntimeStringSlice)
	array := runtimeDefinition(t, factory, api.RuntimeArray)
	tests := []struct {
		name        string
		requested   []api.RuntimeSymbol
		definitions []Definition
	}{
		{"missing", []api.RuntimeSymbol{api.RuntimeStringIndex, api.RuntimeStringSlice}, []Definition{index}},
		{"duplicate", []api.RuntimeSymbol{api.RuntimeStringIndex}, []Definition{index, index}},
		{"extra", []api.RuntimeSymbol{api.RuntimeStringIndex}, []Definition{index, slice}},
		{"wrong module", []api.RuntimeSymbol{api.RuntimeArray}, []Definition{array}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := exactDefinitions(
				api.RuntimeModuleString,
				test.requested,
				test.definitions,
			)
			var assemblyError *AssemblyError
			if !errors.As(err, &assemblyError) {
				t.Fatalf("error = %v, want runtime assembly error", err)
			}
		})
	}
}

func TestDependencyClosureIncludesEveryTransitiveOwner(t *testing.T) {
	closure, err := dependencyClosure(map[api.RuntimeSymbol]struct{}{
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

func TestModuleImportsExactDependencyContract(t *testing.T) {
	factory := tsgo.NewFactory()
	imports, err := moduleImports(
		factory,
		"runtime/array.ts",
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeArray},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 3 {
		t.Fatalf("array runtime imports = %d, want three", len(imports))
	}
	got := make(map[string][]string, len(imports))
	for _, statement := range imports {
		declaration := statement.(tsgo.ImportDeclaration)
		module := declaration.ModuleSpecifier().(tsgo.StringLiteral)
		bindings := declaration.ImportClause().NamedBindings().(tsgo.NamedImports).
			Elements()
		for _, binding := range bindings {
			if binding.PropertyName() != nil {
				t.Fatalf("array runtime binding = %#v, want direct bindings", binding)
			}
			got[module.Text()] = append(got[module.Text()], binding.Name().Text())
		}
	}
	wantImports := map[string][]string{
		"@tsonic/core/lang.js": {"attribute"},
		"./panic.js":           {"GoPanic"},
		"./source-fact.js":     {"GoAggregateFact", "GoOperationFact"},
	}
	if !maps.EqualFunc(got, wantImports, slices.Equal) {
		t.Fatalf("array runtime imports = %v, want %v", got, wantImports)
	}
}

func runtimeDefinition(
	t *testing.T,
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
) Definition {
	t.Helper()
	definition, err := NewDefinition(
		symbol,
		factory.EmptyStatement(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}
