package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitIndexAddressValue(
	context api.Context,
	children api.ChildEmitter,
	receiver types.Type,
	element types.Type,
	parameters []tsgo.Expression,
) (api.ExpressionEmission, bool, error) {
	if _, sliceElement, ok := slicevalue.Resolve(receiver); ok {
		if !types.Identical(sliceElement, element) {
			return api.ExpressionEmission{}, true, shapeError(
				context,
				api.GenericOperationIndexAddress,
			)
		}
		value, err := slicevalue.Address(
			context.WithRole(api.RoleFunctionBody),
			children,
			nil,
			element,
			api.DirectExpression(parameters[0]),
			api.DirectExpression(parameters[1]),
		)
		return value, true, err
	}
	_, pointedType, pointerOK := pointertype.Resolve(receiver)
	if !pointerOK {
		return api.ExpressionEmission{}, false, nil
	}
	array, arrayOK := arrayvalue.Resolve(context, pointedType)
	if !arrayOK {
		return api.ExpressionEmission{}, false, nil
	}
	if !types.Identical(array.ElementType(), element) {
		return api.ExpressionEmission{}, true, shapeError(
			context,
			api.GenericOperationIndexAddress,
		)
	}
	value, err := array.Address(
		context.WithRole(api.RoleFunctionBody),
		children,
		nil,
		api.DirectExpression(parameters[0]),
		api.DirectExpression(parameters[1]),
		true,
	)
	return value, true, err
}
