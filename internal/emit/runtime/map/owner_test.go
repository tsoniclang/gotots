package mapruntime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildCreatesOneTypedGenericMapClass(t *testing.T) {
	statement, err := Build(
		tsgo.NewFactory(),
		api.RuntimeMap,
		panicClassName(t),
	)
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
	if len(class.Members()) != 11 {
		t.Fatalf("members = %d, want constructor and ten map operations", len(class.Members()))
	}
	heritage := class.HeritageClauses()
	if len(heritage) != 1 ||
		heritage[0].Token() != tsgo.HeritageClauseTokenKindExtendsKeyword {
		t.Fatalf("map heritage = %#v, want nominal GoMapValue base", heritage)
	}
	bases := heritage[0].Types()
	if len(bases) != 1 ||
		bases[0].Expression().(tsgo.Identifier).Text() != "GoMapValue" ||
		len(bases[0].TypeArguments()) != 2 {
		t.Fatalf("map base = %#v, want GoMapValue<K,V>", bases)
	}
	constructor := class.Members()[0].(tsgo.ConstructorDeclaration)
	body := constructor.Body().(tsgo.Block)
	if len(body.Statements()) != 1 ||
		body.Statements()[0].(tsgo.ExpressionStatement).
			Expression().(tsgo.CallExpression).
			Expression().Kind() != tsgo.SyntaxKindSuperKeyword {
		t.Fatalf("map constructor = %#v, want one super() call", body)
	}
}

func TestMapValueContractIsNominallyNonThenable(t *testing.T) {
	statement, err := Build(
		tsgo.NewFactory(),
		api.RuntimeMapValue,
		panicClassName(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	contract, ok := statement.(tsgo.ClassDeclaration)
	if !ok {
		t.Fatalf("definition = %T, want abstract ClassDeclaration", statement)
	}
	if len(contract.Members()) != 9 {
		t.Fatalf("members = %d, want eight abstract operations and Promise exclusion", len(contract.Members()))
	}
	modifiers := contract.Modifiers()
	if len(modifiers) != 2 ||
		modifiers[0].Kind() != tsgo.SyntaxKindExportKeyword ||
		modifiers[1].Kind() != tsgo.SyntaxKindAbstractKeyword {
		t.Fatalf("map-value modifiers = %#v, want export abstract", modifiers)
	}
	for index, member := range contract.Members()[:8] {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok || method.Body() != nil ||
			len(method.Modifiers()) != 1 ||
			method.Modifiers()[0].Kind() != tsgo.SyntaxKindAbstractKeyword {
			t.Fatalf("map-value operation %d = %#v, want abstract method", index, member)
		}
	}
	property, ok := contract.Members()[8].(tsgo.PropertyDeclaration)
	if !ok {
		t.Fatalf("Promise exclusion = %T, want PropertyDeclaration", contract.Members()[8])
	}
	modifiers = property.Modifiers()
	if property.Name().(tsgo.Identifier).Text() != "then" ||
		len(modifiers) != 3 ||
		modifiers[0].Kind() != tsgo.SyntaxKindDeclareKeyword ||
		modifiers[1].Kind() != tsgo.SyntaxKindPrivateKeyword ||
		modifiers[2].Kind() != tsgo.SyntaxKindReadonlyKeyword ||
		property.PostfixToken() == nil ||
		property.PostfixToken().Kind() != tsgo.SyntaxKindQuestionToken ||
		property.Type().Kind() != tsgo.SyntaxKindNeverKeyword ||
		property.Initializer() != nil {
		t.Fatalf("map-value Promise exclusion = %#v", property)
	}
}

func TestBuildCreatesGenericMapValueTransport(t *testing.T) {
	statement, err := Build(
		tsgo.NewFactory(),
		api.RuntimeMapStore,
		panicClassName(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	declaration, ok := statement.(tsgo.FunctionDeclaration)
	if !ok {
		t.Fatalf("definition = %T, want FunctionDeclaration", statement)
	}
	contract, err := api.RuntimeContract(api.RuntimeMapStore)
	if err != nil {
		t.Fatal(err)
	}
	if declaration.Name().Text() != contract.ExportedName() ||
		len(declaration.TypeParameters()) != 2 ||
		len(declaration.Parameters()) != 3 {
		t.Fatalf(
			"map transport = %q with %d type parameters and %d parameters",
			declaration.Name().Text(),
			len(declaration.TypeParameters()),
			len(declaration.Parameters()),
		)
	}
	body := declaration.Body().(tsgo.Block).Statements()
	if len(body) != 1 {
		t.Fatalf("map transport statements = %d, want one", len(body))
	}
	call := body[0].(tsgo.ExpressionStatement).Expression().(tsgo.CallExpression)
	access := call.Expression().(tsgo.PropertyAccessExpression)
	if access.Expression().(tsgo.Identifier).Text() != valuesName ||
		access.Name().(tsgo.Identifier).Text() != "set" ||
		len(call.Arguments()) != 2 ||
		call.Arguments()[0].(tsgo.Identifier).Text() != keyName ||
		call.Arguments()[1].(tsgo.Identifier).Text() != valueName {
		t.Fatalf("map transport body = %#v, want values.set(key, value)", body)
	}
}

func TestClearSurfaceBelongsToCompleteMapContract(t *testing.T) {
	factory := tsgo.NewFactory()
	statement, err := Build(
		factory,
		api.RuntimeMap,
		panicClassName(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	class := statement.(tsgo.ClassDeclaration)
	if len(class.Members()) != 11 {
		t.Fatalf("map members = %d, want the complete value contract", len(class.Members()))
	}
	method := class.Members()[9].(tsgo.MethodDeclaration)
	if method.Name().(tsgo.Identifier).Text() != "clear" {
		t.Fatalf("map member = %v, want clear", method.Name())
	}
}

func TestKeysSurfaceBelongsToCompleteMapContract(t *testing.T) {
	factory := tsgo.NewFactory()
	statement, err := Build(
		factory,
		api.RuntimeMap,
		panicClassName(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	class := statement.(tsgo.ClassDeclaration)
	if len(class.Members()) != 11 {
		t.Fatalf("map members = %d, want the complete value contract", len(class.Members()))
	}
	method := class.Members()[10].(tsgo.MethodDeclaration)
	if method.Name().(tsgo.Identifier).Text() != "keys" {
		t.Fatalf("map member = %v, want keys", method.Name())
	}
}

func TestBuildCreatesOneStaticHashPrimitiveOwner(t *testing.T) {
	statement, err := Build(
		tsgo.NewFactory(),
		api.RuntimeMapHash,
		panicClassName(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	class, ok := statement.(tsgo.ClassDeclaration)
	if !ok {
		t.Fatalf("definition = %T, want ClassDeclaration", statement)
	}
	contract, err := api.RuntimeContract(api.RuntimeMapHash)
	if err != nil {
		t.Fatal(err)
	}
	if class.Name().Text() != contract.ExportedName() ||
		len(class.Members()) != 9 {
		t.Fatalf(
			"hash owner = %q with %d members",
			class.Name().Text(),
			len(class.Members()),
		)
	}
	for _, member := range class.Members()[2:8] {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok {
			t.Fatalf("hash owner member = %T, want static method", member)
		}
		static := false
		for _, modifier := range method.Modifiers() {
			if modifier.Kind() == tsgo.SyntaxKindStaticKeyword {
				static = true
			}
		}
		if !static {
			t.Fatalf("hash owner method %v is not static", method.Name())
		}
	}
}

func TestRuntimeMapMutationGuardsOwnMissingAndNilWriteSemantics(t *testing.T) {
	statement, err := Build(
		tsgo.NewFactory(),
		api.RuntimeMap,
		panicClassName(t),
	)
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
	for _, member := range class.Members()[1:11] {
		method := member.(tsgo.MethodDeclaration)
		methods[method.Name().(tsgo.Identifier).Text()] = method
	}
	lookup := methods["lookup"].Body().(tsgo.Block).Statements()
	if len(lookup) != 5 {
		t.Fatalf("lookup statements = %d, want typed storage/value narrowing", len(lookup))
	}
	if _, ok := lookup[0].(tsgo.VariableStatement); !ok {
		t.Fatalf("lookup storage capture = %T", lookup[0])
	}
	missing, ok := lookup[1].(tsgo.IfStatement)
	if !ok || missing.Expression().Kind() != tsgo.SyntaxKindBinaryExpression {
		t.Fatalf("lookup missing-storage guard = %T", lookup[1])
	}
	missingReturn := missing.ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.PropertyAccessExpression)
	if missingReturn.Name().(tsgo.Identifier).Text() != zeroName {
		t.Fatal("missing lookup does not return the represented Go zero")
	}
	if _, ok := lookup[2].(tsgo.VariableStatement); !ok {
		t.Fatalf("lookup value capture = %T", lookup[2])
	}
	if _, ok := lookup[3].(tsgo.IfStatement); !ok {
		t.Fatalf("lookup missing-value guard = %T", lookup[3])
	}
	present := lookup[4].(tsgo.ReturnStatement).Expression()
	if present.Kind() != tsgo.SyntaxKindIdentifier {
		t.Fatalf("present lookup = %T, want narrowed scalar identifier", present)
	}
	lookupOK := methods["lookupOk"].Body().(tsgo.Block).Statements()
	if len(lookupOK) != 5 {
		t.Fatalf("lookupOk statements = %d, want typed storage/value narrowing", len(lookupOK))
	}
	for _, statement := range lookupOK {
		if statement.Kind() == tsgo.SyntaxKindNonNullExpression {
			t.Fatal("lookupOk retains a non-null assertion")
		}
	}

	store := methods["store"].Body().(tsgo.Block).Statements()
	if len(store) != 2 {
		t.Fatalf("store statements = %d, want nil guard and set", len(store))
	}
	nilGuard, ok := store[0].(tsgo.IfStatement)
	if !ok {
		t.Fatalf("store nil guard = %T", store[0])
	}
	panicCall, ok := nilGuard.ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.ExpressionStatement).
		Expression().(tsgo.CallExpression)
	if !ok ||
		panicCall.Expression().(tsgo.PropertyAccessExpression).
			Expression().(tsgo.Identifier).Text() != panicClassName(t) {
		t.Fatal("nil map store bypasses the shared panic ABI")
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
		api.RuntimeArray,
		api.RuntimeSlice,
	} {
		if _, err := Build(
			tsgo.NewFactory(),
			symbol,
			panicClassName(t),
		); err == nil {
			t.Fatalf("symbol %d was accepted by map owner", symbol)
		}
	}
}

func panicClassName(t *testing.T) string {
	t.Helper()
	contract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ExportedName()
}
