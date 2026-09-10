package emptystruct

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestEmptyAssignmentEvaluatesBothOperandsWithoutReplacingStorage(test *testing.T) {
	declaration := Build(tsgo.NewFactory(), "GoEmptyStruct")
	for _, member := range declaration.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok {
			continue
		}
		name, ok := method.Name().(tsgo.Identifier)
		if !ok || name.Text() != assignMember {
			continue
		}
		body, ok := method.Body().(tsgo.Block)
		if !ok || len(method.Parameters()) != 2 || len(body.Statements()) != 0 {
			test.Fatal("empty assignment must accept both evaluated operands without a payload write")
		}
		if method.Type().Kind() != tsgo.SyntaxKind(tsgo.KeywordTypeSyntaxKindVoidKeyword) {
			test.Fatal("empty assignment does not return void")
		}
		return
	}
	test.Fatal("empty assignment operation is absent")
}
