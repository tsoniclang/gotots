package typescriptclass

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const PromiseAssimilationMember = "then"

func Declaration(
	factory tsgo.Factory,
	modifiers []tsgo.ModifierLike,
	name tsgo.Identifier,
	typeParameters []tsgo.TypeParameterDeclaration,
	heritageClauses []tsgo.HeritageClause,
	members []tsgo.ClassElement,
) tsgo.ClassDeclaration {
	ownedMembers := slices.Clone(members)
	if !hasBaseClass(heritageClauses) &&
		!hasPromiseAssimilationExclusion(ownedMembers) {
		ownedMembers = append(
			ownedMembers,
			promiseAssimilationExclusion(factory),
		)
	}
	return factory.ClassDeclaration(
		modifiers,
		name,
		typeParameters,
		heritageClauses,
		ownedMembers,
	)
}

func hasPromiseAssimilationExclusion(members []tsgo.ClassElement) bool {
	for _, member := range members {
		property, ok := member.(tsgo.PropertyDeclaration)
		if !ok || !isPromiseAssimilationExclusion(property) {
			continue
		}
		return true
	}
	return false
}

func isPromiseAssimilationExclusion(
	property tsgo.PropertyDeclaration,
) bool {
	name, ok := property.Name().(tsgo.Identifier)
	if !ok || name.Text() != PromiseAssimilationMember ||
		property.PostfixToken() == nil ||
		property.PostfixToken().Kind() != tsgo.SyntaxKindQuestionToken ||
		property.Type().Kind() != tsgo.SyntaxKindNeverKeyword ||
		property.Initializer() != nil {
		return false
	}
	modifiers := property.Modifiers()
	return len(modifiers) == 3 &&
		modifiers[0].Kind() == tsgo.SyntaxKindDeclareKeyword &&
		modifiers[1].Kind() == tsgo.SyntaxKindPrivateKeyword &&
		modifiers[2].Kind() == tsgo.SyntaxKindReadonlyKeyword
}

func hasBaseClass(heritageClauses []tsgo.HeritageClause) bool {
	for _, clause := range heritageClauses {
		if clause.Token() == tsgo.HeritageClauseTokenKindExtendsKeyword {
			return true
		}
	}
	return false
}

func promiseAssimilationExclusion(
	factory tsgo.Factory,
) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			factory.DeclareKeyword(),
			factory.PrivateKeyword(),
			factory.ReadonlyKeyword(),
		},
		factory.Identifier(PromiseAssimilationMember),
		factory.QuestionToken(),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNeverKeyword),
		nil,
	)
}
