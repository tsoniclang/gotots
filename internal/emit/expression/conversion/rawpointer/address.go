package rawpointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

func convertAddress(context api.Context, source ast.Node, value api.ExpressionEmission, fromRaw bool) (api.ExpressionEmission, error) {
	carrier, ok := integervalue.Describe(context.TypesSizes(), types.Typ[types.Uintptr])
	if !ok || carrier.Width() == 64 && !integervalue.UsesBigInt(context.IntegerRepresentation(), carrier) {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	symbol := tsoniccore.SymbolUint32
	if carrier.Width() == 64 {
		symbol = tsoniccore.SymbolUint64
	}
	reference, err := context.Names().TsonicCore(symbol)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	domain := api.DirectType(context.Factory().TypeReferenceNode(reference.EntityName(context.Factory()), nil), reference.Requests()...)
	abi, err := memorymarker.DataLayout(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	operation := tsoniccore.SymbolAddressIntegerToRawPointer
	if fromRaw {
		operation = tsoniccore.SymbolRawPointerToAddressInteger
	}
	return pointermarker.Operation(context, operation, []api.TypeEmission{domain}, []api.ExpressionEmission{value, abi})
}
