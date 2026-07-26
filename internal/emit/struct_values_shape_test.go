package emit_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNamedStructValuesConstructExactTargetShape(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	for _, name := range []string{"Point", "Box", "Mirror", "Reserved"} {
		class := targetClass(t, source, name)
		if len(class.TypeParameters()) != 0 || len(class.HeritageClauses()) != 0 {
			t.Fatalf("%s has type parameters or heritage clauses", name)
		}
		members := class.Members()
		if len(members) != 6 {
			t.Fatalf("%s members = %d, want brand, constructor, and four operations", name, len(members))
		}
		assertErasedBrand(t, name, members[0])
		if _, ok := members[1].(tsgo.ConstructorDeclaration); !ok {
			t.Fatalf("%s member 1 = %T, want constructor", name, members[1])
		}
		for index, operation := range []string{"$zero", "$copy", "$assign", "$equal"} {
			method, ok := members[index+2].(tsgo.MethodDeclaration)
			if !ok {
				t.Fatalf("%s member %d = %T, want static method", name, index+2, members[index+2])
			}
			if targetName(method.Name()) != operation ||
				!hasModifier(method.Modifiers(), tsgo.SyntaxKindStaticKeyword) {
				t.Fatalf("%s operation %d is not static %s", name, index, operation)
			}
		}
	}

	reserved := targetClass(t, source, "Reserved")
	constructor := reserved.Members()[1].(tsgo.ConstructorDeclaration)
	if got := targetName(constructor.Parameters()[0].Name()); got != "__go_constructor" {
		t.Fatalf("reserved field name = %q, want collision-safe target name", got)
	}
}

func TestNamedStructValuesPreserveConstructionOrderAndOwnership(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	newBox := targetFunction(t, source, "NewBox")
	statements := newBox.Body().(tsgo.Block).Statements()
	if len(statements) != 5 {
		t.Fatalf("NewBox statements = %d, want four captures and return", len(statements))
	}
	captures := make([]tsgo.VariableDeclaration, 4)
	for index := range captures {
		statement, ok := statements[index].(tsgo.VariableStatement)
		if !ok {
			t.Fatalf("NewBox statement %d = %T, want capture", index, statements[index])
		}
		captures[index] = statement.DeclarationList().Declarations()[0]
	}
	if _, ok := captures[0].Initializer().(tsgo.BinaryExpression); !ok {
		t.Fatalf("first source initializer = %T, want Active comparison", captures[0].Initializer())
	}
	visible := captures[1].Initializer().(tsgo.AsExpression).Expression()
	if visible.Kind() != tsgo.SyntaxKindTrueKeyword {
		t.Fatalf("second source initializer kind = %d, want Visible true", visible.Kind())
	}
	if targetName(captures[2].Initializer()) != "value" {
		t.Fatal("third source initializer is not Point.X value")
	}
	point := captures[3].Initializer().(tsgo.NewExpression)
	if targetName(point.Expression()) != "Point" ||
		targetName(point.Arguments()[0]) != targetName(captures[2].Name()) ||
		targetName(point.Arguments()[1]) != targetName(captures[1].Name()) {
		t.Fatal("nested Point construction does not consume source-ordered captures in field order")
	}
	result := statements[4].(tsgo.ReturnStatement).Expression().(tsgo.NewExpression)
	if targetName(result.Expression()) != "Box" ||
		targetName(result.Arguments()[0]) != targetName(captures[3].Name()) ||
		targetName(result.Arguments()[1]) != targetName(captures[0].Name()) {
		t.Fatal("Box construction does not consume source-ordered captures in field order")
	}

	copyResult := targetFunction(t, source, "CopyResult").
		Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
	if targetName(copyResult.Expression()) != "CopyIsolated" {
		t.Fatal("fresh call result was wrapped instead of transferred")
	}
	if _, ok := copyResult.Arguments()[0].(tsgo.CallExpression); !ok {
		t.Fatal("fresh NewBox argument was not transferred directly")
	}
}

func TestNamedStructValuesUseStaticReceiverAndValueOperations(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	invoke := targetFunction(t, source, "Invoke")
	call := invoke.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
	if targetName(call.Expression()) != "Box_WithX" {
		t.Fatalf("receiver call target = %q, want exact named function", targetName(call.Expression()))
	}
	copyCall := call.Arguments()[0].(tsgo.CallExpression)
	if receiver, member := targetProperty(copyCall.Expression()); receiver != "Box" || member != "$copy" {
		t.Fatalf("receiver boundary = %s.%s, want Box.$copy", receiver, member)
	}

	method := targetFunction(t, source, "Box_WithX")
	if len(method.Parameters()) != 2 || targetName(method.Parameters()[0].Name()) != "box" {
		t.Fatal("value receiver was not emitted as the first explicit parameter")
	}
	methodStatements := method.Body().(tsgo.Block).Statements()
	if len(methodStatements) != 2 {
		t.Fatalf("receiver body statements = %d, want source store and return only", len(methodStatements))
	}
	if _, ok := methodStatements[0].(tsgo.ExpressionStatement); !ok {
		t.Fatal("receiver body gained a wrapper or copy prologue")
	}

	assign := targetFunction(t, source, "AssignIsolated")
	assignCall := assign.Body().(tsgo.Block).Statements()[1].(tsgo.ExpressionStatement).Expression().(tsgo.CallExpression)
	if receiver, member := targetProperty(assignCall.Expression()); receiver != "Box" || member != "$assign" {
		t.Fatalf("assignment boundary = %s.%s, want destination-preserving Box.$assign", receiver, member)
	}

	parameter := targetFunction(t, source, "ParameterIsolated")
	initializer := parameter.Body().(tsgo.Block).Statements()[0].(tsgo.VariableStatement).DeclarationList().Declarations()[0].
		Initializer().(tsgo.CallExpression)
	if targetName(initializer.Expression()) != "MutateParameter" {
		t.Fatal("parameter isolation does not call the selected function")
	}
	argumentCopy := initializer.Arguments()[0].(tsgo.CallExpression)
	if receiver, member := targetProperty(argumentCopy.Expression()); receiver != "Box" || member != "$copy" {
		t.Fatalf("argument boundary = %s.%s, want exactly one Box.$copy", receiver, member)
	}
}

func TestNamedStructMultipleResultsCopyBorrowedValuesOnce(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	duplicate := targetFunction(t, source, "Duplicate")
	array := duplicate.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression().(tsgo.ArrayLiteralExpression)
	if len(array.Elements()) != 2 {
		t.Fatalf("Duplicate results = %d, want two", len(array.Elements()))
	}
	for index, element := range array.Elements() {
		call, ok := element.(tsgo.CallExpression)
		if !ok {
			t.Fatalf("Duplicate result %d = %T, want copy call", index, element)
		}
		if receiver, member := targetProperty(call.Expression()); receiver != "Box" || member != "$copy" {
			t.Fatalf("Duplicate result %d = %s.%s, want Box.$copy", index, receiver, member)
		}
	}

	consumer := targetFunction(t, source, "MultipleResultIsolated")
	statements := consumer.Body().(tsgo.Block).Statements()
	for index := 1; index <= 2; index++ {
		initializer := statements[index].(tsgo.VariableStatement).
			DeclarationList().Declarations()[0].Initializer()
		if _, ok := initializer.(tsgo.ElementAccessExpression); !ok {
			t.Fatalf("tuple declaration %d = %T, want owned element transfer", index, initializer)
		}
	}
}

func TestNamedStructCallArgumentsPlacePrerequisitesInSourceOrder(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	function := targetFunction(t, source, "CompositeSecondArgument")
	statements := function.Body().(tsgo.Block).Statements()
	if len(statements) != 7 {
		t.Fatalf(
			"CompositeSecondArgument statements = %d, want two argument captures, four field captures, and return",
			len(statements),
		)
	}
	firstCapture := statements[0].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	firstCall, ok := firstCapture.Initializer().(tsgo.CallExpression)
	if !ok || targetName(firstCall.Expression()) != "DirectValue" {
		t.Fatal("first argument was not evaluated before second-argument prerequisites")
	}
	secondCapture := statements[5].(tsgo.VariableStatement).
		DeclarationList().Declarations()[0]
	if _, ok := secondCapture.Initializer().(tsgo.NewExpression); !ok {
		t.Fatal("second argument was not captured after its field prerequisites")
	}
	call := statements[6].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
	if len(call.Arguments()) != 2 ||
		targetName(call.Arguments()[0]) != targetName(firstCapture.Name()) ||
		targetName(call.Arguments()[1]) != targetName(secondCapture.Name()) {
		t.Fatal("call does not consume argument captures in source order")
	}
}

func assertErasedBrand(t *testing.T, className string, member tsgo.ClassElement) {
	t.Helper()
	brand, ok := member.(tsgo.PropertyDeclaration)
	if !ok || targetName(brand.Name()) != "$goType" || brand.Initializer() != nil {
		t.Fatalf("%s brand = %T, want erased $goType property", className, member)
	}
	for _, kind := range []tsgo.SyntaxKind{
		tsgo.SyntaxKindDeclareKeyword,
		tsgo.SyntaxKindPrivateKeyword,
		tsgo.SyntaxKindReadonlyKeyword,
	} {
		if !hasModifier(brand.Modifiers(), kind) {
			t.Fatalf("%s brand lacks modifier kind %d", className, kind)
		}
	}
}

func hasModifier(modifiers []tsgo.ModifierLike, kind tsgo.SyntaxKind) bool {
	for _, modifier := range modifiers {
		if modifier.Kind() == kind {
			return true
		}
	}
	return false
}

func targetClass(t *testing.T, source tsgo.SourceFile, name string) tsgo.ClassDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		class, ok := statement.(tsgo.ClassDeclaration)
		if ok && class.Name().Text() == name {
			return class
		}
	}
	t.Fatalf("target class %s is absent", name)
	return nil
}

func targetFunction(t *testing.T, source tsgo.SourceFile, name string) tsgo.FunctionDeclaration {
	t.Helper()
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	t.Fatalf("target function %s is absent", name)
	return nil
}

func targetName(node tsgo.Node) string {
	identifier, _ := node.(tsgo.Identifier)
	if identifier == nil {
		return ""
	}
	return identifier.Text()
}

func targetProperty(node tsgo.Node) (string, string) {
	property, _ := node.(tsgo.PropertyAccessExpression)
	if property == nil {
		return "", ""
	}
	return targetName(property.Expression()), targetName(property.Name())
}
