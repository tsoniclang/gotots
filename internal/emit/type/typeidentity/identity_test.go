package typeidentity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"testing"
)

func TestLocalComponentsIncludesInterfaceMethodContracts(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package local

func use() {
	type Local int32
	var _ interface {
		Take(Local)
		Return() Local
		}
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	checked, err := new(types.Config).Check(
		"example.com/local",
		fileSet,
		[]*ast.File{source},
		info,
	)
	if err != nil {
		t.Fatal(err)
	}
	var local *types.TypeName
	var contract *types.Interface
	for identifier, object := range info.Defs {
		if identifier.Name == "Local" {
			local, _ = object.(*types.TypeName)
		}
	}
	ast.Inspect(source, func(node ast.Node) bool {
		target, ok := node.(*ast.InterfaceType)
		if !ok {
			return true
		}
		represented, ok := info.Types[target].Type.(*types.Interface)
		if ok {
			contract = represented
		}
		return true
	})
	if local == nil || contract == nil || local.Pkg() != checked {
		t.Fatal("interface-local-component fixture is incomplete")
	}
	components := LocalComponents(contract)
	if len(components) != 1 || components[0] != local {
		t.Fatalf("local components = %#v, want Local", components)
	}
	firstKey, err := NamedObjectKey(local, source, "local/source.ts")
	if err != nil {
		t.Fatal(err)
	}
	secondKey, err := NamedObjectKey(local, source, "other/source.ts")
	if err != nil {
		t.Fatal(err)
	}
	if firstKey == secondKey {
		t.Fatal("lexically distinct local declarations share one identity")
	}
}

func TestParameterizedKeyUsesOwnerAssignedParameterIdentity(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package generic

func First[T, U any](left T, right U) T { return left }
func Renamed[A, B any](left A, right B) A { return left }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: make(map[*ast.Ident]types.Object)}
	checked, err := new(types.Config).Check(
		"example.com/generic",
		fileSet,
		[]*ast.File{source},
		info,
	)
	if err != nil {
		t.Fatal(err)
	}
	first := checked.Scope().Lookup("First").Type().(*types.Signature)
	renamed := checked.Scope().Lookup("Renamed").Type().(*types.Signature)
	firstKey, err := BuildParameterizedKey(
		receiverFreeSignature(first),
		packageNamedIdentity,
		parameterOrdinalIdentity(first),
	)
	if err != nil {
		t.Fatal(err)
	}
	renamedKey, err := BuildParameterizedKey(
		receiverFreeSignature(renamed),
		packageNamedIdentity,
		parameterOrdinalIdentity(renamed),
	)
	if err != nil {
		t.Fatal(err)
	}
	if firstKey != renamedKey {
		t.Fatalf(
			"renaming equivalent type parameters changed identity: %s != %s",
			firstKey,
			renamedKey,
		)
	}
	_, err = BuildKey(receiverFreeSignature(first), packageNamedIdentity)
	if err == nil || !strings.Contains(err.Error(), "no identity owner") {
		t.Fatalf("unowned type parameter error = %v", err)
	}
}

func packageNamedIdentity(object *types.TypeName) (string, error) {
	return object.Pkg().Path() + "|" + object.Name(), nil
}

func parameterOrdinalIdentity(
	signature *types.Signature,
) TypeParameterIdentity {
	identities := make(map[*types.TypeParam]string)
	for index := range signature.TypeParams().Len() {
		identities[signature.TypeParams().At(index)] =
			"function|" + strconv.Itoa(index)
	}
	return func(parameter *types.TypeParam) (string, error) {
		return identities[parameter], nil
	}
}
