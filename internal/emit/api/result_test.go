package api_test

import (
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSetterStoreTargetOwnsTypedImmutableArguments(t *testing.T) {
	factory := tsgo.NewFactory()
	receiver := api.DirectExpression(factory.Identifier("values"))
	arguments := []api.ExpressionEmission{api.DirectExpression(
		factory.NumericLiteral("1", tsgo.TokenFlagsNone),
	)}
	target, err := api.NewSetterStoreTargetEmission(
		receiver,
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
	if !target.IsSetter() ||
		target.SetterReceiver().Value().(tsgo.Identifier).Text() != "values" ||
		target.SetterMember() != "set" ||
		target.SetterArguments()[0].Value().(tsgo.NumericLiteral).Text() != "1" {
		t.Fatalf("setter target leaked mutable input: %#v", target)
	}
	exposed := target.SetterArguments()
	exposed[0] = api.DirectExpression(
		factory.NumericLiteral("3", tsgo.TokenFlagsNone),
	)
	if target.SetterArguments()[0].Value().(tsgo.NumericLiteral).Text() != "1" {
		t.Fatal("setter target exposed mutable argument backing")
	}
}

func TestOrderedStoreTargetOwnsTypedImmutablePrerequisites(t *testing.T) {
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
	target, err := api.NewOrderedStoreTargetEmission(
		before,
		factory.Identifier("value"),
		types.Typ[types.Int32],
		requests,
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
	if target.Requests()[0].ExportedName() != "cell" {
		t.Fatal("store target leaked mutable request input")
	}

	exposed := target.Before()
	exposed[0] = factory.ExpressionStatement(factory.Identifier("alsoMutated"))
	if name := target.Before()[0].(tsgo.ExpressionStatement).
		Expression().(tsgo.Identifier).Text(); name != "prepare" {
		t.Fatalf("store prerequisite after accessor mutation = %q, want prepare", name)
	}
}

func TestOrderedStoreTargetRejectsNilPrerequisite(t *testing.T) {
	_, err := api.NewOrderedStoreTargetEmission(
		[]tsgo.Statement{nil},
		tsgo.NewFactory().Identifier("value"),
		types.Typ[types.Int32],
		nil,
	)
	if err == nil {
		t.Fatal("nil store prerequisite was accepted")
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
		"../../../support/scalars.js",
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
		"../../../support/scalars.js",
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
	if request.Specifier().Name().Text() != "flip" {
		t.Fatalf("local name = %q, want flip", request.Specifier().Name().Text())
	}
}

func TestExpressionEmissionRejectsMissingValue(t *testing.T) {
	_, err := api.NewExpressionEmission(nil, nil, nil)
	if err == nil {
		t.Fatal("missing target expression was accepted")
	}
}

func TestExpressionForInitializerUsesPinnedTargetCategory(t *testing.T) {
	factory := tsgo.NewFactory()
	result, err := api.ExpressionForInitializer(factory.Identifier("value"))
	if err != nil {
		t.Fatal(err)
	}
	identifier, ok := result.Value().(tsgo.Identifier)
	if !ok || identifier.Text() != "value" {
		t.Fatalf("for initializer = %T, want value identifier", result.Value())
	}
}
