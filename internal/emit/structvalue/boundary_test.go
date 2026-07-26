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

func TestNamedStructUnsupportedNeighborsFailAtTypedOwners(t *testing.T) {
	testCases := []struct {
		name      string
		source    string
		role      api.Role
		category  api.Category
		construct string
	}{
		{
			name: "tag",
			source: "package boundary\n" +
				"type Tagged struct { Value int32 `json:\"value\"` }\n",
			role:      api.RoleStructField,
			category:  api.CategoryDeclaration,
			construct: "*ast.Field",
		},
		{
			name: "embedding",
			source: "package boundary\n" +
				"type Base struct { Value int32 }\n" +
				"type Embedded struct { Base }\n",
			role:      api.RoleStructField,
			category:  api.CategoryDeclaration,
			construct: "*ast.Field",
		},
		{
			name: "pointer field",
			source: "package boundary\n" +
				"type Pointer struct { Value *int32 }\n",
			role:      api.RoleStructFieldType,
			category:  api.CategoryType,
			construct: "*ast.StarExpr",
		},
		{
			name: "interface field",
			source: "package boundary\n" +
				"type Interface struct { Value interface{ Read() int32 } }\n",
			role:      api.RoleStructFieldType,
			category:  api.CategoryType,
			construct: "*ast.InterfaceType",
		},
		{
			name: "unsigned integer field",
			source: "package boundary\n" +
				"type Unsigned struct { Value uint64 }\n",
			role:      api.RoleStructFieldType,
			category:  api.CategoryType,
			construct: "*ast.Ident",
		},
		{
			name: "generic struct",
			source: "package boundary\n" +
				"type Generic[T any] struct { Value T }\n",
			role:      api.RoleFileDeclaration,
			category:  api.CategoryDeclaration,
			construct: "*ast.GenDecl",
		},
		{
			name: "pointer receiver",
			source: "package boundary\n" +
				"type Value struct { X int32 }\n" +
				"func (value *Value) Set(next int32) { value.X = next }\n",
			role:      api.RoleReceiverType,
			category:  api.CategoryType,
			construct: "*ast.StarExpr",
		},
		{
			name: "method expression",
			source: "package boundary\n" +
				"type Value struct { X int32 }\n" +
				"func (value Value) WithX(next int32) Value { value.X = next; return value }\n" +
				"func Use(value Value) Value { return Value.WithX(value, 1) }\n",
			role:      api.RoleReturnResult,
			category:  api.CategoryExpression,
			construct: "*ast.CallExpr",
		},
		{
			name: "method value",
			source: "package boundary\n" +
				"type Value struct { X int32 }\n" +
				"func (value Value) WithX(next int32) Value { value.X = next; return value }\n" +
				"func Use(value Value) Value { method := value.WithX; return method(1) }\n",
			role:      api.RoleLocalValue,
			category:  api.CategoryExpression,
			construct: "*ast.SelectorExpr",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := compileTemporaryStructSource(t, testCase.source)
			var unsupported *api.UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("error = %v, want *api.UnsupportedError", err)
			}
			if unsupported.Role != testCase.role ||
				unsupported.Category != testCase.category ||
				unsupported.Construct != testCase.construct {
				t.Fatalf("unsupported = %#v", unsupported)
			}
		})
	}
}

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
	if targetName(targetCall.Expression()) != "Box_WithX" {
		t.Fatal("receiver call used mutated source spelling instead of selection identity")
	}
	delete(program.Roots()[0].TypesInfo().Selections, selector)
	_, err = emit.Compile(program, roots)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Role != api.RoleReturnResult ||
		unsupported.Category != api.CategoryExpression ||
		unsupported.Construct != "*ast.CallExpr" {
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
	targetFunction(t, source, "Value_Unused")
	class := targetClass(t, source, "Value")
	for _, member := range class.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if ok && targetName(method.Name()) == "Unused" {
			t.Fatal("receiver method was attached to the representation class")
		}
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
