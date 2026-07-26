package mapruntime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildCreatesOneTypedGenericMapClass(t *testing.T) {
	statement, err := Build(tsgo.NewFactory(), api.RuntimeMap)
	if err != nil {
		t.Fatal(err)
	}
	class, ok := statement.(tsgo.ClassDeclaration)
	if !ok {
		t.Fatalf("definition = %T, want ClassDeclaration", statement)
	}
	if class.Name().Text() != "GoMap" {
		t.Fatalf("class name = %q, want GoMap", class.Name().Text())
	}
	if len(class.TypeParameters()) != 2 {
		t.Fatalf("type parameters = %d, want key and value", len(class.TypeParameters()))
	}
	if len(class.Members()) != 9 {
		t.Fatalf("members = %d, want one constructor and eight operations", len(class.Members()))
	}
}

func TestRuntimeMapMutationGuardsOwnMissingAndNilWriteSemantics(t *testing.T) {
	statement, err := Build(tsgo.NewFactory(), api.RuntimeMap)
	if err != nil {
		t.Fatal(err)
	}
	class := statement.(tsgo.ClassDeclaration)
	constructor := class.Members()[0].(tsgo.ConstructorDeclaration)
	valuesType := constructor.Parameters()[1].Type()
	storage, ok := valuesType.(tsgo.UnionTypeNode)
	if !ok {
		t.Fatalf("storage type = %T, want Map<K,V> | undefined", valuesType)
	}
	storageTypes := storage.Types()
	if len(storageTypes) != 2 {
		t.Fatalf("storage union members = %d, want Map<K,V> and undefined", len(storageTypes))
	}
	nativeMap, ok := storageTypes[0].(tsgo.TypeReferenceNode)
	if !ok ||
		nativeMap.TypeName().(tsgo.Identifier).Text() != "Map" ||
		len(nativeMap.TypeArguments()) != 2 {
		t.Fatalf("storage owner = %T, want typed native Map<K,V>", storageTypes[0])
	}
	methods := make(map[string]tsgo.MethodDeclaration)
	for _, member := range class.Members()[1:] {
		method := member.(tsgo.MethodDeclaration)
		methods[method.Name().(tsgo.Identifier).Text()] = method
	}
	lookup := methods["lookup"].Body().(tsgo.Block).Statements()
	if len(lookup) != 2 {
		t.Fatalf("lookup statements = %d, want missing guard and present return", len(lookup))
	}
	missing, ok := lookup[0].(tsgo.IfStatement)
	if !ok || missing.Expression().Kind() != tsgo.SyntaxKindBinaryExpression {
		t.Fatalf("lookup missing guard = %T", lookup[0])
	}
	missingReturn := missing.ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.PropertyAccessExpression)
	if missingReturn.Name().(tsgo.Identifier).Text() != zeroName {
		t.Fatal("missing lookup does not return the represented Go zero")
	}
	present := lookup[1].(tsgo.ReturnStatement).Expression()
	if present.Kind() != tsgo.SyntaxKindNonNullExpression {
		t.Fatalf("present lookup = %T, want checked Map.get result", present)
	}

	store := methods["store"].Body().(tsgo.Block).Statements()
	if len(store) != 2 {
		t.Fatalf("store statements = %d, want nil guard and set", len(store))
	}
	nilGuard, ok := store[0].(tsgo.IfStatement)
	if !ok {
		t.Fatalf("store nil guard = %T", store[0])
	}
	if _, ok := nilGuard.ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.ThrowStatement); !ok {
		t.Fatal("nil map store does not throw")
	}
	setCall := store[1].(tsgo.ExpressionStatement).
		Expression().(tsgo.CallExpression).
		Expression().(tsgo.PropertyAccessExpression)
	if setCall.Name().(tsgo.Identifier).Text() != "set" {
		t.Fatal("non-nil store does not use the one encapsulated Map")
	}
}

func TestBuildRejectsSiblingRuntimeSymbols(t *testing.T) {
	for _, symbol := range []api.RuntimeSymbol{
		api.RuntimeInvalid,
		api.RuntimeStringIndex,
		api.RuntimePointer,
		api.RuntimeArray,
		api.RuntimeSlice,
	} {
		if _, err := Build(tsgo.NewFactory(), symbol); err == nil {
			t.Fatalf("symbol %d was accepted by map owner", symbol)
		}
	}
}
