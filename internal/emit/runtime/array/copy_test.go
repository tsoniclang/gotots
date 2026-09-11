package array

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArrayCopyUsesNativeWholeBackingAndBoundedWindow(test *testing.T) {
	factory := tsgo.NewFactory()
	body := copyBody(factory, "GoArray", factory.TypeReferenceNode(factory.Identifier("T"), nil), factory.TypeReferenceNode(factory.Identifier("N"), nil))
	if _, err := tsgo.EncodeNode(factory.Block(body, true)); err != nil {
		test.Fatal(err)
	}
	guard := body[0].(tsgo.IfStatement)
	result := guard.ThenStatement().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression().(tsgo.NewExpression)
	region := result.Arguments()[0].(tsgo.ObjectLiteralExpression)
	call := region.Properties()[1].(tsgo.PropertyAssignment).Initializer().(tsgo.CallExpression)
	operation := call.Expression().(tsgo.PropertyAccessExpression)
	if operation.Expression().(tsgo.Identifier).Text() != "Array" || operation.Name().(tsgo.Identifier).Text() != "from" {
		test.Fatal("whole backing lost its native bulk copy")
	}
	declaration := body[1].(tsgo.VariableStatement).DeclarationList().Declarations()[0]
	if declaration.Initializer().Kind() != tsgo.SyntaxKindArrayLiteralExpression {
		test.Fatal("window copy traverses its entire retained allocation")
	}
	loop := body[2].(tsgo.ForStatement)
	bound := loop.Condition().(tsgo.BinaryExpression).Right().(tsgo.PropertyAccessExpression)
	if bound.Expression().Kind() != tsgo.SyntaxKindThisKeyword || bound.Name().(tsgo.Identifier).Text() != "length" {
		test.Fatal("window copy is not bounded by its own extent")
	}
}
