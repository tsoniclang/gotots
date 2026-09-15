package emptystruct

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestEmptyStorageKeepsTheValueBrandOutsideItsDataRecord(test *testing.T) {
	declaration := Build(tsgo.NewFactory(), "GoEmptyStruct")
	seen := map[string]bool{}
	for _, member := range declaration.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok {
			continue
		}
		name, ok := method.Name().(tsgo.Identifier)
		if !ok || name.Text() != storageOfMember && name.Text() != fromStorageMember {
			continue
		}
		seen[name.Text()] = true
		if len(method.Parameters()) != 1 {
			test.Fatal("empty storage conversion must evaluate exactly one source argument")
		}
		body, ok := method.Body().(tsgo.Block)
		if !ok || len(body.Statements()) != 1 {
			test.Fatal("empty storage conversion must have one closed return")
		}
		returned, ok := body.Statements()[0].(tsgo.ReturnStatement)
		if !ok {
			test.Fatal("empty storage conversion must return its exact result")
		}
		if name.Text() == storageOfMember {
			storage, ok := method.Type().(tsgo.TypeLiteralNode)
			if !ok || len(storage.Members()) != 0 {
				test.Fatal("empty storage must not retain the nominal value's type-only brands")
			}
			value, ok := returned.Expression().(tsgo.ObjectLiteralExpression)
			if !ok || len(value.Properties()) != 0 {
				test.Fatal("empty storage must construct a fieldless data record")
			}
		} else {
			storage, ok := method.Parameters()[0].Type().(tsgo.TypeLiteralNode)
			if !ok || len(storage.Members()) != 0 {
				test.Fatal("restoring an empty value must consume only its fieldless data record")
			}
			if _, ok := returned.Expression().(tsgo.NewExpression); !ok {
				test.Fatal("restoring an empty value must construct the nominal facade")
			}
		}
	}
	if len(seen) != 2 {
		test.Fatal("both empty storage directions must be present")
	}
}

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
