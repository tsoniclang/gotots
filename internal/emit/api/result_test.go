package api_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
	requests := []api.PlacementRequest{request}
	result, err := api.NewExpressionEmission(
		before,
		factory.Identifier("value"),
		requests,
	)
	if err != nil {
		t.Fatal(err)
	}

	before[0] = factory.ExpressionStatement(factory.Identifier("mutated"))
	requests[0] = api.PlacementRequest{}
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
	exposedRequests[0] = api.PlacementRequest{}
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

	if request.Kind() != api.PlacementImport ||
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
