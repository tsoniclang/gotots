package typescriptclass

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDeclarationOwnsOneRootPromiseAssimilationExclusion(t *testing.T) {
	factory := tsgo.NewFactory()
	root := Declaration(
		factory,
		nil,
		factory.Identifier("Root"),
		nil,
		nil,
		[]tsgo.ClassElement{factory.ConstructorDeclaration(
			nil,
			nil,
			nil,
			nil,
			factory.Block(nil, true),
		)},
	)
	assertPromiseAssimilationExclusion(t, root, 1)
	rebuilt := Declaration(
		factory,
		root.Modifiers(),
		root.Name(),
		root.TypeParameters(),
		root.HeritageClauses(),
		root.Members(),
	)
	assertPromiseAssimilationExclusion(t, rebuilt, 1)

	derived := Declaration(
		factory,
		nil,
		factory.Identifier("Derived"),
		nil,
		[]tsgo.HeritageClause{factory.HeritageClause(
			tsgo.HeritageClauseTokenKindExtendsKeyword,
			[]tsgo.ExpressionWithTypeArguments{factory.ExpressionWithTypeArguments(
				factory.Identifier("Root"),
				nil,
			)},
		)},
		nil,
	)
	assertPromiseAssimilationExclusion(t, derived, 0)
}

func assertPromiseAssimilationExclusion(
	t *testing.T,
	declaration tsgo.ClassDeclaration,
	want int,
) {
	t.Helper()
	found := 0
	for _, member := range declaration.Members() {
		property, ok := member.(tsgo.PropertyDeclaration)
		if !ok {
			continue
		}
		name, ok := property.Name().(tsgo.Identifier)
		if !ok || name.Text() != PromiseAssimilationMember {
			continue
		}
		found++
		modifiers := property.Modifiers()
		if len(modifiers) != 3 ||
			modifiers[0].Kind() != tsgo.SyntaxKindDeclareKeyword ||
			modifiers[1].Kind() != tsgo.SyntaxKindPrivateKeyword ||
			modifiers[2].Kind() != tsgo.SyntaxKindReadonlyKeyword ||
			property.PostfixToken() == nil ||
			property.PostfixToken().Kind() != tsgo.SyntaxKindQuestionToken ||
			property.Type().Kind() != tsgo.SyntaxKindNeverKeyword ||
			property.Initializer() != nil {
			t.Fatalf("promise assimilation exclusion = %#v", property)
		}
	}
	if found != want {
		t.Fatalf("promise assimilation exclusions = %d, want %d", found, want)
	}
}
