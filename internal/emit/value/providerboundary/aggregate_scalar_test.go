package providerboundary

import (
	"errors"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestProviderAggregateScalarAdmission(test *testing.T) {
	for _, fieldType := range []types.Type{
		types.NewArray(types.Typ[types.Uint64], 256),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "Count", types.Typ[types.Uint64], false)}, nil),
		types.NewArray(types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "Count", types.Typ[types.Uint64], false)}, nil), 61),
	} {
		for _, product := range []api.IntegerRepresentation{api.IntegerRepresentationNumber, api.IntegerRepresentationFixed64BigInt} {
			context := scalarBoundaryContext(test, "amd64", product, api.IntegerRepresentationFixed64BigInt)
			err := validateAggregateScalarABI(context, fieldType)
			if product == api.IntegerRepresentationNumber {
				var boundary *api.UnsupportedError
				if !errors.As(err, &boundary) {
					test.Fatalf("differing %v carrier was not rejected: %v", fieldType, err)
				}
			} else if err != nil {
				test.Fatalf("matching %v carrier rejected: %v", fieldType, err)
			}
		}
	}
	context := scalarBoundaryContext(test, "amd64", api.IntegerRepresentationNumber, api.IntegerRepresentationFixed64BigInt)
	value := api.DirectExpression(context.Factory().Identifier("array"))
	for _, convert := range []func(api.Context, api.ChildEmitter, *types.Named, string, types.Type, api.ExpressionEmission) (api.ExpressionEmission, bool, error){FromProviderValue, ToProviderValue} {
		_, _, err := convert(context, nil, nil, "", types.NewArray(types.Typ[types.Uint64], 2), value)
		var boundary *api.UnsupportedError
		if !errors.As(err, &boundary) {
			test.Fatalf("production conversion bypassed aggregate admission: %v", err)
		}
	}
	if err := validateAggregateScalarABI(context, types.NewArray(types.Typ[types.Uint32], 4)); err != nil {
		test.Fatalf("matching narrow integer array rejected: %v", err)
	}
}
