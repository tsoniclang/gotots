package indexedstorage_test

import (
	"testing"

	indexedstorage "github.com/tsoniclang/gotots/internal/emit/runtime/indexedstorage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestElementInlinesPresenceCheckBeforeItsCheckedCast(t *testing.T) {
	factory := tsgo.NewFactory()
	value := indexedstorage.Element(
		factory,
		"GoPanic",
		factory.Identifier("values"),
		factory.Identifier("index"),
		factory.TypeReferenceNode(factory.Identifier("T"), nil),
	)
	conditional, ok := value.Expression().(tsgo.ConditionalExpression)
	if !ok {
		t.Fatalf("checked element = %T, want ConditionalExpression", value.Expression())
	}
	present, ok := conditional.Condition().(tsgo.BinaryExpression)
	if !ok || present.OperatorToken().Kind() != tsgo.SyntaxKindInKeyword {
		t.Fatalf("presence proof = %T, want indexed in check", conditional.Condition())
	}
	if conditional.WhenTrue().Kind() != tsgo.SyntaxKindElementAccessExpression ||
		conditional.WhenFalse().Kind() != tsgo.SyntaxKindCallExpression {
		t.Fatalf(
			"checked branches = %v/%v, want element/panic",
			conditional.WhenTrue().Kind(),
			conditional.WhenFalse().Kind(),
		)
	}
}

func TestElementMutationWithoutPresenceProofIsDetected(t *testing.T) {
	factory := tsgo.NewFactory()
	unchecked := factory.AsExpression(
		factory.ElementAccessExpression(
			factory.Identifier("values"),
			nil,
			factory.Identifier("index"),
			tsgo.NodeFlagsNone,
		),
		factory.TypeReferenceNode(factory.Identifier("T"), nil),
	)
	if _, ok := unchecked.Expression().(tsgo.ConditionalExpression); ok {
		t.Fatal("mutation control unexpectedly retained the presence proof")
	}
}
