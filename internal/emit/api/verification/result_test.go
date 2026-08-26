package api_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type accessorContextServices struct {
	api.Names
	api.Values
}

func TestAccessorReadPreservesReceiverPrerequisitesAndRequests(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./box.js",
		"Box",
		"Box",
	)
	if err != nil {
		t.Fatal(err)
	}
	receiver, err := api.NewExpressionEmission(
		[]tsgo.Statement{
			factory.ExpressionStatement(factory.Identifier("prepare")),
		},
		factory.Identifier("Box"),
		[]api.RootRequest{request},
	)
	if err != nil {
		t.Fatal(err)
	}
	target, err := api.NewAccessorStoreTargetEmission(
		receiver,
		"$get",
		"$set",
		nil,
		types.Typ[types.Int32],
	)
	if err != nil {
		t.Fatal(err)
	}
	services := &accessorContextServices{}
	context, err := api.NewContext(
		api.RoleAssignmentTarget,
		token.NewFileSet(),
		types.NewPackage("example.com/accessor", "accessor"),
		&types.Info{},
		types.SizesFor("gc", "amd64"),
		api.MemoryByteOrderLittleEndian,
		factory,
		services,
		services,
		api.IntegerRepresentationNumber,
		api.EvaluationOrderDirect,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := target.ReadValue(context, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Before()) != 1 ||
		result.Before()[0].(tsgo.ExpressionStatement).
			Expression().(tsgo.Identifier).Text() != "prepare" ||
		len(result.Requests()) != 1 ||
		result.Requests()[0].ExportedName() != "Box" {
		t.Fatalf(
			"accessor read prerequisites = %d, requests = %d",
			len(result.Before()),
			len(result.Requests()),
		)
	}
}

func TestAccessorStoreTargetOwnsTypedImmutableArguments(t *testing.T) {
	factory := tsgo.NewFactory()
	receiver := api.DirectExpression(factory.Identifier("values"))
	arguments := []api.ExpressionEmission{api.DirectExpression(
		factory.NumericLiteral("1", tsgo.TokenFlagsNone),
	)}
	target, err := api.NewAccessorStoreTargetEmission(
		receiver,
		"get",
		"set",
		arguments,
		types.Typ[types.Int32],
	)
	if err != nil {
		t.Fatal(err)
	}
	arguments[0] = api.DirectExpression(
		factory.NumericLiteral("2", tsgo.TokenFlagsNone),
	)
	if !target.IsAccessor() ||
		target.CopiesValue() ||
		target.AccessorReceiver().Value().(tsgo.Identifier).Text() != "values" ||
		target.GetterMember() != "get" ||
		target.SetterMember() != "set" ||
		target.AccessorArguments()[0].Value().(tsgo.NumericLiteral).Text() != "1" {
		t.Fatalf("accessor target leaked mutable input: %#v", target)
	}
	exposed := target.AccessorArguments()
	exposed[0] = api.DirectExpression(
		factory.NumericLiteral("3", tsgo.TokenFlagsNone),
	)
	if target.AccessorArguments()[0].Value().(tsgo.NumericLiteral).Text() != "1" {
		t.Fatal("accessor target exposed mutable argument backing")
	}
}

func TestCopyingAccessorOwnsTheValueCopyBoundary(t *testing.T) {
	factory := tsgo.NewFactory()
	target, err := api.NewCopyingAccessorStoreTargetEmission(
		api.DirectExpression(factory.Identifier("values")),
		"get",
		"set",
		[]api.ExpressionEmission{
			api.DirectExpression(
				factory.NumericLiteral("1", tsgo.TokenFlagsNone),
			),
		},
		types.Typ[types.Int32],
	)
	if err != nil {
		t.Fatal(err)
	}
	if !target.IsAccessor() || !target.CopiesValue() {
		t.Fatalf("copying accessor ownership = %#v", target)
	}
}

func TestFunctionStoreTargetOwnsTypedImmutableOperations(t *testing.T) {
	factory := tsgo.NewFactory()
	getterRequest, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./pointer.js",
		"load",
		"load",
	)
	if err != nil {
		t.Fatal(err)
	}
	setterRequest, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./pointer.js",
		"store",
		"store",
	)
	if err != nil {
		t.Fatal(err)
	}
	arguments := []api.ExpressionEmission{
		api.DirectExpression(factory.Identifier("pointer")),
	}
	target, err := api.NewFunctionStoreTargetEmission(
		api.DirectExpression(factory.Identifier("load"), getterRequest),
		api.DirectExpression(factory.Identifier("store"), setterRequest),
		arguments,
		types.Typ[types.Int32],
	)
	if err != nil {
		t.Fatal(err)
	}
	arguments[0] = api.DirectExpression(factory.Identifier("mutated"))
	var requestNames []string
	if err := api.WalkRootRequests(
		target.Requests(),
		func(request api.RootRequest) error {
			requestNames = append(requestNames, request.ExportedName())
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if !target.IsAccessor() ||
		!target.Valid() ||
		target.AccessorArguments()[0].Value().(tsgo.Identifier).Text() !=
			"pointer" ||
		len(requestNames) != 2 ||
		requestNames[0] != "load" ||
		requestNames[1] != "store" {
		t.Fatalf("function store target = %#v", target)
	}
	exposed := target.AccessorArguments()
	exposed[0] = api.DirectExpression(factory.Identifier("alsoMutated"))
	if target.AccessorArguments()[0].Value().(tsgo.Identifier).Text() !=
		"pointer" {
		t.Fatal("function store target exposed argument backing")
	}
	if _, err := api.NewNamedReturnControl("return", []api.StoreTargetEmission{
		target,
	}); err != nil {
		t.Fatalf("named return rejected a valid function-backed store: %v", err)
	}
}

func TestFunctionStoreTargetRejectsOperationPrerequisites(t *testing.T) {
	factory := tsgo.NewFactory()
	getter, err := api.NewExpressionEmission(
		[]tsgo.Statement{
			factory.ExpressionStatement(factory.Identifier("prepare")),
		},
		factory.Identifier("load"),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = api.NewFunctionStoreTargetEmission(
		getter,
		api.DirectExpression(factory.Identifier("store")),
		nil,
		types.Typ[types.Int32],
	)
	if err == nil {
		t.Fatal("function store target accepted an unstable operation")
	}
}

func TestPropertyStoreTargetOwnsTypedImmutablePrerequisites(t *testing.T) {
	factory := tsgo.NewFactory()
	before := []tsgo.Statement{
		factory.ExpressionStatement(factory.Identifier("prepare")),
	}
	request, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./storage.js",
		"cell",
		"cell",
	)
	if err != nil {
		t.Fatal(err)
	}
	requests := []api.RootRequest{request}
	receiver, err := api.NewExpressionEmission(
		before,
		factory.Identifier("owner"),
		requests,
	)
	if err != nil {
		t.Fatal(err)
	}
	target, err := api.NewPropertyStoreTargetEmission(
		factory,
		receiver,
		"value",
		types.Typ[types.Int32],
	)
	if err != nil {
		t.Fatal(err)
	}

	before[0] = factory.ExpressionStatement(factory.Identifier("mutated"))
	requests[0] = api.RootRequest{}
	if name := target.Before()[0].(tsgo.ExpressionStatement).
		Expression().(tsgo.Identifier).Text(); name != "prepare" {
		t.Fatalf("store prerequisite = %q, want prepare", name)
	}
	if !target.IsProperty() ||
		target.Requests()[0].ExportedName() != "cell" {
		t.Fatal("store target leaked mutable request input")
	}

	exposed := target.Before()
	exposed[0] = factory.ExpressionStatement(factory.Identifier("alsoMutated"))
	if name := target.Before()[0].(tsgo.ExpressionStatement).
		Expression().(tsgo.Identifier).Text(); name != "prepare" {
		t.Fatalf("store prerequisite after accessor mutation = %q, want prepare", name)
	}
}

func TestPropertyStoreTargetRejectsEmptyMember(t *testing.T) {
	factory := tsgo.NewFactory()
	_, err := api.NewPropertyStoreTargetEmission(
		factory,
		api.DirectExpression(factory.Identifier("owner")),
		"",
		types.Typ[types.Int32],
	)
	if err == nil {
		t.Fatal("empty property member was accepted")
	}
}

func TestEmissionResultsOwnImmutableTargetNodesAndRequests(t *testing.T) {
	factory := tsgo.NewFactory()
	before := []tsgo.Statement{
		factory.ExpressionStatement(factory.Identifier("prepare")),
	}
	request, err := api.NewImportRequest(
		factory,
		api.ImportPhaseType,
		"../../../runtime/scalars.js",
		"int64",
		"int64",
	)
	if err != nil {
		t.Fatal(err)
	}
	requests := []api.RootRequest{request}
	result, err := api.NewExpressionEmission(
		before,
		factory.Identifier("value"),
		requests,
	)
	if err != nil {
		t.Fatal(err)
	}

	before[0] = factory.ExpressionStatement(factory.Identifier("mutated"))
	requests[0] = api.RootRequest{}
	if name := result.Before()[0].(tsgo.ExpressionStatement).
		Expression().(tsgo.Identifier).Text(); name != "prepare" {
		t.Fatalf("before statement = %q, want prepare", name)
	}
	if got := result.Requests()[0].ExportedName(); got != "int64" {
		t.Fatalf("request exported name = %q, want int64", got)
	}

	exposedBefore := result.Before()
	exposedBefore[0] = factory.ExpressionStatement(factory.Identifier("alsoMutated"))
	exposedRequests := result.Requests()
	exposedRequests[0] = api.RootRequest{}
	if name := result.Before()[0].(tsgo.ExpressionStatement).
		Expression().(tsgo.Identifier).Text(); name != "prepare" {
		t.Fatalf("before statement after accessor mutation = %q, want prepare", name)
	}
	if got := result.Requests()[0].ExportedName(); got != "int64" {
		t.Fatalf("request after accessor mutation = %q, want int64", got)
	}
}

func TestPrimitiveAliasRequestCarriesGeneratedSupportIdentity(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := api.NewPrimitiveAliasRequest(
		factory,
		"../../../runtime/scalars.js",
		api.PrimitiveInt64,
		"sourceInt64",
	)
	if err != nil {
		t.Fatal(err)
	}
	alias, ok := request.PrimitiveAlias()
	if !ok || alias != api.PrimitiveInt64 {
		t.Fatalf("primitive alias = %d, %v; want int64, true", alias, ok)
	}
	if request.ExportedName() != "int64" || request.LocalName() != "sourceInt64" {
		t.Fatalf(
			"primitive import = %s as %s, want int64 as sourceInt64",
			request.ExportedName(),
			request.LocalName(),
		)
	}
}

func TestImportRequestCarriesExactPlacementPolicy(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./logic.js",
		"flip",
		"flip",
	)
	if err != nil {
		t.Fatal(err)
	}

	if request.Kind() != api.RootRequestImport ||
		request.LegalScope() != api.ScopeFileImports ||
		request.PreferredScope() != api.ScopeFileImports ||
		request.Execution() != api.ExecutionStatic {
		t.Fatalf("request policy = %#v", request)
	}
	if request.ModuleSpecifier().Text() != "./logic.js" {
		t.Fatalf("module = %q, want ./logic.js", request.ModuleSpecifier().Text())
	}
	if request.ImportBinding() != api.ImportBindingNamed ||
		request.NamespaceSpecifier() != nil {
		t.Fatalf("named import binding = %d", request.ImportBinding())
	}
	if request.Specifier().Name().Text() != "flip" {
		t.Fatalf("local name = %q, want flip", request.Specifier().Name().Text())
	}
}

func TestNamespaceImportRequestCarriesExactPlacementPolicy(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := api.NewNamespaceImportRequest(
		factory,
		api.ImportPhaseValue,
		"@gotots/gostdlib/strings.js",
		"strings",
	)
	if err != nil {
		t.Fatal(err)
	}
	if request.Kind() != api.RootRequestImport ||
		request.ImportBinding() != api.ImportBindingNamespace ||
		request.ExportedName() != "" ||
		request.LocalName() != "strings" ||
		request.Specifier() != nil ||
		request.NamespaceSpecifier().Name().Text() != "strings" {
		t.Fatalf("namespace request = %#v", request)
	}
}

func TestSideEffectImportRequestCarriesExactPlacementPolicy(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := api.NewSideEffectImportRequest(factory, "./reflection-types.js")
	if err != nil {
		t.Fatal(err)
	}
	if request.Kind() != api.RootRequestImport ||
		request.ImportPhase() != api.ImportPhaseValue ||
		request.ImportBinding() != api.ImportBindingSideEffect ||
		request.ExportedName() != "" ||
		request.LocalName() != "" ||
		request.Specifier() != nil ||
		request.NamespaceSpecifier() != nil ||
		request.ModuleSpecifier().Text() != "./reflection-types.js" {
		t.Fatalf("side-effect request = %#v", request)
	}
}

func TestQualifiedNameReferenceBuildsTypedValueAndTypePaths(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := api.NewNamespaceImportRequest(
		factory,
		api.ImportPhaseValue,
		"@gotots/gostdlib/strings.js",
		"strings",
	)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := api.NewQualifiedNameReference(
		"strings",
		"Builder",
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	qualifier, qualified := reference.Qualifier()
	value := reference.Expression(factory).(tsgo.PropertyAccessExpression)
	entity := reference.EntityName(factory).(tsgo.QualifiedName)
	member, err := reference.MemberExpression(factory, "String")
	if err != nil {
		t.Fatal(err)
	}
	valueName, valueNameOK := value.Name().(tsgo.Identifier)
	memberName, memberNameOK := member.Name().(tsgo.Identifier)
	if !qualified ||
		qualifier != "strings" ||
		reference.Name() != "Builder" ||
		value.Expression().(tsgo.Identifier).Text() != "strings" ||
		!valueNameOK ||
		valueName.Text() != "Builder" ||
		entity.Left().(tsgo.Identifier).Text() != "strings" ||
		entity.Right().Text() != "Builder" ||
		!memberNameOK ||
		memberName.Text() != "String" ||
		len(reference.Requests()) != 1 {
		t.Fatalf("qualified reference = %#v", reference)
	}
}

func TestExpressionEmissionRejectsMissingValue(t *testing.T) {
	_, err := api.NewExpressionEmission(nil, nil, nil)
	if err == nil {
		t.Fatal("missing target expression was accepted")
	}
}
