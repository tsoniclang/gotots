package callable

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func SelectedMethodCall(
	context api.Context,
	method *types.Func,
	memberSuffix string,
	receiver tsgo.Expression,
	typeArguments []tsgo.TypeNode,
	arguments []tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	if method == nil || receiver == nil {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "selected-method invocation identity is invalid",
		}
	}
	method = method.Origin()
	target, err := context.Names().MethodTarget(method)
	if err != nil {
		return nil, nil, err
	}
	name := target.Name() + memberSuffix
	requests := target.Requests()
	var callee tsgo.Expression
	switch target.Kind() {
	case api.MethodTargetClassMember:
		owner := api.MethodReceiverTypeName(method)
		if owner == nil {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "class method has no class owner",
			}
		}
		if api.ValueReceiverTypeName(method) == nil {
			reference, referenceErr := context.Names().Reference(owner)
			if referenceErr != nil {
				return nil, nil, referenceErr
			}
			callee = context.Factory().PropertyAccessExpression(
				reference.Expression(context.Factory()),
				nil,
				context.Factory().Identifier(name),
				tsgo.NodeFlagsNone,
			)
			arguments = append(
				[]tsgo.Expression{receiver},
				arguments...,
			)
			requests = append(requests, reference.Requests()...)
			break
		}
		if len(typeArguments) != 0 {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "instance class method received method type arguments",
			}
		}
		callee = context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(name),
			tsgo.NodeFlagsNone,
		)
	case api.MethodTargetEnvironmentFunction:
		callee = context.Factory().Identifier(name)
		arguments = append(
			[]tsgo.Expression{receiver},
			arguments...,
		)
	default:
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "selected method has no target form",
		}
	}
	return context.Factory().CallExpression(
		callee,
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	), api.CombineRequests(requests), nil
}
