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
	return selectedMethodCall(
		context,
		method,
		memberSuffix,
		receiver,
		typeArguments,
		nil,
		arguments,
	)
}

func SelectedDeferredMethodCall(
	context api.Context,
	method *types.Func,
	memberSuffix string,
	receiver tsgo.Expression,
	typeArguments []tsgo.TypeNode,
	recovery tsgo.Expression,
	arguments []tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	if recovery == nil {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "selected deferred method has no recovery authority",
		}
	}
	target, err := context.Names().MethodTarget(method)
	if err != nil {
		return nil, nil, err
	}
	if target.ReceiverABI() == api.MethodReceiverABISourceRepresentation {
		reference, referenceErr := context.Names().DeferredCallable(
			method,
			memberSuffix,
		)
		if referenceErr != nil {
			return nil, nil, referenceErr
		}
		callArguments := []tsgo.Expression{recovery, receiver}
		callArguments = append(callArguments, arguments...)
		return context.Factory().CallExpression(
				reference.Expression(context.Factory()),
				nil,
				typeArguments,
				callArguments,
				tsgo.NodeFlagsNone,
			), api.CombineRequests(
				target.Requests(),
				reference.Requests(),
			), nil
	}
	return selectedMethodCall(
		context,
		method,
		memberSuffix+api.DeferredEntrySuffix,
		receiver,
		typeArguments,
		[]tsgo.Expression{recovery},
		arguments,
	)
}

func selectedMethodCall(
	context api.Context,
	method *types.Func,
	memberSuffix string,
	receiver tsgo.Expression,
	typeArguments []tsgo.TypeNode,
	prefixArguments []tsgo.Expression,
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
				append(prefixArguments, receiver),
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
		arguments = append(prefixArguments, arguments...)
	case api.MethodTargetEnvironmentFunction:
		callee = context.Factory().Identifier(name)
		arguments = append(
			append(prefixArguments, receiver),
			arguments...,
		)
	case api.MethodTargetSourceFunction:
		callee = context.Factory().Identifier(name)
		arguments = append(
			append(prefixArguments, receiver),
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
