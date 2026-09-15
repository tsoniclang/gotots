package emptystruct

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestEmptyStorageKeepsTheValueBrandOutsideItsDataRecord(test *testing.T) {
	declaration, err := Build(tsgo.NewFactory(), "GoEmptyStruct")
	if err != nil {
		test.Fatal(err)
	}
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
	declaration, err := Build(tsgo.NewFactory(), "GoEmptyStruct")
	if err != nil {
		test.Fatal(err)
	}
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

func TestEmptyValueDeclaresBothExactStorageFacets(test *testing.T) {
	declaration, err := Build(tsgo.NewFactory(), "GoEmptyStruct")
	if err != nil {
		test.Fatal(err)
	}
	remaining := map[string]bool{}
	for _, symbol := range []api.RuntimeSymbol{api.RuntimeStorageTypeToken, api.RuntimeContainerStorageToken} {
		contract, contractErr := api.RuntimeContract(symbol)
		if contractErr != nil {
			test.Fatal(contractErr)
		}
		remaining[contract.ExportedName()] = true
	}
	for _, member := range declaration.Members() {
		property, ok := member.(tsgo.PropertyDeclaration)
		if !ok {
			continue
		}
		computed, ok := property.Name().(tsgo.ComputedPropertyName)
		if !ok {
			continue
		}
		name, ok := computed.Expression().(tsgo.Identifier)
		if !ok || !remaining[name.Text()] {
			test.Fatal("empty value has an unexpected or duplicated storage token")
		}
		delete(remaining, name.Text())
		storage, ok := property.Type().(tsgo.TypeLiteralNode)
		if !ok || len(storage.Members()) != 0 || property.Initializer() != nil {
			test.Fatal("storage facet must describe the empty type without runtime initialization")
		}
		modifiers := property.Modifiers()
		if len(modifiers) != 2 || modifiers[0].Kind() != tsgo.SyntaxKind(tsgo.ModifierSyntaxKindDeclareKeyword) ||
			modifiers[1].Kind() != tsgo.SyntaxKind(tsgo.ModifierSyntaxKindReadonlyKeyword) {
			test.Fatal("storage facet must be a declared readonly property")
		}
	}
	if len(remaining) != 0 {
		test.Fatal("empty value is missing a storage facet")
	}
}
