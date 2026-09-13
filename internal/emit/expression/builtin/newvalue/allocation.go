package newvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
)

func allocate(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	element types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if array, ok := arrayvalue.Resolve(context, element); ok {
		return array.PointerToValue(context, children, source, value)
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleCallArgument),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return pointermarker.Operation(
		context,
		tsoniccore.SymbolAllocatePointer,
		[]api.TypeEmission{targetElement},
		[]api.ExpressionEmission{value},
	)
}
