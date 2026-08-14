package structvalue_test

import (
	"context"
	"errors"
	"go/ast"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNamedStructCompositeMutationFailsAtElementOwner(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: structValuesDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0].Files()[0].Syntax()
	newBox := sourceFunction(t, source, "NewBox")
	composite := newBox.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CompositeLit)
	composite.Elts[1] = composite.Elts[1].(*ast.KeyValueExpr).Value

	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(program, roots)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Role != api.RoleCompositeElement ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.CompositeLit" {
		t.Fatalf("error = %#v, want mixed-composite typed failure", err)
	}
}

func TestNamedStructReceiverSelectionUsesGoTypesIdentity(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: structValuesDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0].Files()[0].Syntax()
	invoke := sourceFunction(t, source, "Invoke")
	call := invoke.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.CallExpr)
	selector := call.Fun.(*ast.SelectorExpr)
	selector.Sel.Name = "forgedSpelling"
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	target := targetFunction(t, structTargetSource(t, emission), "Invoke")
	targetCall := target.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
	if receiver, member := targetProperty(targetCall.Expression()); receiver != "value" ||
		member != "WithX" {
		t.Fatal("receiver call used mutated source spelling instead of selection identity")
	}
	delete(program.Roots()[0].TypesInfo().Selections, selector)
	_, err = emit.Compile(program, roots)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Role != api.RoleCallCallee ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.SelectorExpr" {
		t.Fatalf("missing selection error = %#v, want typed receiver-call failure", err)
	}
}

func TestExportedValueReceiverIsAReachabilityRoot(t *testing.T) {
	emission, err := compileTemporaryStructProgram(t, `package boundary

type Value struct {
	X int32
}

func (value Value) Unused() Value {
	return value
}
`)
	if err != nil {
		t.Fatal(err)
	}
	source := structTargetSource(t, emission)
	class := targetClass(t, source, "Value")
	targetMethod(t, class, "Unused")
	if targetFunctionOrNil(source, "Value_Unused") != nil {
		t.Fatal("receiver method was duplicated as a top-level function")
	}
}

func TestStructFieldCannotOccupyPromiseAssimilationMember(t *testing.T) {
	emission, err := compileTemporaryStructProgram(t, `package boundary

type Value struct {
	then func() int
}

func NewValue(then func() int) Value {
	return Value{then: then}
}

func (value *Value) Invoke() int {
	return value.then()
}
`)
	if err != nil {
		t.Fatal(err)
	}
	class := targetClass(t, structTargetSource(t, emission), "Value")
	constructor := classConstructor(t, class)
	if len(constructor.Parameters()) != 1 {
		t.Fatalf("Value constructor parameters = %d, want one", len(constructor.Parameters()))
	}
	fieldName := targetName(constructor.Parameters()[0].Name())
	if fieldName == "then" || fieldName == "" {
		t.Fatalf("callable Go field target name = %q", fieldName)
	}
	method := targetMethod(t, class, "Invoke")
	statements := method.Body().(tsgo.Block).Statements()
	if len(statements) != 2 {
		t.Fatalf("Value.Invoke statements = %d, want capture and return", len(statements))
	}
	capture, ok := statements[0].(tsgo.VariableStatement)
	if !ok || len(capture.DeclarationList().Declarations()) != 1 {
		t.Fatalf("Value.Invoke capture = %T", statements[0])
	}
	selected := selectedMemberName(
		capture.DeclarationList().Declarations()[0].Initializer(),
	)
	if selected != fieldName {
		t.Fatalf(
			"Value.Invoke capture selects %q, want %q",
			selected,
			fieldName,
		)
	}
}

func selectedMemberName(node tsgo.Node) string {
	switch node := node.(type) {
	case tsgo.PropertyAccessExpression:
		return targetName(node.Name())
	case tsgo.ParenthesizedExpression:
		return selectedMemberName(node.Expression())
	case tsgo.BinaryExpression:
		if name := selectedMemberName(node.Left()); name != "" {
			return name
		}
		return selectedMemberName(node.Right())
	case tsgo.CallExpression:
		return selectedMemberName(node.Expression())
	default:
		return ""
	}
}

func TestSingleStructReturnCopiesBorrowedField(t *testing.T) {
	emission, err := compileTemporaryStructProgram(t, `package boundary

type Point struct {
	X int32
}

type Box struct {
	Point Point
}

func Snapshot(value *Box) Point {
	return value.Point
}
`)
	if err != nil {
		t.Fatal(err)
	}
	function := targetFunction(t, structTargetSource(t, emission), "Snapshot")
	result := function.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression()
	call, ok := result.(tsgo.CallExpression)
	if !ok {
		t.Fatalf("Snapshot result = %T, want Point.$copy call", result)
	}
	receiver, member := targetProperty(call.Expression())
	if receiver != "Point" || member != "$copy" || len(call.Arguments()) != 1 {
		t.Fatalf(
			"Snapshot result = %s.%s with %d arguments, want Point.$copy with one argument",
			receiver,
			member,
			len(call.Arguments()),
		)
	}
}

func compileTemporaryStructSource(t *testing.T, source string) error {
	t.Helper()
	_, err := compileTemporaryStructProgram(t, source)
	return err
}

func sourceFunction(
	t *testing.T,
	source *ast.File,
	name string,
) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range source.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	t.Fatalf("fixture function %s is absent", name)
	return nil
}

func compileTemporaryStructProgram(
	t *testing.T,
	source string,
) (emit.ProgramEmission, error) {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/boundary\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(directory, "source.go"), source)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	return emit.Compile(program, roots)
}
