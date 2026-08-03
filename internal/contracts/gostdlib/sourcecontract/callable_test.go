package sourcecontract

import (
	"go/token"
	"go/types"
	"testing"
)

func TestDirectCallableParameterExcludesNamedFunctionValues(t *testing.T) {
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int])),
		types.NewTuple(types.NewVar(token.NoPos, nil, "ok", types.Typ[types.Bool])),
		false,
	)
	if _, direct := DirectCallableParameterSignature(signature); !direct {
		t.Fatal("unnamed function type was not classified as a direct callable")
	}
	named := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "Sequence", nil),
		signature,
		nil,
	)
	if _, direct := DirectCallableParameterSignature(named); direct {
		t.Fatal("named function value was classified as a direct callable")
	}
}
