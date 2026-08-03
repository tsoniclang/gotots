package sequence

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestTargetSequenceFallthrough(t *testing.T) {
	factory := tsgo.Factory{}
	returnStatement := factory.ReturnStatement(nil)
	expressionStatement := factory.ExpressionStatement(factory.TrueLiteral())
	if FallsThrough([]tsgo.Statement{
		expressionStatement,
		returnStatement,
		expressionStatement,
	}) {
		t.Fatal("a sequence continued past a return statement")
	}
	if FallsThrough([]tsgo.Statement{
		factory.Block([]tsgo.Statement{returnStatement}, true),
	}) {
		t.Fatal("a terminal block fell through")
	}
	if FallsThrough([]tsgo.Statement{
		factory.LabeledStatement(
			factory.Identifier("target"),
			factory.ContinueStatement(factory.Identifier("dispatch")),
		),
	}) {
		t.Fatal("a labeled continue fell through")
	}
	terminalIf := factory.IfStatement(
		factory.TrueLiteral(),
		returnStatement,
		factory.ThrowStatement(
			factory.StringLiteral("failure", tsgo.TokenFlagsNone),
		),
	)
	if FallsThrough([]tsgo.Statement{terminalIf}) {
		t.Fatal("an if with two terminal branches fell through")
	}
	partialIf := factory.IfStatement(
		factory.TrueLiteral(),
		returnStatement,
		nil,
	)
	if !FallsThrough([]tsgo.Statement{partialIf}) {
		t.Fatal("an if without an else was classified as terminal")
	}
	if FallsThrough([]tsgo.Statement{
		factory.ForStatement(nil, nil, nil, factory.Block(nil, true)),
	}) {
		t.Fatal("a conditionless for statement fell through")
	}
}
