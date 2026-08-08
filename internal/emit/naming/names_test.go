package naming

import (
	"errors"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNameOwnerSeparatesShadowAndTemporaryNamespaces(t *testing.T) {
	packageScope := types.NewScope(nil, token.NoPos, token.NoPos, "package")
	fileScope := types.NewScope(packageScope, token.NoPos, token.NoPos, "file")
	functionScope := types.NewScope(fileScope, token.NoPos, token.NoPos, "function")
	blockScope := types.NewScope(functionScope, token.NoPos, token.NoPos, "block")
	outer := types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int])
	shadow := types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int])
	reservedShadow := types.NewVar(
		token.NoPos,
		nil,
		"value__shadow_1",
		types.Typ[types.Int],
	)
	reservedTemporary := types.NewVar(
		token.NoPos,
		nil,
		"__gotots_assign_0",
		types.Typ[types.Int],
	)
	reservedResults := types.NewVar(
		token.NoPos,
		nil,
		"__gotots_results_0",
		types.Typ[types.Int],
	)
	functionScope.Insert(outer)
	blockScope.Insert(shadow)
	blockScope.Insert(reservedShadow)
	blockScope.Insert(reservedTemporary)
	blockScope.Insert(reservedResults)
	info := &types.Info{Defs: map[*ast.Ident]types.Object{
		{Name: "value"}:             outer,
		{Name: "shadow"}:            shadow,
		{Name: "reservedShadow"}:    reservedShadow,
		{Name: "reservedTemporary"}: reservedTemporary,
		{Name: "reservedResults"}:   reservedResults,
	}}
	owner := newNameOwner(packageScope, info)

	if name, err := owner.declare(outer, targetBinding{}); err != nil || name != "value" {
		t.Fatalf("outer = %q, %v", name, err)
	}
	if name, err := owner.declare(shadow, targetBinding{}); err != nil ||
		name != "value__shadow_2" {
		t.Fatalf("shadow = %q, %v", name, err)
	}
	file := &File{
		owner:       owner,
		temporaries: make(map[api.TemporaryKind]uint64),
	}
	if name, err := file.Temporary(api.TemporaryAssignmentValue); err != nil ||
		name != "__gotots_assign_1" {
		t.Fatalf("temporary = %q, %v", name, err)
	}
	if name, err := file.Temporary(api.TemporaryMultipleResults); err != nil ||
		name != "__gotots_results_1" {
		t.Fatalf("result temporary = %q, %v", name, err)
	}
}

func TestNameOwnerSeparatesNestedButNotSiblingDeclarationSpaces(t *testing.T) {
	packageScope := types.NewScope(nil, token.NoPos, token.NoPos, "package")
	fileScope := types.NewScope(packageScope, token.NoPos, token.NoPos, "file")
	firstFunction := types.NewScope(fileScope, token.NoPos, token.NoPos, "first function")
	firstBlock := types.NewScope(firstFunction, token.NoPos, token.NoPos, "first block")
	siblingBlock := types.NewScope(firstFunction, token.NoPos, token.NoPos, "sibling block")
	secondFunction := types.NewScope(fileScope, token.NoPos, token.NoPos, "second function")
	firstChild := types.NewVar(token.NoPos, nil, "item", types.Typ[types.Int])
	sibling := types.NewVar(token.NoPos, nil, "item", types.Typ[types.Int])
	lateParent := types.NewVar(token.NoPos, nil, "item", types.Typ[types.Int])
	otherFunction := types.NewVar(token.NoPos, nil, "item", types.Typ[types.Int])
	firstBlock.Insert(firstChild)
	siblingBlock.Insert(sibling)
	firstFunction.Insert(lateParent)
	secondFunction.Insert(otherFunction)
	info := &types.Info{Defs: map[*ast.Ident]types.Object{
		{Name: "firstChild"}:    firstChild,
		{Name: "sibling"}:       sibling,
		{Name: "lateParent"}:    lateParent,
		{Name: "otherFunction"}: otherFunction,
	}}
	owner := newNameOwner(packageScope, info)

	if name, err := owner.declare(firstChild, targetBinding{}); err != nil ||
		name != "item__shadow_1" {
		t.Fatalf("first child = %q, %v", name, err)
	}
	if name, err := owner.declare(sibling, targetBinding{}); err != nil ||
		name != "item__shadow_1" {
		t.Fatalf("sibling = %q, %v", name, err)
	}
	if name, err := owner.declare(lateParent, targetBinding{}); err != nil ||
		name != "item" {
		t.Fatalf("late parent = %q, %v", name, err)
	}
	if name, err := owner.declare(otherFunction, targetBinding{}); err != nil ||
		name != "item" {
		t.Fatalf("other function = %q, %v", name, err)
	}
}

func TestNameOwnerIndexesCheckerImplicitDeclarations(t *testing.T) {
	packageScope := types.NewScope(nil, token.NoPos, token.NoPos, "package")
	functionScope := types.NewScope(
		packageScope,
		token.NoPos,
		token.NoPos,
		"function",
	)
	caseScope := types.NewScope(
		functionScope,
		token.NoPos,
		token.NoPos,
		"type switch case",
	)
	implicit := types.NewVar(
		token.NoPos,
		nil,
		"selected",
		types.Typ[types.Int],
	)
	caseScope.Insert(implicit)
	owner := newNameOwner(packageScope, &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Implicits: map[ast.Node]types.Object{
			&ast.CaseClause{}: implicit,
		},
	})

	name, err := owner.declare(implicit, targetBinding{})
	if err != nil {
		t.Fatal(err)
	}
	if name != "selected" {
		t.Fatalf("implicit declaration = %q, want selected", name)
	}
}

func TestNameOwnerRejectsDeclarationOutsideIndexedTypeGraph(t *testing.T) {
	packageScope := types.NewScope(nil, token.NoPos, token.NoPos, "package")
	fileScope := types.NewScope(packageScope, token.NoPos, token.NoPos, "file")
	functionScope := types.NewScope(fileScope, token.NoPos, token.NoPos, "function")
	owner := newNameOwner(packageScope, &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	})
	unindexed := types.NewVar(token.NoPos, nil, "late", types.Typ[types.Int])
	functionScope.Insert(unindexed)

	_, err := owner.declare(unindexed, targetBinding{})
	var nameError *api.NameError
	if !errors.As(err, &nameError) ||
		nameError.Name != "late" ||
		nameError.Reason != "declaration object was not indexed from its Go scope" {
		t.Fatalf("error = %#v, want indexed-scope NameError", err)
	}
}

func TestBlankSlotsAreTargetOnlyAndBlankMethodsAreNotPreallocated(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/blank", "blank")
	typeName := types.NewTypeName(
		token.Pos(1),
		sourcePackage,
		"Item",
		nil,
	)
	named := types.NewNamed(typeName, types.NewStruct(nil, nil), nil)
	sourcePackage.Scope().Insert(typeName)
	receiver := types.NewVar(
		token.Pos(2),
		sourcePackage,
		"",
		named,
	)
	blankMethod := types.NewFunc(
		token.Pos(3),
		sourcePackage,
		"_",
		types.NewSignatureType(
			receiver,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(),
			false,
		),
	)
	owner := NewOwner(
		sourcePackage.Scope(),
		&types.Info{Defs: map[*ast.Ident]types.Object{
			{Name: "_"}: blankMethod,
		}},
		NewRegistry(),
	)
	if _, exists := owner.targetNameByObject[blankMethod]; exists {
		t.Fatal("blank method received a target declaration name")
	}

	file := &File{owner: owner}
	blankParameter := types.NewVar(
		token.Pos(4),
		sourcePackage,
		"_",
		types.Typ[types.Int],
	)
	parameterName, err := file.Parameter(blankParameter, 2)
	if err != nil || parameterName != "$2" {
		t.Fatalf("blank parameter = %q, %v; want $2", parameterName, err)
	}
	resultName, err := file.Result(blankParameter, 2)
	if err != nil || resultName != "$result2" {
		t.Fatalf("blank result = %q, %v; want $result2", resultName, err)
	}
	if _, exists := owner.byObject[blankParameter]; exists {
		t.Fatal("blank slot entered the Go binding table")
	}
}

func TestCrossPackageReferenceRequiresItsExactObjectBeforeImporting(t *testing.T) {
	currentPackage := types.NewPackage("example.com/current", "current")
	importedPackage := types.NewPackage("example.com/dependency", "dependency")
	object := types.NewFunc(
		token.Pos(1),
		importedPackage,
		"Run",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	importedPackage.Scope().Insert(object)
	sourceFile := &ast.File{}
	declarationFile := &ast.File{}
	registry := NewRegistry()
	if err := registry.reserve(object, targetBinding{
		name:         "Run",
		sourceFile:   declarationFile,
		sourcePath:   "modules/dependency/dependency.ts",
		moduleExport: true,
		kind:         targetBindingSource,
	}); err != nil {
		t.Fatal(err)
	}
	required := errors.New("enqueue mutation sentinel")
	names := testFileNames(
		t,
		NewOwner(
			currentPackage.Scope(),
			&types.Info{Defs: make(map[*ast.Ident]types.Object)},
			registry,
		),
		sourceFile,
		currentPackage.Scope(),
		tsgo.NewFactory(),
		"modules/current/current.ts",
		stubEnvironmentObserver{requireUse: func(
			actual types.Object,
			demand environmentcontract.UseDemand,
			selection gostdlib.UseSelection,
		) error {
			if actual != object {
				t.Fatalf("required object = %v, want imported Run", actual)
			}
			if demand != environmentcontract.UseDemandCallable ||
				selection.Kind() != gostdlib.UseSelectionNone {
				t.Fatalf(
					"required demand/selection = %v/%v, want callable/none",
					demand,
					selection.Kind(),
				)
			}
			return required
		}},
	)
	if _, err := names.Reference(object); !errors.Is(err, required) {
		t.Fatalf("reference error = %v, want enqueue sentinel", err)
	}
}

func TestPackageStatePlacementRejectsRuntimeImports(t *testing.T) {
	factory := tsgo.NewFactory()
	typeRequest, err := api.NewImportRequest(
		factory,
		api.ImportPhaseType,
		"./model.js",
		"Model",
		"Model",
	)
	if err != nil {
		t.Fatal(err)
	}
	placement := targetplacement.New()
	if err := placement.Apply([]api.RootRequest{typeRequest}); err != nil {
		t.Fatal(err)
	}
	if err := placement.RequireTypeOnly(); err != nil {
		t.Fatalf("type-only state placement failed: %v", err)
	}

	valueRequest, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./runtime.js",
		"Runtime",
		"Runtime",
	)
	if err != nil {
		t.Fatal(err)
	}
	placement = targetplacement.New()
	if err := placement.Apply([]api.RootRequest{valueRequest}); err != nil {
		t.Fatal(err)
	}
	var placementError *api.PlacementError
	if err := placement.RequireTypeOnly(); !errors.As(err, &placementError) {
		t.Fatalf("runtime state import error = %T, want PlacementError", err)
	}
}

func TestValueImportDominatesTypeRequestIndependentOfOrder(t *testing.T) {
	factory := tsgo.NewFactory()
	typeRequest, err := api.NewImportRequest(
		factory,
		api.ImportPhaseType,
		"./model.js",
		"Point",
		"Point__from_model",
	)
	if err != nil {
		t.Fatal(err)
	}
	valueRequest, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./model.js",
		"Point",
		"Point__from_model",
	)
	if err != nil {
		t.Fatal(err)
	}
	if typeRequest.Owner() != valueRequest.Owner() {
		t.Fatal("type and value requests for one binding have different owners")
	}
	for _, requests := range [][]api.RootRequest{
		{typeRequest, valueRequest},
		{valueRequest, typeRequest},
	} {
		placement := targetplacement.New()
		if err := placement.Apply(requests); err != nil {
			t.Fatal(err)
		}
		statements := placement.Statements(factory)
		if len(statements) != 1 {
			t.Fatalf("import declarations = %d, want one", len(statements))
		}
		declaration := statements[0].(tsgo.ImportDeclaration)
		if declaration.ImportClause().PhaseModifier() != 0 {
			t.Fatal("type-only request incorrectly dominated a value import")
		}
	}
	conflicting, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./model.js",
		"Point",
		"OtherPoint",
	)
	if err != nil {
		t.Fatal(err)
	}
	placement := targetplacement.New()
	if err := placement.Apply([]api.RootRequest{
		typeRequest,
		conflicting,
	}); err == nil {
		t.Fatal("one import binding accepted two local names")
	}
}

func TestNamespaceImportPlacementIsOneStaticBinding(t *testing.T) {
	factory := tsgo.NewFactory()
	typeRequest, err := api.NewNamespaceImportRequest(
		factory,
		api.ImportPhaseType,
		"@gotots/gostdlib/strings.js",
		"strings",
	)
	if err != nil {
		t.Fatal(err)
	}
	valueRequest, err := api.NewNamespaceImportRequest(
		factory,
		api.ImportPhaseValue,
		"@gotots/gostdlib/strings.js",
		"strings",
	)
	if err != nil {
		t.Fatal(err)
	}
	if typeRequest.Owner() != valueRequest.Owner() {
		t.Fatal("type and value namespace requests have different owners")
	}
	placement := targetplacement.New()
	if err := placement.Apply([]api.RootRequest{
		typeRequest,
		valueRequest,
	}); err != nil {
		t.Fatal(err)
	}
	statements := placement.Statements(factory)
	if len(statements) != 1 {
		t.Fatalf("namespace import declarations = %d, want one", len(statements))
	}
	declaration := statements[0].(tsgo.ImportDeclaration)
	clause := declaration.ImportClause()
	namespace, ok := clause.NamedBindings().(tsgo.NamespaceImport)
	if !ok ||
		clause.PhaseModifier() != 0 ||
		namespace.Name().Text() != "strings" {
		t.Fatalf("namespace import clause = %#v", clause)
	}
}

func TestPrimitiveAliasImportAvoidsSourceNamesAndRemainsOneTypedOwner(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/current", "current")
	packageScope := sourcePackage.Scope()
	int32Object := types.NewVar(
		token.Pos(1),
		sourcePackage,
		"int32",
		types.Typ[types.Int32],
	)
	reservedAlias := types.NewVar(
		token.Pos(2),
		sourcePackage,
		"int32__from_gotots_support",
		types.Typ[types.Int32],
	)
	packageScope.Insert(int32Object)
	packageScope.Insert(reservedAlias)
	owner := newNameOwner(packageScope, &types.Info{
		Defs: map[*ast.Ident]types.Object{
			{Name: "int32"}:                      int32Object,
			{Name: "int32__from_gotots_support"}: reservedAlias,
		},
	})
	names := testFileNames(
		t,
		owner,
		&ast.File{},
		packageScope,
		tsgo.NewFactory(),
		"modules/current/source.ts",
		nil,
	)

	first, err := names.Primitive(api.PrimitiveInt32)
	if err != nil {
		t.Fatal(err)
	}
	second, err := names.Primitive(api.PrimitiveInt32)
	if err != nil {
		t.Fatal(err)
	}
	const expectedLocal = "int32__from_gotots_support_1"
	if first.Name() != expectedLocal || second.Name() != expectedLocal {
		t.Fatalf("primitive names = %q, %q; want %q", first.Name(), second.Name(), expectedLocal)
	}
	requests := first.Requests()
	if len(requests) != 1 ||
		requests[0].ExportedName() != "int32" ||
		requests[0].LocalName() != expectedLocal ||
		requests[0].ModulePath() != "@gotots/runtime/scalars.js" {
		t.Fatalf("primitive request = %#v", requests)
	}
	alias, ok := requests[0].PrimitiveAlias()
	if !ok || alias != api.PrimitiveInt32 {
		t.Fatalf("primitive owner = %d, %v; want int32, true", alias, ok)
	}
}

func TestRuntimeImportAvoidsSourceNamesAndRemainsOneTypedOwner(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/current", "current")
	packageScope := sourcePackage.Scope()
	reservedExport := types.NewVar(
		token.Pos(1),
		sourcePackage,
		"goStringIndex",
		types.Typ[types.Int],
	)
	reservedAlias := types.NewVar(
		token.Pos(2),
		sourcePackage,
		"goStringIndex__from_gotots_runtime",
		types.Typ[types.Int],
	)
	packageScope.Insert(reservedExport)
	packageScope.Insert(reservedAlias)
	owner := newNameOwner(packageScope, &types.Info{
		Defs: map[*ast.Ident]types.Object{
			{Name: "goStringIndex"}:                      reservedExport,
			{Name: "goStringIndex__from_gotots_runtime"}: reservedAlias,
		},
	})
	names := testFileNames(
		t,
		owner,
		&ast.File{},
		packageScope,
		tsgo.NewFactory(),
		"modules/current/source.ts",
		nil,
	)

	first, err := names.Runtime(api.RuntimeStringIndex, api.ImportPhaseValue)
	if err != nil {
		t.Fatal(err)
	}
	second, err := names.Runtime(api.RuntimeStringIndex, api.ImportPhaseValue)
	if err != nil {
		t.Fatal(err)
	}
	const expectedLocal = "goStringIndex__from_gotots_runtime_1"
	if first.Name() != expectedLocal || second.Name() != expectedLocal {
		t.Fatalf("runtime names = %q, %q; want %q", first.Name(), second.Name(), expectedLocal)
	}
	requests := first.Requests()
	if len(requests) != 1 ||
		requests[0].ExportedName() != "goStringIndex" ||
		requests[0].LocalName() != expectedLocal ||
		requests[0].ModulePath() != "@gotots/runtime/string.js" {
		t.Fatalf("runtime request = %#v", requests)
	}
	symbol, ok := requests[0].RuntimeSymbol()
	if !ok || symbol != api.RuntimeStringIndex {
		t.Fatalf("runtime owner = %d, %v; want string-index, true", symbol, ok)
	}
}

func TestPlacementRuntimeSymbolsAreExactAndSorted(t *testing.T) {
	factory := tsgo.NewFactory()
	placement := targetplacement.New()
	for _, symbol := range []api.RuntimeSymbol{
		api.RuntimeArray,
		api.RuntimeStringSlice,
		api.RuntimeStringIndex,
		api.RuntimeStringSlice,
	} {
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			t.Fatal(err)
		}
		request, err := api.NewRuntimeImportRequest(
			factory,
			api.ImportPhaseValue,
			"../../"+contract.OutputPath(),
			symbol,
			contract.ExportedName(),
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := placement.Apply([]api.RootRequest{request}); err != nil {
			t.Fatal(err)
		}
	}
	actual := placement.RuntimeSymbols()
	expected := []api.RuntimeSymbol{
		api.RuntimeStringIndex,
		api.RuntimeStringSlice,
		api.RuntimeArray,
	}
	if !slices.Equal(actual, expected) {
		t.Fatalf("runtime symbols = %v, want %v", actual, expected)
	}
}
