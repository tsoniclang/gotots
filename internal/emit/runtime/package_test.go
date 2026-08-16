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
		api.ConcurrencySemanticsDisabled,
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
		api.ConcurrencySemanticsDisabled,
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

func TestAwaitableSupportIsEmittedOnlyWhenRequested(t *testing.T) {
	factory := tsgo.NewFactory()
	without, err := AssemblePackage(
		factory,
		testScalarABI(t, api.IntegerRepresentationNumber),
		api.ConcurrencySemanticsCooperative,
		nil,
		[]api.PrimitiveAlias{api.PrimitiveInt32},
	)
	if err != nil {
		t.Fatal(err)
	}
	withoutStatements := without.Files()[0].SourceFile().Statements()
	if len(withoutStatements) != 1 ||
		withoutStatements[0].(tsgo.TypeAliasDeclaration).Name().Text() != "int32" {
		t.Fatalf("undemanded scalar support = %#v", withoutStatements)
	}

	with, err := AssemblePackage(
		factory,
		testScalarABI(t, api.IntegerRepresentationNumber),
		api.ConcurrencySemanticsCooperative,
		map[api.RuntimeSymbol]struct{}{api.RuntimeAwaitable: {}},
		[]api.PrimitiveAlias{api.PrimitiveInt32},
	)
	if err != nil {
		t.Fatal(err)
	}
	withStatements := with.Files()[0].SourceFile().Statements()
	if len(withStatements) != 2 {
		t.Fatalf("demanded scalar support statements = %d, want two", len(withStatements))
	}
	wantNames := []string{"Awaitable", "int32"}
	for index, statement := range withStatements {
		declaration, ok := statement.(tsgo.TypeAliasDeclaration)
		if !ok || declaration.Name().Text() != wantNames[index] {
			t.Fatalf("scalar support statement %d = %#v, want %s", index, statement, wantNames[index])
		}
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
	if len(imports) != 1 {
		t.Fatalf("array runtime imports = %d, want one", len(imports))
	}
	got := make(map[string]string, len(imports))
	for _, statement := range imports {
		declaration := statement.(tsgo.ImportDeclaration)
		module := declaration.ModuleSpecifier().(tsgo.StringLiteral)
		bindings := declaration.ImportClause().NamedBindings().(tsgo.NamedImports).
			Elements()
		if len(bindings) != 1 || bindings[0].PropertyName() != nil {
			t.Fatalf("array runtime bindings = %#v, want one direct binding", bindings)
		}
		got[module.Text()] = bindings[0].Name().Text()
	}
	wantImports := map[string]string{
		"./panic.js": "GoPanic",
	}
	if !maps.Equal(got, wantImports) {
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
