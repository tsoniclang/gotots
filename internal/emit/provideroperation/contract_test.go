package provideroperation

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestSelectionMapsEveryProviderOperation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source gostdlib.GenericOperationKind
		target api.GenericOperation
	}{
		{gostdlib.GenericOperationCopy, api.GenericOperationCopy},
		{gostdlib.GenericOperationZero, api.GenericOperationZero},
		{gostdlib.GenericOperationEqual, api.GenericOperationEqual},
		{gostdlib.GenericOperationBinaryLess, api.GenericOperationBinaryLess},
		{gostdlib.GenericOperationConvert, api.GenericOperationConvert},
		{gostdlib.GenericOperationMapConstruct, api.GenericOperationMapConstruct},
		{gostdlib.GenericOperationToStorage, api.GenericOperationToStorage},
		{gostdlib.GenericOperationFromStorage, api.GenericOperationFromStorage},
		{gostdlib.GenericOperationToContainerStorage, api.GenericOperationToContainerStorage},
		{gostdlib.GenericOperationFromContainerStorage, api.GenericOperationFromContainerStorage},
		{gostdlib.GenericOperationInterfaceAssertOK, api.GenericOperationInterfaceAssertOK},
	}
	for _, testCase := range cases {
		selected, err := Selection(testCase.source)
		if err != nil {
			t.Fatalf("Selection(%q): %v", testCase.source, err)
		}
		if selected.Operation() != testCase.target {
			t.Fatalf(
				"Selection(%q) = %v, want %v",
				testCase.source,
				selected.Operation(),
				testCase.target,
			)
		}
	}
	if _, err := Selection(gostdlib.GenericOperationInvalid); err == nil {
		t.Fatal("invalid provider operation passed")
	} else {
		var contractError *ContractError
		if !errors.As(err, &contractError) {
			t.Fatalf("invalid operation error = %T, want *ContractError", err)
		}
	}
}

func TestSignatureResolvesProviderTypeReferences(t *testing.T) {
	t.Parallel()
	owner := genericFunction(t)
	document := gostdlib.GenericOperationDocument{
		Kind: gostdlib.GenericOperationConvert,
		Parameters: []gostdlib.ContractTypeDocument{
			gostdlib.ContractTypeParameterReference(0),
			gostdlib.ContractCallableParameterReference(1),
			gostdlib.ContractSliceReference(
				gostdlib.ContractTypeParameterReference(0),
			),
			gostdlib.ContractMapReference(
				gostdlib.ContractIntReference(),
				gostdlib.ContractTypeParameterReference(0),
			),
		},
		Results: []gostdlib.ContractTypeDocument{
			gostdlib.ContractBoolReference(),
		},
	}
	signature, err := Signature(owner, document)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := types.TypeString(signature, nil), "func(T, int, []T, map[int]T) bool"; got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}

	invalid := document
	invalid.Parameters = []gostdlib.ContractTypeDocument{
		gostdlib.ContractTypeParameterReference(1),
	}
	if _, err := Signature(owner, invalid); err == nil {
		t.Fatal("out-of-range type parameter passed")
	} else {
		var contractError *ContractError
		if !errors.As(err, &contractError) {
			t.Fatalf("invalid signature error = %T, want *ContractError", err)
		}
	}
}

func genericFunction(t *testing.T) *types.Func {
	t.Helper()
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(
		fileSet,
		"fixture.go",
		"package fixture; func Transform[T any](value T, count int) {}",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	configuration := types.Config{}
	checked, err := configuration.Check(
		"example.test/fixture",
		fileSet,
		[]*ast.File{file},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	owner, ok := checked.Scope().Lookup("Transform").(*types.Func)
	if !ok {
		t.Fatal("Transform is not a function")
	}
	return owner
}
