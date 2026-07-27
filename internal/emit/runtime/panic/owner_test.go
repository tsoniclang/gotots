package panicruntime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildCreatesOneTypedPanicCarrier(t *testing.T) {
	class := Build(tsgo.NewFactory(), "GoPanic").(tsgo.ClassDeclaration)
	if class.Name().Text() != "GoPanic" ||
		len(class.TypeParameters()) != 1 ||
		len(class.Members()) != 2 {
		t.Fatalf(
			"panic class = %q, type parameters %d, members %d",
			class.Name().Text(),
			len(class.TypeParameters()),
			len(class.Members()),
		)
	}
	constructor := class.Members()[0].(tsgo.ConstructorDeclaration)
	if len(constructor.Modifiers()) != 1 ||
		constructor.Modifiers()[0].Kind() != tsgo.SyntaxKindPrivateKeyword ||
		len(constructor.Parameters()) != 1 {
		t.Fatal("panic carrier constructor is not private and typed")
	}
	raise := class.Members()[1].(tsgo.MethodDeclaration)
	if raise.Name().(tsgo.Identifier).Text() != RaiseName ||
		len(raise.TypeParameters()) != 1 ||
		raise.Type().Kind() != tsgo.SyntaxKindNeverKeyword {
		t.Fatal("panic carrier lacks one generic static never-returning raise method")
	}
	body := raise.Body().(tsgo.Block).Statements()
	if len(body) != 1 || body[0].Kind() != tsgo.SyntaxKindThrowStatement {
		t.Fatal("panic raise path does not throw exactly once")
	}
}

func TestCallUsesTheClosedRaiseMember(t *testing.T) {
	factory := tsgo.NewFactory()
	call := Call(
		factory,
		"GoPanic",
		factory.StringLiteral("boom", tsgo.TokenFlagsNone),
	)
	member := call.Expression().(tsgo.PropertyAccessExpression)
	if member.Expression().(tsgo.Identifier).Text() != "GoPanic" ||
		member.Name().(tsgo.Identifier).Text() != RaiseName {
		t.Fatal("panic call bypasses the closed runtime member")
	}
}
