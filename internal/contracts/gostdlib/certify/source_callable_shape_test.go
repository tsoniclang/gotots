package certify

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestSourceCallableParameterCountPreservesGoShape(t *testing.T) {
	parameters := types.NewTuple(
		types.NewVar(token.NoPos, nil, "left", types.Typ[types.Int]),
		types.NewVar(token.NoPos, nil, "right", types.Typ[types.Int]),
	)
	named := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "Counter", nil),
		types.NewStruct(nil, nil),
		nil,
	)
	tests := []struct {
		name     string
		receiver *types.Var
		params   *types.Tuple
		access   gostdlib.AccessKind
		expected int
	}{
		{name: "zero parameter function", access: gostdlib.AccessExport},
		{
			name:     "package function",
			params:   parameters,
			access:   gostdlib.AccessExport,
			expected: 2,
		},
		{
			name:     "value receiver",
			receiver: types.NewVar(token.NoPos, nil, "receiver", named),
			params:   parameters,
			access:   gostdlib.AccessInstanceMethod,
			expected: 2,
		},
		{
			name:     "pointer receiver",
			receiver: types.NewVar(token.NoPos, nil, "receiver", types.NewPointer(named)),
			params:   parameters,
			access:   gostdlib.AccessStaticMethod,
			expected: 3,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			signature := types.NewSignatureType(
				test.receiver,
				nil,
				nil,
				test.params,
				types.NewTuple(),
				false,
			)
			actual, err := sourceCallableParameterCount(signature, test.access)
			if err != nil || actual != test.expected {
				t.Fatalf("parameter count = %d, %v; want %d", actual, err, test.expected)
			}
		})
	}
}

func TestSourceCallableArityRejectsHiddenProviderParameter(t *testing.T) {
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int])),
		types.NewTuple(),
		false,
	)
	err := verifySourceCallableShape(
		"example.com/provider|Function",
		signature,
		gostdlib.AccessExport,
		2,
		0,
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"target has 2 value parameters, selected Go shape requires 1",
	) {
		t.Fatalf("hidden-parameter error = %v", err)
	}
}

func TestSourceCallableShapeRejectsHiddenProviderTypeParameter(t *testing.T) {
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "Value", nil),
		types.Universe.Lookup("any").Type(),
	)
	signature := types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{parameter},
		types.NewTuple(types.NewVar(token.NoPos, nil, "value", parameter)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "result", parameter)),
		false,
	)
	err := verifySourceCallableShape(
		"example.com/provider|Identity",
		signature,
		gostdlib.AccessExport,
		1,
		2,
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"target has 2 type parameters, selected Go shape requires 1",
	) {
		t.Fatalf("hidden-type-parameter error = %v", err)
	}
}
