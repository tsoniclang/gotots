package newvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
)

func allocate(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	element types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
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
