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
