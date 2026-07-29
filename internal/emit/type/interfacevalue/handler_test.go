package interfacevalue

import (
	"go/token"
	"go/types"
	"testing"
)

func TestResolveDistinguishesTypeParameterConstraintFromInterfaceValue(
	t *testing.T,
) {
	contract := types.NewInterfaceType(nil, nil).Complete()
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		contract,
	)
	if _, ok := Resolve(parameter); ok {
		t.Fatal("type parameter was classified as a represented interface value")
	}
	if _, ok := Resolve(contract); !ok {
		t.Fatal("actual interface type was not classified as an interface value")
	}
}
