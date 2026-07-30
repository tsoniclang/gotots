package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildIndexAddressCapability(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
	modifiers []tsgo.ModifierLike,
	signature *types.Signature,
	selection api.GenericOperationSelection,
) (tsgo.Statement, []api.RootRequest, bool, error) {
	receiver, index, element, selected :=
		api.GenericIndexAddressOperation(selection, signature)
	if !selected {
		return nil, nil, false, nil
	}
	if !integerType(index) || context.IsCooperative() {
		return nil, nil, true, shapeError(
			context,
			api.GenericOperationIndexAddress,
		)
	}
	signatureRole := api.RoleFileDeclaration
	if artifact.Placement() == api.GeneratedArtifactPlacementLexical {
		signatureRole = api.RoleLocalDeclaration
	}
	target, err := callable.EmitAdapter(
		context.WithRole(signatureRole),
		children,
		nil,
		signature,
	)
	if err != nil {
		return nil, nil, true, err
	}
	parameters := target.ParameterReferences(context.Factory())
	value, handled, err := emitIndexAddressValue(
		context,
		children,
		receiver,
		element,
		parameters,
	)
	if err != nil {
		return nil, nil, true, err
	}
	if !handled {
		return nil, nil, true, shapeError(
			context,
			api.GenericOperationIndexAddress,
		)
	}
	body := append(
		value.Before(),
		context.Factory().ReturnStatement(value.Value()),
	)
	return context.Factory().FunctionDeclaration(
			modifiers,
			nil,
			context.Factory().Identifier(artifact.TargetName()),
			nil,
			target.Parameters(),
			target.Result(),
			context.Factory().Block(body, true),
		), api.CombineRequests(
			target.Requests(),
			value.Requests(),
		), true, nil
}

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
