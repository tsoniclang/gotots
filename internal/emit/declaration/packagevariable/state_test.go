package packagevariable

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestStateConstructionInitializesEveryExactField(t *testing.T) {
	factory := tsgo.Factory{}
	fields := []tsgo.PropertyDeclaration{
		factory.PropertyDeclaration(nil, factory.Identifier("Count"), nil,
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword), nil),
		factory.PropertyDeclaration(nil, factory.Identifier("Payload"), nil,
			factory.TypeReferenceNode(factory.Identifier("PayloadStorage"), nil), nil),
	}
	statements, err := StateDeclarations(factory, fields)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 3 {
		t.Fatalf("state declarations = %d, want class, live binding and initializer", len(statements))
	}
	class := statements[0].(tsgo.ClassDeclaration)
	var constructor tsgo.ConstructorDeclaration
	for _, member := range class.Members() {
		if selected, ok := member.(tsgo.ConstructorDeclaration); ok {
			constructor = selected
		}
	}
	if constructor == nil || len(constructor.Parameters()) != len(fields) {
		t.Fatal("state constructor does not cover the exact field bank")
	}
	assignments := constructor.Body().(tsgo.Block).Statements()
	if len(assignments) != len(fields) {
		t.Fatal("state constructor does not assign every field once")
	}
	for index, field := range fields {
		parameter := constructor.Parameters()[index]
		assignment := assignments[index].(tsgo.ExpressionStatement).Expression().(tsgo.BinaryExpression)
		target := assignment.Left().(tsgo.PropertyAccessExpression)
		if parameter.Type() != field.Type() || target.Expression().Kind() != tsgo.SyntaxKindThisKeyword ||
			target.Name().(tsgo.Identifier).Text() != field.Name().(tsgo.Identifier).Text() ||
			assignment.Right().(tsgo.Identifier).Text() != parameter.Name().(tsgo.Identifier).Text() ||
			assignment.OperatorToken().Kind() != tsgo.SyntaxKindEqualsToken {
			t.Fatalf("field %d lost its exact constructor assignment", index)
		}
	}
	binding := statements[1].(tsgo.VariableStatement).DeclarationList().(tsgo.VariableDeclarationList)
	if binding.Flags() != tsgo.NodeFlagsLet || len(binding.Declarations()) != 1 ||
		binding.Declarations()[0].Initializer() != nil || binding.Declarations()[0].Type() == nil {
		t.Fatal("state binding must be typed, live and unconstructed until package initialization")
	}
	initializer := statements[2].(tsgo.FunctionDeclaration)
	if initializer.Name().Text() != StateInitializerName || len(initializer.Parameters()) != len(fields) {
		t.Fatal("state initializer has the wrong exact constructor contract")
	}
	publication := initializer.Body().(tsgo.Block).Statements()[0].(tsgo.ExpressionStatement).Expression().(tsgo.BinaryExpression)
	construction := publication.Right().(tsgo.NewExpression)
	if publication.Left().(tsgo.Identifier).Text() != StateValueName ||
		construction.Expression().(tsgo.Identifier).Text() != StateClassName ||
		len(construction.Arguments()) != len(fields) {
		t.Fatal("state initialization must publish exactly one fully constructed instance")
	}
	for index, argument := range construction.Arguments() {
		parameter := initializer.Parameters()[index]
		if parameter.Type() != fields[index].Type() || argument.(tsgo.Identifier).Text() != parameter.Name().(tsgo.Identifier).Text() {
			t.Fatalf("state initializer argument %d lost its exact field correspondence", index)
		}
	}
}

func TestStateConstructionRejectsIncompleteStorage(t *testing.T) {
	factory := tsgo.Factory{}
	valueType := factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
	valid := factory.PropertyDeclaration(nil, factory.Identifier("Value"), nil, valueType, nil)
	cases := map[string][]tsgo.PropertyDeclaration{
		"empty":         nil,
		"missing field": {nil},
		"missing type":  {factory.PropertyDeclaration(nil, factory.Identifier("Value"), nil, nil, nil)},
		"duplicate":     {valid, valid},
		"ambient":       {factory.PropertyDeclaration([]tsgo.ModifierLike{factory.DeclareKeyword()}, factory.Identifier("Value"), nil, valueType, nil)},
		"optional":      {factory.PropertyDeclaration(nil, factory.Identifier("Value"), factory.QuestionToken(), valueType, nil)},
		"computed":      {factory.PropertyDeclaration(nil, factory.ComputedPropertyName(factory.Identifier("key")), nil, valueType, nil)},
	}
	for name, fields := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := StateDeclarations(factory, fields); err == nil {
				t.Fatal("invalid package storage was admitted")
			}
		})
	}
}
