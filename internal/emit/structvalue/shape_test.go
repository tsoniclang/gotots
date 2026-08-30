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
		wantMembers := 3 + len(operations[name])
		if name == "Box" {
			wantMembers++
		}
		if len(members) != wantMembers {
			t.Fatalf(
				"%s members = %d, want %d: %v",
				name,
				len(members),
				wantMembers,
				classMemberNames(members),
			)
		}
		assertErasedBrand(t, name, members[0])
		constructor := classConstructor(t, class)
		if len(constructor.Parameters()) != fieldCounts[name] {
			t.Fatalf("%s constructor parameters = %d, want %d", name, len(constructor.Parameters()), fieldCounts[name])
		}
	}

	reserved := targetClass(t, source, "Reserved")
	constructor := classConstructor(t, reserved)
	if got := targetName(constructor.Parameters()[0].Name()); got != "__go_constructor" {
		t.Fatalf("reserved constructor parameter = %q, want collision-safe target name", got)
	}

}

func classMemberNames(members []tsgo.ClassElement) []string {
	names := make([]string, 0, len(members))
	for _, member := range members {
		switch member := member.(type) {
		case tsgo.ConstructorDeclaration:
			names = append(names, "constructor")
		case tsgo.PropertyDeclaration:
			names = append(names, targetName(member.Name()))
		case tsgo.MethodDeclaration:
			names = append(names, targetName(member.Name()))
		default:
			names = append(names, "unknown")
		}
	}
	return names
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
	if targetName(direct.Expression()) != "Point" || len(direct.Arguments()) != 2 {
		t.Fatal("direct constructor is not one source-named positional new Point")
	}
	if targetName(direct.Arguments()[0].(tsgo.CallExpression).Expression()) != "DirectX" {
		t.Fatal("direct constructor did not select declaration-order X")
	}
	if targetName(direct.Arguments()[1].(tsgo.CallExpression).Expression()) != "DirectVisible" {
		t.Fatal("direct constructor did not select declaration-order Visible")
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
	if len(statements) != 5 {
		t.Fatalf("NewBox statements = %d, want four captures and one return", len(statements))
	}
	point := capturedValue(t, statements[3]).(tsgo.NewExpression)
	if targetName(point.Expression()) != "Point" || len(point.Arguments()) != 2 {
		t.Fatal("nested Point capture is not direct positional construction")
	}
	result := statements[4].(tsgo.ReturnStatement).Expression().(tsgo.NewExpression)
	if targetName(result.Expression()) != "Box" || len(result.Arguments()) != 2 {
		t.Fatal("Box construction is not direct declaration-order construction")
	}

	callStatements := targetFunction(t, source, "CompositeCalls").
		Body().(tsgo.Block).Statements()
	if len(callStatements) != 3 {
		t.Fatalf("CompositeCalls statements = %d, want two captures and one return", len(callStatements))
	}
	firstCall := capturedValue(t, callStatements[0]).(tsgo.CallExpression)
	secondCall := capturedValue(t, callStatements[1]).(tsgo.CallExpression)
	if targetName(firstCall.Expression()) != "DirectVisible" ||
		targetName(secondCall.Expression()) != "DirectX" {
		t.Fatal("preserve-go did not capture call-valued keyed fields in source order")
	}
	callValue := callStatements[2].(tsgo.ReturnStatement).Expression().(tsgo.PropertyAccessExpression).Expression().(tsgo.NewExpression)
	if len(callValue.Arguments()) != 2 {
		t.Fatal("preserve-go Point construction is not direct positional output")
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

func capturedValue(t *testing.T, statement tsgo.Statement) tsgo.Expression {
	t.Helper()
	declaration, ok := statement.(tsgo.VariableStatement)
	if !ok || len(declaration.DeclarationList().Declarations()) != 1 {
		t.Fatalf("capture statement = %T, want one variable declaration", statement)
	}
	return declaration.DeclarationList().Declarations()[0].Initializer()
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
	if len(statements) != 7 {
		t.Fatalf(
			"CompositeSecondArgument statements = %d, want six captures and one call",
			len(statements),
		)
	}
	firstCall, ok := capturedValue(t, statements[0]).(tsgo.CallExpression)
	if !ok || targetName(firstCall.Expression()) != "DirectValue" {
		t.Fatal("first argument was not evaluated before second-argument prerequisites")
	}
	box, ok := capturedValue(t, statements[5]).(tsgo.NewExpression)
	if !ok || targetName(box.Expression()) != "Box" || len(box.Arguments()) != 2 {
		t.Fatalf("captured second argument = %T, want direct new Box", capturedValue(t, statements[5]))
	}
	call := statements[6].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
	if len(call.Arguments()) != 2 {
		t.Fatalf("CompositeSecondArgument arguments = %d, want two", len(call.Arguments()))
	}
	for index, argument := range call.Arguments() {
		if _, ok := argument.(tsgo.Identifier); !ok {
			t.Fatalf("final call argument %d = %T, want captured identifier", index, argument)
		}
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
