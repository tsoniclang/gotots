package function_test

import (
	"context"
	"errors"
	"go/ast"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestCallableUnsupportedNeighborsFailAtTypedOwners(t *testing.T) {
	testCases := []struct {
		name      string
		source    string
		role      api.Role
		category  api.Category
		construct string
	}{
		{
			name: "variadic declaration",
			source: `package boundary

func Variadic(values ...int32) int32 {
	return 0
}
`,
			role:      api.RoleFileDeclaration,
			category:  api.CategoryType,
			construct: "*ast.FuncType",
		},
		{
			name: "generic declaration",
			source: `package boundary

func Identity[T any](value T) T {
	return value
}
`,
			role:      api.RoleFileDeclaration,
			category:  api.CategoryDeclaration,
			construct: "*ast.FuncDecl",
		},
		{
			name: "variadic function value",
			source: `package boundary

func Accept(transform func(...int32) int32) int32 {
	return 0
}
`,
			role:      api.RoleParameterType,
			category:  api.CategoryType,
			construct: "*ast.FuncType",
		},
		{
			name: "nil function value",
			source: `package boundary

func NilValue() func() int32 {
	var callback func() int32
	return callback
}
`,
			role:      api.RoleLocalValue,
			category:  api.CategoryExpression,
			construct: "*ast.Ident",
		},
		{
			name: "function nil comparison",
			source: `package boundary

func IsNil(callback func() int32) bool {
	return callback == nil
}
`,
			role:      api.RoleReturnResult,
			category:  api.CategoryExpression,
			construct: "*ast.BinaryExpr",
		},
		{
			name: "new function value requires nil callable representation",
			source: `package boundary

func NewCallback() *func() int32 {
	return new(func() int32)
}
`,
			role:      api.RoleCallArgument,
			category:  api.CategoryExpression,
			construct: "*ast.FuncType",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := compileTemporaryFunctionSource(t, testCase.source)
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

func TestCallableSyntaxAndTypeMutationsFailAtSignatureOwner(t *testing.T) {
	t.Run("named parameter loses syntax binding", func(t *testing.T) {
		loaded := loadCallableValuesProject(t)
		apply := sourceFunction(t, loaded.Files()[0].Syntax(), "Apply")
		apply.Type.Params.List[0].Names = nil

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleParameterType,
			api.CategoryType,
			"*ast.Field",
		)
	})

	t.Run("result type loses checker fact", func(t *testing.T) {
		loaded := loadCallableValuesProject(t)
		offset := sourceFunction(t, loaded.Files()[0].Syntax(), "Offset")
		resultType := offset.Type.Results.List[0].Type
		delete(loaded.TypesInfo().Types, resultType)

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleResultType,
			api.CategoryType,
			"*ast.Field",
		)
	})

	t.Run("nested result points at outer result", func(t *testing.T) {
		loaded := loadNamedResultsProject(t)
		nested := sourceFunction(t, loaded.Files()[0].Syntax(), "Nested")
		outerName := nested.Type.Results.List[0].Names[0]
		declaration := nested.Body.List[0].(*ast.AssignStmt)
		literal := declaration.Rhs[0].(*ast.FuncLit)
		innerName := literal.Type.Results.List[0].Names[0]
		loaded.TypesInfo().Defs[innerName] = loaded.TypesInfo().Defs[outerName]

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleCallableResult,
			api.CategoryType,
			"*ast.Ident",
		)
	})

	t.Run("capture loses object identity", func(t *testing.T) {
		loaded := loadCallableValuesProject(t)
		offset := sourceFunction(t, loaded.Files()[0].Syntax(), "Offset")
		literal := offset.Body.List[0].(*ast.ReturnStmt).Results[0].(*ast.FuncLit)
		result := literal.Body.List[0].(*ast.ReturnStmt)
		capture := result.Results[0].(*ast.BinaryExpr).Y.(*ast.Ident)
		delete(loaded.TypesInfo().Uses, capture)

		err := compileLoadedPackage(t, loaded)
		assertUnsupported(
			t,
			err,
			api.RoleBinaryRight,
			api.CategoryExpression,
			"*ast.Ident",
		)
	})
}

func compileTemporaryFunctionSource(t *testing.T, source string) error {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/callableboundary\n\ngo 1.26.4\n",
	)
	writeFile(t, filepath.Join(directory, "source.go"), source)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return compileLoadedPackage(t, loaded)
}

func compileLoadedPackage(t *testing.T, loaded *load.Package) error {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), roots)
	return err
}

func assertUnsupported(
	t *testing.T,
	err error,
	role api.Role,
	category api.Category,
	construct string,
) {
	t.Helper()
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want *api.UnsupportedError", err)
	}
	if unsupported.Role != role ||
		unsupported.Category != category ||
		unsupported.Construct != construct {
		t.Fatalf("unsupported = %#v", unsupported)
	}
}
