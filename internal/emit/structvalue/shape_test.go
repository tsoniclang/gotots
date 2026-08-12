package structvalue_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNamedStructValuesConstructExactTargetShape(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	operations := map[string][]string{
		"Point": {"$zero", "$copy", "$equal"},
		"Box":   {"$zero", "$copy", "$equal"},
		"Empty": {"$zero", "$equal"},
	}
	fieldCounts := map[string]int{
		"Point": 2, "Box": 2, "Mirror": 2,
		"Reserved": 1, "Grouped": 2, "Empty": 0,
	}
	for _, name := range []string{
		"Point",
		"Box",
		"Mirror",
		"Reserved",
		"Grouped",
		"Empty",
	} {
		class := targetClass(t, source, name)
		if len(class.TypeParameters()) != 0 || len(class.HeritageClauses()) != 0 {
			t.Fatalf("%s has type parameters or heritage clauses", name)
		}
		members := class.Members()
		wantMembers := 2 + fieldCounts[name] + len(operations[name])
		if name == "Box" {
			wantMembers++
		}
		if len(members) != wantMembers {
			t.Fatalf("%s members = %d, want %d", name, len(members), wantMembers)
		}
		assertErasedBrand(t, name, members[0])
		constructor := classConstructor(t, class)
		if len(constructor.Parameters()) != 1 && fieldCounts[name] != 0 {
			t.Fatalf("%s constructor parameters = %d, want one", name, len(constructor.Parameters()))
		}
	}

	reserved := targetClass(t, source, "Reserved")
	constructor := classConstructor(t, reserved)
	input := constructor.Parameters()[0].Type().(tsgo.TypeLiteralNode)
	if got := typeLiteralMemberNames(input)[0]; got != "__go_constructor" {
		t.Fatalf("reserved constructor field = %q, want collision-safe target name", got)
	}

}

func TestNamedStructOperationsAreUniqueAndOwnedByClass(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	assertStaticOperationSequence(
		t,
		source,
		"Point",
		[]string{"$zero", "$copy", "$equal"},
	)
	assertStaticOperationSequence(
		t,
		source,
		"Box",
		[]string{"$zero", "$copy", "$equal"},
	)
	assertStaticOperationSequence(
		t,
		source,
		"Empty",
		[]string{"$zero", "$equal"},
	)
	for _, name := range []string{"Mirror", "Reserved", "Grouped"} {
		assertStaticOperationSequence(t, source, name, nil)
	}
}

func TestNamedStructValuesDefaultToDirectConstruction(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	for _, name := range []string{
		"NewBox",
		"CompositeArgument",
		"CompositeCalls",
		"CompositeSecondArgument",
		"CompositeField",
	} {
		statements := targetFunction(t, source, name).Body().(tsgo.Block).Statements()
		if len(statements) != 1 {
			t.Fatalf("%s statements = %d, want one direct return", name, len(statements))
		}
		if _, ok := statements[0].(tsgo.ReturnStatement); !ok {
			t.Fatalf("%s statement = %T, want direct return", name, statements[0])
		}
	}
	direct := targetFunction(t, source, "CompositeCalls").
		Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.PropertyAccessExpression).Expression().(tsgo.NewExpression)
	if targetName(direct.Expression()) != "Point" || len(direct.Arguments()) != 1 {
		t.Fatal("direct constructor is not one named-object new Point")
	}
	properties := objectAssignments(direct.Arguments()[0].(tsgo.ObjectLiteralExpression))
	if targetName(properties[0].Name()) != "Visible" ||
		targetName(properties[0].Initializer().(tsgo.CallExpression).Expression()) != "DirectVisible" {
		t.Fatal("direct constructor did not preserve source-order Visible")
	}
	if targetName(properties[1].Name()) != "X" ||
		targetName(properties[1].Initializer().(tsgo.CallExpression).Expression()) != "DirectX" {
		t.Fatal("direct constructor did not preserve source-order X")
	}
}

func TestNamedStructValuesPreserveConstructionOrderWhenSelected(t *testing.T) {
	source := structTargetSource(t, compileStructFixtureWithOptions(
		t,
		emit.Options{
			IntegerRepresentation: emit.IntegerRepresentationNumber,
			EvaluationOrder:       emit.EvaluationOrderPreserveGo,
		},
	))
	newBox := targetFunction(t, source, "NewBox")
	statements := newBox.Body().(tsgo.Block).Statements()
	if len(statements) != 1 {
		t.Fatalf("NewBox statements = %d, want one source-ordered return", len(statements))
	}
	result := statements[0].(tsgo.ReturnStatement).Expression().(tsgo.NewExpression)
	properties := objectAssignments(result.Arguments()[0].(tsgo.ObjectLiteralExpression))
	if targetName(properties[0].Name()) != "Active" || targetName(properties[1].Name()) != "Point" {
		t.Fatal("Box construction does not preserve Go keyed-field order")
	}
	point := properties[1].Initializer().(tsgo.NewExpression)
	pointProperties := objectAssignments(point.Arguments()[0].(tsgo.ObjectLiteralExpression))
	if targetName(pointProperties[0].Name()) != "Visible" ||
		targetName(pointProperties[1].Name()) != "X" {
		t.Fatal("nested Point construction does not preserve Go keyed-field order")
	}

	callStatements := targetFunction(t, source, "CompositeCalls").
		Body().(tsgo.Block).Statements()
	if len(callStatements) != 1 {
		t.Fatalf("CompositeCalls statements = %d, want one ordered return", len(callStatements))
	}
	callValue := callStatements[0].(tsgo.ReturnStatement).Expression().(tsgo.PropertyAccessExpression).Expression().(tsgo.NewExpression)
	callProperties := objectAssignments(callValue.Arguments()[0].(tsgo.ObjectLiteralExpression))
	firstCall := callProperties[0].Initializer().(tsgo.CallExpression)
	secondCall := callProperties[1].Initializer().(tsgo.CallExpression)
	if targetName(firstCall.Expression()) != "DirectVisible" ||
		targetName(secondCall.Expression()) != "DirectX" {
		t.Fatal("preserve-go did not capture call-valued keyed fields in source order")
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

func classConstructor(t *testing.T, class tsgo.ClassDeclaration) tsgo.ConstructorDeclaration {
	t.Helper()
	for _, member := range class.Members() {
		if constructor, ok := member.(tsgo.ConstructorDeclaration); ok {
			return constructor
		}
	}
	t.Fatalf("%s constructor is absent", class.Name().Text())
	return nil
}

func objectAssignments(source tsgo.ObjectLiteralExpression) []tsgo.PropertyAssignment {
	result := make([]tsgo.PropertyAssignment, 0, len(source.Properties()))
	for _, member := range source.Properties() {
		if property, ok := member.(tsgo.PropertyAssignment); ok {
			result = append(result, property)
		}
	}
	return result
}

func TestNamedStructValuesUseStaticallySelectedClassOperations(t *testing.T) {
	source := structTargetSource(t, compileStructFixture(t))
	invoke := targetFunction(t, source, "Invoke")
	call := invoke.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
	if receiver, member := targetProperty(call.Expression()); receiver != "value" ||
		member != "WithX" {
		t.Fatalf("receiver call target = %s.%s, want value.WithX", receiver, member)
	}
	if len(call.Arguments()) != 1 || targetName(call.Arguments()[0]) != "next" {
		t.Fatal("value-receiver copy leaked into the call site")
	}

	class := targetClass(t, source, "Box")
	method := targetMethod(t, class, "WithX")
	if hasModifier(method.Modifiers(), tsgo.SyntaxKindStaticKeyword) {
		t.Fatal("value receiver method was emitted as a static class member")
	}
	if len(method.Parameters()) != 1 || targetName(method.Parameters()[0].Name()) != "value" {
		t.Fatal("value receiver remained an explicit target parameter")
	}
	methodStatements := method.Body().(tsgo.Block).Statements()
	if len(methodStatements) != 3 {
		t.Fatalf("receiver body statements = %d, want copy, source store, and return", len(methodStatements))
	}
	copyDeclaration, ok := methodStatements[0].(tsgo.VariableStatement)
	if !ok {
		t.Fatalf("receiver copy statement = %T, want variable declaration", methodStatements[0])
	}
	copy := copyDeclaration.DeclarationList().Declarations()[0]
	copyCall, ok := copy.Initializer().(tsgo.CallExpression)
	if !ok || targetName(copy.Name()) != "box" {
		t.Fatal("value receiver does not begin with its local Go copy")
	}
	if receiver, member := targetProperty(copyCall.Expression()); receiver != "Box" ||
		member != "$copy" || len(copyCall.Arguments()) != 1 ||
		copyCall.Arguments()[0].Kind() != tsgo.SyntaxKindThisKeyword {
		t.Fatal("value receiver copy is not Box.$copy(this)")
	}
	if _, ok := methodStatements[1].(tsgo.ExpressionStatement); !ok {
		t.Fatal("receiver source store is absent after the copy")
	}
	if _, ok := methodStatements[2].(tsgo.ReturnStatement); !ok {
		t.Fatal("receiver source return is absent after the copy")
	}
	if targetFunctionOrNil(source, "Box_WithX") != nil {
		t.Fatal("value receiver method was duplicated as a top-level function")
	}

	assign := targetFunction(t, source, "AssignIsolated")
	assignExpression := assign.Body().(tsgo.Block).Statements()[1].(tsgo.ExpressionStatement).Expression().(tsgo.BinaryExpression)
	if assignExpression.OperatorToken().Kind() != tsgo.SyntaxKindEqualsToken ||
		targetName(assignExpression.Left()) != "target" {
		t.Fatal("assignment boundary is not a direct rebinding")
	}
	assignCopy := assignExpression.Right().(tsgo.CallExpression)
	if receiver, member := targetProperty(assignCopy.Expression()); receiver != "Box" ||
		member != "$copy" {
		t.Fatalf("assignment copy = %s.%s, want Box.$copy", receiver, member)
	}

	parameter := targetFunction(t, source, "ParameterIsolated")
	initializer := parameter.Body().(tsgo.Block).Statements()[0].(tsgo.VariableStatement).DeclarationList().Declarations()[0].
		Initializer().(tsgo.CallExpression)
	if targetName(initializer.Expression()) != "MutateParameter" {
		t.Fatal("parameter isolation does not call the selected function")
	}
	argumentCopy := initializer.Arguments()[0].(tsgo.CallExpression)
	if receiver, member := targetProperty(argumentCopy.Expression()); receiver != "Box" ||
		member != "$copy" {
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
		receiver, member := targetProperty(call.Expression())
		if receiver != "Box" || member != "$copy" {
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
	source := structTargetSource(t, compileStructFixtureWithOptions(
		t,
		emit.Options{
			IntegerRepresentation: emit.IntegerRepresentationNumber,
			EvaluationOrder:       emit.EvaluationOrderPreserveGo,
		},
	))
	function := targetFunction(t, source, "CompositeSecondArgument")
	statements := function.Body().(tsgo.Block).Statements()
	if len(statements) != 1 {
		t.Fatalf(
			"CompositeSecondArgument statements = %d, want one ordered call",
			len(statements),
		)
	}
	call := statements[0].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
	if len(call.Arguments()) != 2 {
		t.Fatalf("CompositeSecondArgument arguments = %d, want two", len(call.Arguments()))
	}
	firstCall, ok := call.Arguments()[0].(tsgo.CallExpression)
	if !ok || targetName(firstCall.Expression()) != "DirectValue" {
		t.Fatal("first argument is not the source DirectValue call")
	}
	box, ok := call.Arguments()[1].(tsgo.NewExpression)
	if !ok || targetName(box.Expression()) != "Box" {
		t.Fatalf("second argument = %T, want direct new Box", call.Arguments()[1])
	}
	properties := objectAssignments(box.Arguments()[0].(tsgo.ObjectLiteralExpression))
	if targetName(properties[0].Name()) != "Active" ||
		targetName(properties[1].Name()) != "Point" {
		t.Fatal("second argument does not preserve source field order")
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

func targetMethod(
	t *testing.T,
	class tsgo.ClassDeclaration,
	name string,
) tsgo.MethodDeclaration {
	t.Helper()
	for _, member := range class.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if ok && targetName(method.Name()) == name {
			return method
		}
	}
	t.Fatalf("target method %s.%s is absent", class.Name().Text(), name)
	return nil
}

func targetFunction(t *testing.T, source tsgo.SourceFile, name string) tsgo.FunctionDeclaration {
	t.Helper()
	if function := targetFunctionOrNil(source, name); function != nil {
		return function
	}
	t.Fatalf("target function %s is absent", name)
	return nil
}

func targetFunctionOrNil(source tsgo.SourceFile, name string) tsgo.FunctionDeclaration {
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == name {
			return function
		}
	}
	return nil
}

func assertStaticOperationSequence(
	t *testing.T,
	source tsgo.SourceFile,
	owner string,
	want []string,
) {
	t.Helper()
	class := targetClass(t, source, owner)
	counts := make(map[string]int, len(want))
	var got []string
	for _, member := range class.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok {
			continue
		}
		if !hasModifier(method.Modifiers(), tsgo.SyntaxKindStaticKeyword) {
			continue
		}
		name := targetName(method.Name())
		got = append(got, name)
		counts[name]++
	}
	if len(got) != len(want) {
		t.Fatalf("%s static operation set = %v, want %v", owner, got, want)
	}
	for index, name := range want {
		if got[index] != name {
			t.Fatalf("%s static operation %d = %s, want %s", owner, index, got[index], name)
		}
		if counts[name] != 1 {
			t.Fatalf("%s.%s definition count = %d, want one", owner, name, counts[name])
		}
	}
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
