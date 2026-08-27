package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
)

func Inline(
	context api.Context,
	children api.ChildEmitter,
	selection api.GenericOperationSelection,
	signature *types.Signature,
) (api.ExpressionEmission, bool, error) {
	if !selection.Valid() || signature == nil {
		return api.ExpressionEmission{}, true, shapeError(
			context,
			selection.Operation(),
		)
	}
	operation := selection.Operation()
	if operation == api.GenericOperationConstraintMethod ||
		operation == api.GenericOperationDeferredCallableRegistry {
		return api.ExpressionEmission{}, false, nil
	}
	if storage, handled, err := inlineStorageCapability(
		context,
		children,
		signature,
		selection,
	); handled {
		return storage, true, err
	}
	var target callable.SignatureEmission
	var err error
	if operation == api.GenericOperationInterfaceAssert ||
		operation == api.GenericOperationInterfaceAssertOK {
		target, err = callable.EmitAdapterWithRootInterfaceParameter(
			context.WithRole(api.RoleCallArgument),
			children,
			nil,
			signature,
		)
	} else {
		target, err = callable.EmitAdapter(
			context.WithRole(api.RoleCallArgument),
			children,
			nil,
			signature,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	parameters := target.ParameterReferences(context.Factory())
	var value api.ExpressionEmission
	if receiver, _, element, selected :=
		api.GenericIndexAddressOperation(selection, signature); selected {
		var handled bool
		value, handled, err = emitIndexAddressValue(
			context,
			children,
			receiver,
			element,
			parameters,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		if !handled {
			return api.ExpressionEmission{}, true, shapeError(context, operation)
		}
	} else {
		value, err = emitValue(
			context.WithRole(api.RoleFunctionLiteralBody),
			children,
			selection,
			signature,
			parameters,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	return inlineArrow(context, target, signature, value), true, nil
}

func inlineArrow(
	context api.Context,
	target callable.SignatureEmission,
	signature *types.Signature,
	value api.ExpressionEmission,
) api.ExpressionEmission {
	body := value.Before()
	if signature.Results().Len() == 0 {
		body = append(body, context.Factory().ExpressionStatement(value.Value()))
	} else {
		body = append(body, context.Factory().ReturnStatement(value.Value()))
	}
	arrow := context.Factory().ArrowFunction(
		nil,
		nil,
		target.Parameters(),
		target.Result(),
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(body, true),
	)
	return api.DirectExpression(
		arrow,
		api.CombineRequests(target.Requests(), value.Requests())...,
	)
}
