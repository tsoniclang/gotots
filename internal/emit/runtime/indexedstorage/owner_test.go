package indexedstorage_test

import (
	"testing"

	indexedstorage "github.com/tsoniclang/gotots/internal/emit/runtime/indexedstorage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestDenseIndexUsesCheckedPresenceNarrowing(t *testing.T) {
	factory := tsgo.NewFactory()
	class := indexedstorage.Build(
		factory,
		"GoDenseIndex",
		"GoPanic",
	)
	members := class.Members()
	if len(members) != 2 {
		t.Fatalf("dense-index members = %d, want get and Promise exclusion", len(members))
	}
	get := members[0].(tsgo.MethodDeclaration)
	statements := get.Body().(tsgo.Block).Statements()
	if len(statements) != 3 {
		t.Fatalf("dense get statements = %d, want value/check/return", len(statements))
	}
	value := statements[0].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0].
		Initializer()
	if _, ok := value.(tsgo.ElementAccessExpression); !ok {
		t.Fatalf("dense get value = %T, want ElementAccessExpression", value)
	}
	condition := statements[1].(tsgo.IfStatement).
		Expression().(tsgo.PrefixUnaryExpression).
		Operand().(tsgo.BinaryExpression)
	if condition.OperatorToken().Kind() != tsgo.SyntaxKindInKeyword {
		t.Fatalf("dense get check = %v, want direct in guard", condition.OperatorToken().Kind())
	}
	if _, ok := statements[2].(tsgo.ReturnStatement).
		Expression().(tsgo.AsExpression); !ok {
		t.Fatal("dense get does not return the presence-checked value")
	}
}

func TestDenseIndexReferenceIsOneStaticCall(t *testing.T) {
	factory := tsgo.NewFactory()
	call := indexedstorage.Element(
		factory,
		"GoDenseIndex",
		factory.Identifier("values"),
		factory.Identifier("index"),
	)
	property, ok := call.Expression().(tsgo.PropertyAccessExpression)
	if !ok ||
		property.Expression().(tsgo.Identifier).Text() != "GoDenseIndex" ||
		property.Name().(tsgo.Identifier).Text() != "get" ||
		len(call.Arguments()) != 2 {
		t.Fatalf("dense index reference = %#v", call)
	}
}
