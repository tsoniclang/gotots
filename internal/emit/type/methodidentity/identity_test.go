package methodidentity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestMethodIdentityUsesVisibilityPackageAndExactSignature(t *testing.T) {
	firstPackage := types.NewPackage("example.com/first", "first")
	secondPackage := types.NewPackage("example.com/second", "second")
	receiver := types.NewVar(token.NoPos, firstPackage, "receiver", types.Typ[types.Int32])
	result := types.NewVar(token.NoPos, firstPackage, "", types.Typ[types.Int32])
	signature := func(parameterType types.Type) *types.Signature {
		return types.NewSignatureType(
			receiver,
			nil,
			nil,
			types.NewTuple(types.NewVar(
				token.NoPos,
				firstPackage,
				"value",
				parameterType,
			)),
			types.NewTuple(result),
			false,
		)
	}
	exportedFirst := types.NewFunc(
		token.Pos(1),
		firstPackage,
		"Read",
		signature(types.Typ[types.Int32]),
	)
	exportedSecond := types.NewFunc(
		token.Pos(2),
		secondPackage,
		"Read",
		signature(types.Typ[types.Int32]),
	)
	privateFirst := types.NewFunc(
		token.Pos(3),
		firstPackage,
		"read",
		signature(types.Typ[types.Int32]),
	)
	privateSecond := types.NewFunc(
		token.Pos(4),
		secondPackage,
		"read",
		signature(types.Typ[types.Int32]),
	)
	different := types.NewFunc(
		token.Pos(5),
		firstPackage,
		"Read",
		signature(types.Typ[types.String]),
	)
	identity := func(object *types.TypeName) (string, error) {
		if object.Pkg() == nil {
			return object.Name(), nil
		}
		return object.Pkg().Path() + "." + object.Name(), nil
	}
	key := func(method *types.Func) string {
		t.Helper()
		result, err := BuildKey(method, identity)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	if key(exportedFirst) != key(exportedSecond) ||
		!Equivalent(exportedFirst, exportedSecond) {
		t.Fatal("exported equivalent methods do not share identity")
	}
	if key(privateFirst) == key(privateSecond) ||
		Equivalent(privateFirst, privateSecond) {
		t.Fatal("unexported methods from different packages share identity")
	}
	if key(exportedFirst) == key(different) ||
		Equivalent(exportedFirst, different) {
		t.Fatal("different method signatures share identity")
	}
}

func TestGenericReceiverMethodIdentityUsesParameterOrdinals(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package generic

type First[T, U any] struct{}

func (First[T, U]) private(left T, right U) T { return left }

type Renamed[A, B any] struct{}

func (Renamed[A, B]) private(left A, right B) A { return left }

type Swapped[A, B any] struct{}

func (Swapped[A, B]) private(left B, right A) A { return right }
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
	method := func(typeName string) *types.Func {
		t.Helper()
		named, ok := checked.Scope().Lookup(typeName).Type().(*types.Named)
		if !ok || named.NumMethods() != 1 {
			t.Fatalf("%s method set is incomplete", typeName)
		}
		return named.Method(0)
	}
	identity := func(object *types.TypeName) (string, error) {
		return object.Pkg().Path() + "." + object.Name(), nil
	}
	key := func(typeName string) string {
		t.Helper()
		result, buildErr := BuildKey(method(typeName), identity)
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		return result
	}
	if key("First") != key("Renamed") {
		t.Fatal("renaming receiver type parameters changed method identity")
	}
	if key("First") == key("Swapped") {
		t.Fatal("changing receiver type-parameter positions preserved method identity")
	}
}

func TestInstantiatedInterfaceMethodMatchesConcreteImplementation(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package generic

type Value[T any] interface {
	Get() T
}

type Box struct{}

func (Box) Get() int32 { return 0 }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := new(types.Config).Check(
		"example.com/generic",
		fileSet,
		[]*ast.File{source},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	value := checked.Scope().Lookup("Value").Type().(*types.Named)
	instantiated, err := types.Instantiate(
		types.NewContext(),
		value,
		[]types.Type{types.Typ[types.Int32]},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	interfaceMethod := instantiated.Underlying().(*types.Interface).Method(0)
	boxMethod := checked.Scope().Lookup("Box").Type().(*types.Named).Method(0)
	identity := func(object *types.TypeName) (string, error) {
		return object.Pkg().Path() + "." + object.Name(), nil
	}
	interfaceKey, err := BuildKey(interfaceMethod, identity)
	if err != nil {
		t.Fatal(err)
	}
	boxKey, err := BuildKey(boxMethod, identity)
	if err != nil {
		t.Fatal(err)
	}
	if interfaceKey != boxKey || !Equivalent(interfaceMethod, boxMethod) {
		t.Fatal("instantiated generic interface method does not match its concrete implementation")
	}
}

func TestGenericInterfaceMethodIdentityUsesParameterOrdinals(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package generic

type First[T any] interface {
	Read() T
}

type Renamed[U any] interface {
	Read() U
}

type Different[T any] interface {
	Write() T
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := new(types.Config).Check(
		"example.com/generic",
		fileSet,
		[]*ast.File{source},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	method := func(typeName string) *types.Func {
		t.Helper()
		named := checked.Scope().Lookup(typeName).Type().(*types.Named)
		return named.Underlying().(*types.Interface).Method(0)
	}
	identity := func(object *types.TypeName) (string, error) {
		return object.Pkg().Path() + "." + object.Name(), nil
	}
	key := func(typeName string) string {
		t.Helper()
		result, buildErr := BuildKey(method(typeName), identity)
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		return result
	}
	if key("First") != key("Renamed") ||
		!Equivalent(method("First"), method("Renamed")) {
		t.Fatal("alpha-equivalent generic interface methods do not share identity")
	}
	if key("First") == key("Different") ||
		Equivalent(method("First"), method("Different")) {
		t.Fatal("differently named generic interface methods share identity")
	}
}
