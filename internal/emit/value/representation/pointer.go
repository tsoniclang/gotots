package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
)

func (owner Owner) Pointee(
	context api.Context,
	source ast.Node,
	pointerSourceType types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	_, element, ok := pointertype.Resolve(pointerSourceType)
	defined, definedOK := definedtype.ResolvePointer(pointerSourceType)
	if definedOK {
		sourcePointer, _ := defined.Pointer()
		element = sourcePointer.Elem()
		ok = true
	}
	if !ok {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	if definedOK {
		var err error
		pointer, err = defined.Project(context, pointer)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	targetElement, err := owner.children.RepresentedType(
		context.WithRole(api.RoleUnaryOperand),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	guarded, err := pointermarker.Guard(context, pointer)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	loaded, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolLoadPointer,
		[]api.TypeEmission{targetElement},
		[]api.ExpressionEmission{guarded},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return owner.Transfer(
		context,
		source,
		element,
		element,
		api.ValueTransferCopy,
		loaded,
	)
}
