package maprepresentation

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNativeKeySpecializationExecutesExactMapSemantics(t *testing.T) {
	targetContext, key, value, pointer := nativeSpecializationContext(t)
	factory := targetContext.Factory()
	keyType := factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword)
	valueType := factory.TypeReferenceNode(factory.Identifier("Box"), nil)
	pointerType := factory.UnionTypeNode([]tsgo.TypeNode{
		valueType,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
	valueSource := nativeSpecializationSource(
		t,
		targetContext,
		"NativeBoxMap",
		types.NewMap(key, value),
		keyType,
		valueType,
	)
	pointerSource := nativeSpecializationSource(
		t,
		targetContext,
		"NativePointerMap",
		types.NewMap(key, pointer),
		keyType,
		pointerType,
	)
	t.Logf(
		"native specialization bytes: value=%d pointer=%d",
		len(valueSource),
		len(pointerSource),
	)
	for _, source := range []string{valueSource, pointerSource} {
		for _, required := range []string{
			"private readonly values: Map<string, ",
			"private static $copyValue",
			"goMapStore(values, key, ",
			"values.has(key)",
			"values.delete(key)",
			"Array.from(values.keys())",
		} {
			if !strings.Contains(source, required) {
				t.Fatalf("native-key specialization lacks %q:\n%s", required, source)
			}
		}
		for _, forbidden := range []string{
			"$hash(",
			"$equal(",
			"$copyKey(",
			"$find(",
			"buckets",
			"count",
			"GoDenseIndex",
			"GoMapHash",
			"private readonly values: Map<string, [",
			"values.set(key, [",
			" as ",
		} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("native-key specialization contains %q:\n%s", forbidden, source)
			}
		}
		store := specializationMethodSource(t, source, "store", "delete")
		lookup := specializationMethodSource(t, source, "lookup", "lookupOk")
		lookupOK := specializationMethodSource(t, source, "lookupOk", "store")
		if strings.Contains(store, "values.get(") ||
			strings.Contains(store, "entry === undefined") ||
			strings.Count(store, "goMapStore(values, key, ") != 1 ||
			strings.Contains(store, "values.set(key, ") ||
			strings.Count(store, "$copyValue(value)") != 1 {
			t.Fatalf("native store is not one copy-and-set operation:\n%s", store)
		}
		if strings.Contains(lookup, "entry[0]") ||
			strings.Contains(lookupOK, "entry[0]") {
			t.Fatalf("native lookup retains a value cell:\n%s\n%s", lookup, lookupOK)
		}
	}
	t.Log("native store work: map-get=0 map-set=1 value-copy=1 tuple-allocation=0 semantic-branches=0")
	typeScriptOutput := compileAndRunSpecialization(t, mapValueTestContract+`class Box {
    constructor(public value: number) {}
}

class GoPanic {
    static raiseRuntime(message: string): never { throw new Error(message); }
}
`+valueSource+pointerSource+`
const nilValues = NativeBoxMap.nil();
const firstMissing = nilValues.lookup("missing");
firstMissing.value = 99;
let nilStoreFailed = false;
try { nilValues.store("missing", new Box(1)); } catch { nilStoreFailed = true; }
console.log(nilValues.lookup("missing").value, nilValues.length(), nilValues.isNil(), nilStoreFailed);

const values = NativeBoxMap.make(0, []);
const source = new Box(10);
values.store("alpha", source);
source.value = 90;
const read = values.lookup("alpha");
read.value = 70;
const alias = values;
alias.store("beta", new Box(20));
alias.store("alpha", new Box(30));
console.log(values.lookup("alpha").value, values.lookup("beta").value, values.length());

const pointers = NativePointerMap.make(0, []);
pointers.store("nil", undefined);
const [stored, storedOK] = pointers.lookupOk("nil");
const [absent, absentOK] = pointers.lookupOk("absent");
console.log(stored === undefined, storedOK, absent === undefined, absentOK, pointers.length());

values.delete("alpha");
console.log(values.lookup("beta").value, values.length(), values.keys().sort().join(","));
values.clear();
console.log(values.length(), values.isNil(), values.lookup("beta").value);
`)
	goOutput := executeNativeKeyMapGo(t)
	if typeScriptOutput != goOutput {
		t.Fatalf("TypeScript output = %q, Go output = %q", typeScriptOutput, goOutput)
	}
}

func specializationMethodSource(
	t *testing.T,
	source string,
	name string,
	next string,
) string {
	t.Helper()
	start := strings.Index(source, "\n    "+name+"(")
	end := strings.Index(source, "\n    "+next+"(")
	if start < 0 || end <= start {
		t.Fatalf("specialization methods %s/%s are absent:\n%s", name, next, source)
	}
	return source[start:end]
}

func TestNativeMapKeySelectionUsesDirectPrimitiveCarrier(t *testing.T) {
	targetContext, _, _, _ := nativeSpecializationContext(t)
	pkg := types.NewPackage("example.com/native-map-key", "nativekey")
	namedString := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NamedString", nil),
		types.Typ[types.String],
		nil,
	)
	namedInt32 := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NamedInt32", nil),
		types.Typ[types.Int32],
		nil,
	)
	for _, testCase := range []struct {
		name       string
		sourceType types.Type
		direct     bool
	}{
		{name: "bool", sourceType: types.Typ[types.Bool], direct: true},
		{name: "int32", sourceType: types.Typ[types.Int32], direct: true},
		{name: "uint64", sourceType: types.Typ[types.Uint64], direct: true},
		{name: "string", sourceType: types.Typ[types.String], direct: true},
		{name: "float64", sourceType: types.Typ[types.Float64]},
		{name: "complex128", sourceType: types.Typ[types.Complex128]},
		{name: "pointer", sourceType: types.NewPointer(types.Typ[types.Int32])},
		{name: "named string", sourceType: namedString, direct: true},
		{name: "named int32", sourceType: namedInt32, direct: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			actual := nativeMapKey(targetContext, testCase.sourceType)
			if actual != testCase.direct {
				t.Fatalf("native map key = %t, want %t", actual, testCase.direct)
			}
		})
	}
}

func TestProjectedPrimitiveKeyUsesSpecialization(t *testing.T) {
	projectedContext, _, _, _ := nativeSpecializationContext(t)
	pkg := types.NewPackage("example.com/native-map-owner", "nativeowner")
	namedString := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NamedString", nil),
		types.Typ[types.String],
		nil,
	)
	projected, ok := Source(
		projectedContext,
		types.NewMap(namedString, types.Typ[types.Int32]),
	)
	if !ok || projected.Storage() != StorageNative {
		t.Fatalf(
			"projected primitive map storage = %d/%t, want native specialization",
			projected.Storage(),
			ok,
		)
	}

	identityContext, _, _, _ := nativeSpecializationContextWithNames(
		t,
		generatedNumericSpecializationNames{},
	)
	namedInt32 := types.NewNamed(
		types.NewTypeName(token.NoPos, pkg, "NamedInt32", nil),
		types.Typ[types.Int32],
		nil,
	)
	identity, ok := Source(
		identityContext,
		types.NewMap(namedInt32, types.Typ[types.Int32]),
	)
	if !ok || identity.Storage() != StorageScalar {
		t.Fatalf(
			"identity primitive map storage = %d/%t, want scalar owner",
			identity.Storage(),
			ok,
		)
	}
}

func nativeSpecializationSource(
	t *testing.T,
	targetContext api.Context,
	className string,
	mapType *types.Map,
	keyType tsgo.TypeNode,
	valueType tsgo.TypeNode,
) string {
	t.Helper()
	factory := targetContext.Factory()
	specialization, err := BuildSpecialization(
		targetContext,
		nil,
		className,
		mapType,
		keyType,
		keyType,
		valueType,
	)
	if err != nil {
		t.Fatal(err)
	}
	return printSpecialization(t, factory, factory.ClassDeclaration(
		nil,
		factory.Identifier(className),
		nil,
		specialization.HeritageClauses(),
		specialization.Members(),
	))
}

func nativeSpecializationContext(
	t *testing.T,
) (api.Context, types.Type, types.Type, types.Type) {
	return nativeSpecializationContextWithNames(t, staticSpecializationNames{})
}

func nativeSpecializationContextWithNames(
	t *testing.T,
	names api.Names,
) (api.Context, types.Type, types.Type, types.Type) {
	t.Helper()
	key := types.Typ[types.String]
	value := types.NewStruct(
		[]*types.Var{types.NewField(
			token.NoPos,
			nil,
			"value",
			types.Typ[types.Int32],
			false,
		)},
		nil,
	)
	pointer := types.NewPointer(value)
	values := nativeSpecializationValues{
		staticSpecializationValues: staticSpecializationValues{
			key:   key,
			value: value,
		},
		pointer: pointer,
	}
	targetContext, err := api.NewContext(
		api.RoleMapReceiver,
		token.NewFileSet(),
		types.NewPackage("example.com/native-map", "nativemap"),
		&types.Info{},
		types.SizesFor("gc", "amd64"),
		api.MemoryByteOrderLittleEndian,
		tsgo.NewFactory(),
		names,
		values,
		api.IntegerRepresentationNumber,
		api.EvaluationOrderPreserveGo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return targetContext, key, value, pointer
}

type generatedNumericSpecializationNames struct {
	staticSpecializationNames
}

func (generatedNumericSpecializationNames) DefinedValueRepresentation(
	*types.TypeName,
) (api.DefinedValueRepresentation, error) {
	return api.NewDefinedValueRepresentation(
		api.DefinedValueRepresentationGeneratedNumeric,
		api.NameReference{},
	)
}

type nativeSpecializationValues struct {
	staticSpecializationValues
	pointer types.Type
}

func (v nativeSpecializationValues) Zero(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	if sourceType != v.pointer {
		return v.staticSpecializationValues.Zero(context, source, sourceType)
	}
	return api.DirectExpression(context.Factory().VoidExpression(
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
	)), nil
}

func (v nativeSpecializationValues) Transfer(
	context api.Context,
	source ast.Node,
	actualType types.Type,
	destinationType types.Type,
	mode api.ValueTransferMode,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if actualType != v.pointer || destinationType != v.pointer {
		return v.staticSpecializationValues.Transfer(
			context,
			source,
			actualType,
			destinationType,
			mode,
			value,
		)
	}
	if mode != api.ValueTransferCopy {
		panic("unexpected native pointer transfer")
	}
	return value, nil
}

func executeNativeKeyMapGo(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/native-map-test\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "main.go"),
		[]byte(`package main

import (
	"fmt"
	"sort"
)

type Box struct{ value int32 }

func nilStoreFails(values map[string]Box) (failed bool) {
	defer func() { failed = recover() != nil }()
	values["missing"] = Box{value: 1}
	return false
}

func main() {
	var nilValues map[string]Box
	firstMissing := nilValues["missing"]
	firstMissing.value = 99
	fmt.Println(nilValues["missing"].value, len(nilValues), nilValues == nil, nilStoreFails(nilValues))

	values := make(map[string]Box)
	source := Box{value: 10}
	values["alpha"] = source
	source.value = 90
	read := values["alpha"]
	read.value = 70
	alias := values
	alias["beta"] = Box{value: 20}
	alias["alpha"] = Box{value: 30}
	fmt.Println(values["alpha"].value, values["beta"].value, len(values))

	pointers := make(map[string]*Box)
	pointers["nil"] = nil
	stored, storedOK := pointers["nil"]
	absent, absentOK := pointers["absent"]
	fmt.Println(stored == nil, storedOK, absent == nil, absentOK, len(pointers))

	delete(values, "alpha")
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Println(values["beta"].value, len(values), keys[0])
	clear(values)
	fmt.Println(len(values), values == nil, values["beta"].value)
}
`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		filepath.Join(runtime.GOROOT(), "bin", "go"),
		"run",
		".",
	)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("execute native-key Go differential: %v\n%s", err, output)
	}
	return string(output)
}
