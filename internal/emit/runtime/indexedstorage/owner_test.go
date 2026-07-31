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
		t.Fatalf("dense-index members = %d, want two", len(members))
	}
	present := members[0].(tsgo.MethodDeclaration)
	if present.Name().(tsgo.Identifier).Text() != "present" {
		t.Fatalf("presence owner = %v, want present", present.Name())
	}
	if _, ok := present.Type().(tsgo.TypePredicateNode); !ok {
		t.Fatalf("presence result = %T, want TypePredicateNode", present.Type())
	}
	presentReturn := present.Body().(tsgo.Block).
		Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.BinaryExpression)
	if presentReturn.OperatorToken().Kind() != tsgo.SyntaxKindInKeyword {
		t.Fatalf(
			"presence operator = %v, want in",
			presentReturn.OperatorToken().Kind(),
		)
	}

	get := members[1].(tsgo.MethodDeclaration)
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
	condition := statements[1].(tsgo.IfStatement).Expression()
	if _, ok := condition.(tsgo.PrefixUnaryExpression); !ok {
		t.Fatalf("dense get check = %T, want checked negated predicate", condition)
	}
	if _, ok := statements[2].(tsgo.ReturnStatement).
		Expression().(tsgo.Identifier); !ok {
		t.Fatal("dense get does not return the narrowed value")
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
