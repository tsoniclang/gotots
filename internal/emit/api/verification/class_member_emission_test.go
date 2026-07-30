package api_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestClassMemberContributionIsTypedAndImmutable(t *testing.T) {
	factory := tsgo.Factory{}
	owner := types.NewTypeName(token.NoPos, nil, "Record", nil)
	member := factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier("Read"),
		nil,
		nil,
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.NumericLiteral("1", tsgo.TokenFlagsNone),
				),
			},
			true,
		),
	)
	members := []tsgo.ClassElement{member}
	emission, err := api.ClassMemberContributionEmission(owner, members, nil)
	if err != nil {
		t.Fatal(err)
	}
	members[0] = nil

	selectedOwner, selected, ok := emission.ClassMemberContribution()
	if !ok ||
		selectedOwner != owner ||
		len(selected) != 1 ||
		selected[0] != member ||
		emission.Disposition() !=
			api.DeclarationDispositionClassMemberContribution ||
		len(emission.Declarations()) != 0 {
		t.Fatal("class-member contribution lost its closed target shape")
	}
	selected[0] = nil
	_, reread, ok := emission.ClassMemberContribution()
	if !ok || len(reread) != 1 || reread[0] != member {
		t.Fatal("class-member contribution exposed its backing slice")
	}
}

func TestClassMemberContributionRejectsInvalidShape(t *testing.T) {
	factory := tsgo.Factory{}
	owner := types.NewTypeName(token.NoPos, nil, "Record", nil)
	cases := []struct {
		owner   *types.TypeName
		members []tsgo.ClassElement
	}{
		{nil, []tsgo.ClassElement{factory.MethodDeclaration(
			nil,
			nil,
			factory.Identifier("Read"),
			nil,
			nil,
			nil,
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
			nil,
		)}},
		{owner, nil},
		{owner, []tsgo.ClassElement{nil}},
	}
	for index, testCase := range cases {
		if _, err := api.ClassMemberContributionEmission(
			testCase.owner,
			testCase.members,
			nil,
		); err == nil {
			t.Fatalf("invalid case %d was admitted", index)
		}
	}
}
