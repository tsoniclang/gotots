package semanticname

import (
	"errors"
	"go/token"
	"go/types"
	"testing"
)

func TestConcretizationSuffixIsSemanticAndExact(t *testing.T) {
	packageType := types.NewNamed(
		types.NewTypeName(
			token.NoPos,
			types.NewPackage("example.com/model", "model"),
			"Item",
			nil,
		),
		types.NewStruct(nil, nil),
		nil,
	)
	tests := []struct {
		name        string
		arguments   []types.Type
		synchronous bool
		want        string
	}{
		{name: "basic", arguments: []types.Type{types.Typ[types.Int32]}, want: "$int32"},
		{name: "aggregate", arguments: []types.Type{types.NewSlice(packageType)}, want: "$SliceOf_Named_example_u2e_com_u2f_model_Item"},
		{name: "effect", arguments: []types.Type{types.Typ[types.String]}, synchronous: true, want: "$string$synchronous"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ConcretizationSuffix(test.arguments, test.synchronous)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("suffix = %q, want %q", got, test.want)
			}
		})
	}
}

func TestOperationNameDescribesTheExactContract(t *testing.T) {
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		types.NewInterfaceType(nil, nil).Complete(),
	)
	signature := types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{parameter},
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", parameter),
			types.NewVar(token.NoPos, nil, "", parameter),
		),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", parameter)),
		false,
	)
	name, err := OperationName("binary_add", nil, signature)
	if err != nil {
		t.Fatal(err)
	}
	if name != "$go$binary_add$T0_T0_to_T0" {
		t.Fatalf("operation name = %q", name)
	}
}

func TestOperationNamesIgnoreTypeParameterSpelling(t *testing.T) {
	first := identitySignature("T")
	second := identitySignature("Value")
	if !types.Identical(first, second) {
		t.Fatal("renamed type-parameter signatures are not identical")
	}
	firstName, err := OperationName("copy", nil, first)
	if err != nil {
		t.Fatal(err)
	}
	secondName, err := OperationName("copy", nil, second)
	if err != nil {
		t.Fatal(err)
	}
	if firstName != secondName {
		t.Fatalf("renamed type parameters changed operation name: %q / %q", firstName, secondName)
	}
}

func TestOperationNamesPreserveTypeParameterPosition(t *testing.T) {
	first := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "K", nil),
		types.NewInterfaceType(nil, nil).Complete(),
	)
	second := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "V", nil),
		types.NewInterfaceType(nil, nil).Complete(),
	)
	_ = types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{first, second},
		types.NewTuple(),
		types.NewTuple(),
		false,
	)
	firstName, err := OperationName("copy", nil, unarySignature(first))
	if err != nil {
		t.Fatal(err)
	}
	secondName, err := OperationName("copy", nil, unarySignature(second))
	if err != nil {
		t.Fatal(err)
	}
	if firstName == secondName ||
		firstName != "$go$copy$T0_to_T0" ||
		secondName != "$go$copy$T1_to_T1" {
		t.Fatalf("positioned type-parameter names = %q / %q", firstName, secondName)
	}
}

func unarySignature(source types.Type) *types.Signature {
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "", source)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", source)),
		false,
	)
}

func identitySignature(parameterName string) *types.Signature {
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, parameterName, nil),
		types.NewInterfaceType(nil, nil).Complete(),
	)
	return types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{parameter},
		types.NewTuple(types.NewVar(token.NoPos, nil, "", parameter)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", parameter)),
		false,
	)
}

func TestConstraintMethodNamesPreserveMethodIdentity(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
		false,
	)
	read := types.NewFunc(token.NoPos, sourcePackage, "Read", signature)
	write := types.NewFunc(token.NoPos, sourcePackage, "Write", signature)
	readName, err := OperationName("constraint_method", read, signature)
	if err != nil {
		t.Fatal(err)
	}
	writeName, err := OperationName("constraint_method", write, signature)
	if err != nil {
		t.Fatal(err)
	}
	if readName != "$go$constraint_method$Read$void_to_int" ||
		writeName != "$go$constraint_method$Write$void_to_int" ||
		readName == writeName {
		t.Fatalf("constraint method names = %q / %q", readName, writeName)
	}
}

func TestSemanticModulesNeverContainDigestSegments(t *testing.T) {
	ownerPackage := types.NewPackage("example.com/math/vector", "vector")
	owner := types.NewFunc(
		token.NoPos,
		ownerPackage,
		"Add",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(),
			false,
		),
	)
	module, err := ConcretizationModule(owner)
	if err != nil {
		t.Fatal(err)
	}
	if module != "example_u2e_com/math/vector/Add" {
		t.Fatalf("module = %q", module)
	}
}

func TestMethodConcretizationNameIncludesReceiver(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	receiverName := types.NewTypeName(token.NoPos, sourcePackage, "Box", nil)
	receiverType := types.NewNamed(receiverName, types.NewStruct(nil, nil), nil)
	method := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"Apply",
		types.NewSignatureType(
			types.NewVar(token.NoPos, sourcePackage, "", receiverType),
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(),
			false,
		),
	)
	name, err := ConcretizationName(method, "$int32")
	if err != nil {
		t.Fatal(err)
	}
	if name != "Box$Apply$int32" {
		t.Fatalf("method concretization name = %q", name)
	}
}

func TestSemanticTypeNamesSeparateEmptyAndAuthoredStructTags(t *testing.T) {
	field := types.NewField(
		token.NoPos,
		types.NewPackage("example.com/model", "model"),
		"Value",
		types.Typ[types.Int],
		false,
	)
	withoutTag, err := Type(types.NewStruct([]*types.Var{field}, []string{""}))
	if err != nil {
		t.Fatal(err)
	}
	withTag, err := Type(types.NewStruct([]*types.Var{field}, []string{"empty"}))
	if err != nil {
		t.Fatal(err)
	}
	if withoutTag == withTag {
		t.Fatalf("distinct struct tags share semantic name %q", withoutTag)
	}
}

func TestSemanticTypeNamesSeparateNamedAndStructuralTypes(t *testing.T) {
	sourcePackage := types.NewPackage("SliceOf", "SliceOf")
	named := types.NewNamed(
		types.NewTypeName(token.NoPos, sourcePackage, "int", nil),
		types.Typ[types.Int],
		nil,
	)
	namedName, err := Type(named)
	if err != nil {
		t.Fatal(err)
	}
	sliceName, err := Type(types.NewSlice(types.Typ[types.Int]))
	if err != nil {
		t.Fatal(err)
	}
	if namedName == sliceName {
		t.Fatalf("named and structural types share semantic name %q", namedName)
	}
}

func TestSemanticNameFailuresAreTyped(t *testing.T) {
	_, err := Type(nil)
	var semanticError *Error
	if !errors.As(err, &semanticError) || semanticError.Reason == "" {
		t.Fatalf("semantic name error = %#v", err)
	}
}

func TestSemanticInterfaceNamesUseTheCompleteMethodSet(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
		false,
	)
	method := types.NewFunc(token.NoPos, sourcePackage, "Read", signature)
	direct := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	embedded := types.NewInterfaceType(nil, []types.Type{direct}).Complete()
	if !types.Identical(direct, embedded) {
		t.Fatal("interface fixture does not have one semantic method set")
	}
	directName, err := Type(direct)
	if err != nil {
		t.Fatal(err)
	}
	embeddedName, err := Type(embedded)
	if err != nil {
		t.Fatal(err)
	}
	if directName != embeddedName {
		t.Fatalf("identical interface names = %q / %q", directName, embeddedName)
	}
}

func TestSemanticStructNamesIgnoreExportedFieldPackage(t *testing.T) {
	firstPackage := types.NewPackage("example.com/first", "first")
	secondPackage := types.NewPackage("example.com/second", "second")
	first := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, firstPackage, "Value", types.Typ[types.Int], false),
	}, nil)
	second := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, secondPackage, "Value", types.Typ[types.Int], false),
	}, nil)
	if !types.Identical(first, second) {
		t.Fatal("struct fixture is not semantically identical")
	}
	firstName, err := Type(first)
	if err != nil {
		t.Fatal(err)
	}
	secondName, err := Type(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstName != secondName {
		t.Fatalf("identical struct names = %q / %q", firstName, secondName)
	}
}
