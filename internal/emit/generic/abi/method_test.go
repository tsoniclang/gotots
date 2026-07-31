package abi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestClassMethodABIExactJoinCanonicalizesIdentities(t *testing.T) {
	method, _, operation := methodABIFixture(t)
	capability, err := Capability(operation, "capability")
	if err != nil {
		t.Fatal(err)
	}
	source, err := SourceParameters(method, []string{"source"})
	if err != nil {
		t.Fatal(err)
	}
	joined, err := JoinClassMethod(
		method,
		[]*api.GenericOperationContract{operation},
		Combine(source, []Binding[string]{capability}),
	)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"capability", "source"}
	if !slices.Equal(joined, expected) {
		t.Fatalf("method ABI = %v, want %v", joined, expected)
	}
	sourceFirstMutation := []string{"source", "capability"}
	if slices.Equal(joined, sourceFirstMutation) {
		t.Fatal("source-first method ABI mutation was not distinguished")
	}
}

func TestClassMethodABIExactJoinRejectsForeignSameShapeOwner(t *testing.T) {
	method, foreign, operation := methodABIFixture(t)
	capability, err := Capability(operation, "capability")
	if err != nil {
		t.Fatal(err)
	}
	foreignSource, err := SourceParameters(foreign, []string{"source"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = JoinClassMethod(
		method,
		[]*api.GenericOperationContract{operation},
		Combine(
			[]Binding[string]{capability},
			foreignSource,
		),
	)
	if err == nil {
		t.Fatal("foreign same-shape method owner joined the source ABI")
	}
}

func methodABIFixture(
	t *testing.T,
) (*types.Func, *types.Func, *api.GenericOperationContract) {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(
		fileSet,
		"method.go",
		`package fixture
type Box[T comparable] struct{ Value T }
func (box Box[T]) Same(other Box[T]) bool {
	return box.Value == other.Value
}
type Other[T comparable] struct{ Value T }
func (other Other[T]) Same(value Other[T]) bool {
	return other.Value == value.Value
}`,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	}
	pkg, err := (&types.Config{}).Check(
		"example.com/fixture",
		fileSet,
		[]*ast.File{file},
		info,
	)
	if err != nil {
		t.Fatal(err)
	}
	method := namedMethod(t, pkg, "Box")
	foreign := namedMethod(t, pkg, "Other")
	parameters := api.GenericDeclarationParameters(method)
	if len(parameters) != 1 {
		t.Fatalf("generic method parameters = %d, want 1", len(parameters))
	}
	operationSignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "left", parameters[0]),
			types.NewVar(token.NoPos, pkg, "right", parameters[0]),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, pkg, "result", types.Typ[types.Bool]),
		),
		false,
	)
	selection, err := api.SelectGenericOperation(api.GenericOperationEqual)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := api.NewGenericOperationContract(
		method,
		"equal",
		"$equal",
		api.GenericFunctionOperationConsumer(),
		selection,
		operationSignature,
	)
	if err != nil {
		t.Fatal(err)
	}
	return method, foreign, operation
}

func namedMethod(
	t *testing.T,
	pkg *types.Package,
	name string,
) *types.Func {
	t.Helper()
	typeName, ok := pkg.Scope().Lookup(name).(*types.TypeName)
	if !ok {
		t.Fatalf("%s is not a type name", name)
	}
	named, ok := typeName.Type().(*types.Named)
	if !ok || named.NumMethods() != 1 {
		t.Fatalf("%s method set is invalid", name)
	}
	return named.Method(0).Origin()
}
