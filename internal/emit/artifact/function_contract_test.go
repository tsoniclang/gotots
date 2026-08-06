package artifact

import (
	"bytes"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestObservableFunctionContractIgnoresBody(t *testing.T) {
	factory := tsgo.NewFactory()
	first := artifactTestFunction(factory, "value", nil)
	second := artifactTestFunction(
		factory,
		"value",
		[]tsgo.Statement{factory.ReturnStatement(
			factory.NumericLiteral("1", tsgo.TokenFlagsNone),
		)},
	)
	firstContract, err := ProjectContract(factory, []tsgo.Statement{first})
	if err != nil {
		t.Fatal(err)
	}
	secondContract, err := ProjectContract(factory, []tsgo.Statement{second})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(
		artifactFacetBytes(firstContract, api.ArtifactFacetCallableSignature),
		artifactFacetBytes(secondContract, api.ArtifactFacetCallableSignature),
	) {
		t.Fatal("function body changed the observable callable signature")
	}

	changed := factory.FunctionDeclaration(
		first.Modifiers(),
		nil,
		first.Name(),
		nil,
		[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
			nil,
			nil,
			factory.Identifier("flag"),
			nil,
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
			nil,
		)},
		first.Type(),
		first.Body(),
	)
	changedContract, err := ProjectContract(
		factory,
		[]tsgo.Statement{changed},
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(
		artifactFacetBytes(firstContract, api.ArtifactFacetCallableSignature),
		artifactFacetBytes(changedContract, api.ArtifactFacetCallableSignature),
	) {
		t.Fatal("parameter change was absent from the callable signature")
	}
}

func TestObservableFunctionContractSeparatesTopLevelRegistration(t *testing.T) {
	factory := tsgo.NewFactory()
	function := factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier("F"),
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block(nil, true),
	)
	registration := factory.ExpressionStatement(factory.CallExpression(
		factory.Identifier("register"),
		nil,
		nil,
		[]tsgo.Expression{factory.Identifier("F")},
		tsgo.NodeFlagsNone,
	))
	contract, err := ProjectContract(
		factory,
		[]tsgo.Statement{function, registration},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !contract.hasFacet(api.ArtifactFacetCallableSignature) {
		t.Fatal("callable signature facet is absent")
	}
	if !contract.hasFacet(api.ArtifactFacetImplementation) {
		t.Fatal("top-level registration implementation facet is absent")
	}
}
