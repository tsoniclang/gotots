package arrayvalue_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestMultiResultPrerequisiteCapturesEachResultInOrder(t *testing.T) {
	emission := compileArrayFixture(t)
	var function tsgo.FunctionDeclaration
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			candidate, ok := statement.(tsgo.FunctionDeclaration)
			if ok && candidate.Name().Text() == "MultiReturnArray" {
				function = candidate
			}
		}
	}
	if function == nil {
		t.Fatal("MultiReturnArray target function is absent")
	}
	statements := function.Body().(tsgo.Block).Statements()
	var captures []string
	for _, statement := range statements {
		variable, ok := statement.(tsgo.VariableStatement)
		if !ok {
			continue
		}
		for _, declaration := range variable.DeclarationList().Declarations() {
			name, ok := declaration.Name().(tsgo.Identifier)
			if ok && strings.HasPrefix(name.Text(), "results") {
				captures = append(captures, name.Text())
			}
		}
	}
	if len(captures) != 2 {
		t.Fatalf("multi-return captures = %v, want one per result", captures)
	}
	returned, ok := statements[len(statements)-1].(tsgo.ReturnStatement)
	if !ok {
		t.Fatalf("last MultiReturnArray statement = %T, want return", statements[len(statements)-1])
	}
	tuple, ok := returned.Expression().(tsgo.ArrayLiteralExpression)
	if !ok || len(tuple.Elements()) != 2 {
		t.Fatalf("multi-return target = %T, want two-element tuple", returned.Expression())
	}
	for index, element := range tuple.Elements() {
		identifier, ok := element.(tsgo.Identifier)
		if !ok || identifier.Text() != captures[index] {
			t.Fatalf("tuple element %d = %#v, want %s", index, element, captures[index])
		}
	}
}

func TestArraySourceAndRuntimeUseTypedTsGoAST(t *testing.T) {
	emission := compileArrayFixture(t)
	var runtimeClass tsgo.ClassDeclaration
	foundLengthTwo := false
	foundLengthThree := false
	for _, file := range emission.Files() {
		if file.OutputPath() == "runtime/array.ts" {
			statements := file.SourceFile().Statements()
			for _, statement := range statements {
				class, ok := statement.(tsgo.ClassDeclaration)
				if ok && class.Name().Text() == "GoArray" {
					runtimeClass = class
				}
			}
			continue
		}
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Type() == nil {
				continue
			}
			reference, ok := function.Type().(tsgo.TypeReferenceNode)
			if !ok || len(reference.TypeArguments()) != 2 {
				continue
			}
			literal, ok := reference.TypeArguments()[1].(tsgo.LiteralTypeNode)
			if !ok {
				continue
			}
			number, ok := literal.Literal().(tsgo.NumericLiteral)
			if !ok {
				continue
			}
			switch number.Text() {
			case "2":
				foundLengthTwo = true
			case "3":
				foundLengthThree = true
			}
		}
	}
	if runtimeClass == nil {
		t.Fatal("array runtime class was not emitted")
	}
	if runtimeClass.Name().Text() != "GoArray" ||
		len(runtimeClass.TypeParameters()) != 2 {
		t.Fatalf(
			"runtime class = %s with %d parameters",
			runtimeClass.Name().Text(),
			len(runtimeClass.TypeParameters()),
		)
	}
	if !foundLengthTwo || !foundLengthThree {
		t.Fatalf(
			"source array lengths = two %v, three %v",
			foundLengthTwo,
			foundLengthThree,
		)
	}
	assertArrayRuntimeMembers(t, runtimeClass)
}

func TestArrayRuntimeRequestExactJoinsOneDefinition(t *testing.T) {
	emission := compileArrayFixture(t)
	count := 0
	for _, file := range emission.Files() {
		if file.OutputPath() != "runtime/array.ts" {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			class, ok := statement.(tsgo.ClassDeclaration)
			if ok && class.Name().Text() == "GoArray" {
				count++
			}
		}
	}
	if count != 1 {
		t.Fatalf("RuntimeArray definitions = %d, want one", count)
	}
	contract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		t.Fatal(err)
	}
	if contract.OutputPath() != "runtime/array.ts" ||
		contract.ExportedName() != "GoArray" {
		t.Fatalf(
			"RuntimeArray contract = %q/%q",
			contract.OutputPath(),
			contract.ExportedName(),
		)
	}
}

func assertArrayRuntimeMembers(
	t *testing.T,
	class tsgo.ClassDeclaration,
) {
	t.Helper()
	found := make(map[string]tsgo.MethodDeclaration)
	for _, member := range class.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok {
			continue
		}
		name, ok := method.Name().(tsgo.Identifier)
		if ok {
			found[name.Text()] = method
		}
	}
	for _, name := range []string{
		"zero",
		"literal",
		"copy",
		"get",
		"set",
		"$check",
	} {
		if found[name] == nil {
			t.Fatalf("array runtime has no %s method", name)
		}
	}
	if len(found) != 6 {
		t.Fatalf("array runtime methods = %v, want the six closed members", found)
	}
	assertFreshZeroConstruction(t, found["zero"])
	assertCheckedAccess(t, found["get"])
	assertCheckedAccess(t, found["set"])
	assertBoundsFailure(t, found["$check"])
}

func assertFreshZeroConstruction(
	t *testing.T,
	method tsgo.MethodDeclaration,
) {
	t.Helper()
	statements := method.Body().(tsgo.Block).Statements()
	if len(statements) < 2 {
		t.Fatal("RuntimeArray.zero has no local storage construction")
	}
	variable, ok := statements[0].(tsgo.VariableStatement)
	if !ok {
		t.Fatalf("RuntimeArray.zero first statement = %T", statements[0])
	}
	declarations := variable.DeclarationList().Declarations()
	if len(declarations) != 1 {
		t.Fatalf("RuntimeArray.zero locals = %d", len(declarations))
	}
	if _, ok := declarations[0].Initializer().(tsgo.ArrayLiteralExpression); !ok {
		t.Fatalf(
			"RuntimeArray.zero storage = %T, want fresh array literal",
			declarations[0].Initializer(),
		)
	}
	result, ok := statements[len(statements)-1].(tsgo.ReturnStatement)
	if !ok {
		t.Fatalf("RuntimeArray.zero result = %T", statements[len(statements)-1])
	}
	if _, ok := result.Expression().(tsgo.NewExpression); !ok {
		t.Fatalf("RuntimeArray.zero return = %T, want fresh construction", result.Expression())
	}
}

func assertCheckedAccess(
	t *testing.T,
	method tsgo.MethodDeclaration,
) {
	t.Helper()
	statements := method.Body().(tsgo.Block).Statements()
	if len(statements) == 0 {
		t.Fatalf("RuntimeArray.%s has no body", methodName(method))
	}
	var operation tsgo.Expression
	wantOperation := "goRegionRead"
	switch statement := statements[0].(type) {
	case tsgo.ReturnStatement:
		operation = statement.Expression()
	case tsgo.ExpressionStatement:
		operation = statement.Expression()
		wantOperation = "goRegionWrite"
	default:
		t.Fatalf("RuntimeArray.%s first statement = %T", methodName(method), statement)
	}
	call, ok := operation.(tsgo.CallExpression)
	if !ok || len(call.Arguments()) < 2 {
		t.Fatalf("RuntimeArray.%s does not use a region operation", methodName(method))
	}
	operationName, ok := call.Expression().(tsgo.Identifier)
	if !ok || operationName.Text() != wantOperation {
		t.Fatalf("RuntimeArray.%s region operation = %T", methodName(method), call.Expression())
	}
	checked, ok := call.Arguments()[1].(tsgo.CallExpression)
	if !ok || len(checked.Arguments()) != 1 {
		t.Fatalf("RuntimeArray.%s region index is unchecked", methodName(method))
	}
	property, ok := checked.Expression().(tsgo.PropertyAccessExpression)
	if !ok {
		t.Fatalf("RuntimeArray.%s check is not a member call", methodName(method))
	}
	name, nameOK := property.Name().(tsgo.Identifier)
	if !nameOK || name.Text() != "$check" || property.Expression().Kind() != tsgo.SyntaxKindThisKeyword {
		t.Fatalf("RuntimeArray.%s does not call $check", methodName(method))
	}
}

func assertBoundsFailure(
	t *testing.T,
	method tsgo.MethodDeclaration,
) {
	t.Helper()
	statements := method.Body().(tsgo.Block).Statements()
	if len(statements) < 2 {
		t.Fatal("RuntimeArray.$check has no bounds branch")
	}
	branch, ok := statements[0].(tsgo.IfStatement)
	if !ok {
		t.Fatalf("RuntimeArray.$check bounds statement = %T", statements[0])
	}
	consequent := []tsgo.Statement{branch.ThenStatement()}
	if body, block := branch.ThenStatement().(tsgo.Block); block {
		consequent = body.Statements()
	}
	if len(consequent) != 1 {
		t.Fatal("RuntimeArray.$check bounds branch is not a single failure")
	}
	call, ok := consequent[0].(tsgo.ExpressionStatement).
		Expression().(tsgo.CallExpression)
	if !ok ||
		call.Expression().(tsgo.PropertyAccessExpression).
			Name().(tsgo.Identifier).Text() != "raiseRuntime" {
		t.Fatalf(
			"RuntimeArray.$check failure = %T, want shared panic call",
			consequent[0],
		)
	}
}

func methodName(method tsgo.MethodDeclaration) string {
	name, _ := method.Name().(tsgo.Identifier)
	return name.Text()
}
