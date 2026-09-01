package scalar_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	scalarcontract "github.com/tsoniclang/gotots/internal/emit/runtime/scalar"
)

func TestSharedDeclarationSelectsOnlyExactTargetNeutralPrimitives(t *testing.T) {
	number32 := scalarABI(
		t,
		api.IntegerRepresentationNumber,
		api.NativeIntegerWidth32,
	)
	number64 := scalarABI(
		t,
		api.IntegerRepresentationNumber,
		api.NativeIntegerWidth64,
	)
	bigint64 := scalarABI(
		t,
		api.IntegerRepresentationBigInt,
		api.NativeIntegerWidth64,
	)
	for _, testCase := range []struct {
		name   string
		alias  api.PrimitiveAlias
		abi    api.ScalarABI
		export string
	}{
		{name: "number int32", alias: api.PrimitiveInt32, abi: number64, export: "int32"},
		{name: "number int64", alias: api.PrimitiveInt64, abi: number64},
		{name: "bigint int64", alias: api.PrimitiveInt64, abi: bigint64, export: "int64"},
		{name: "native 32", alias: api.PrimitiveInt, abi: number32, export: "int32"},
		{name: "native 64 number", alias: api.PrimitiveInt, abi: number64},
		{name: "native 64 bigint", alias: api.PrimitiveInt, abi: bigint64, export: "int64"},
		{name: "float32", alias: api.PrimitiveFloat32, abi: number64, export: "float32"},
		{name: "string", alias: api.PrimitiveString, abi: number64},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			declaration, selected, err := scalarcontract.SharedDeclaration(
				testCase.alias,
				testCase.abi,
			)
			if err != nil {
				t.Fatal(err)
			}
			if selected != (testCase.export != "") {
				t.Fatalf("selected = %v, want %v", selected, testCase.export != "")
			}
			if selected && (declaration.Export() != testCase.export ||
				declaration.Module() != "@tsonic/core/types.js") {
				t.Fatalf("declaration = %q from %q", declaration.Export(), declaration.Module())
			}
		})
	}
}

func TestSharedDeclarationRejectsInvalidAlias(t *testing.T) {
	if _, _, err := scalarcontract.SharedDeclaration(
		api.PrimitiveInvalid,
		scalarABI(
			t,
			api.IntegerRepresentationNumber,
			api.NativeIntegerWidth64,
		),
	); err == nil {
		t.Fatal("invalid primitive alias was accepted")
	}
}

func scalarABI(
	t *testing.T,
	representation api.IntegerRepresentation,
	width api.NativeIntegerWidth,
) api.ScalarABI {
	t.Helper()
	abi, err := api.NewScalarABI(representation, width)
	if err != nil {
		t.Fatal(err)
	}
	return abi
}
