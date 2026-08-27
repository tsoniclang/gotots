package methodcall

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type RecoveryInvocation struct {
	selection Selection
	reference api.RecoveryCallableReference
	requests  []api.RootRequest
	provider  bool
	direct    bool
}

func (s Selection) ResolveRecovery(
	context api.Context,
) (RecoveryInvocation, error) {
	profileEffect, profiled, profileRequests, err :=
		providerboundary.ResolveStatefulMethodEffect(context, s.owner)
	if err != nil {
		return RecoveryInvocation{}, err
	}
	if profiled {
		if err := providerboundary.RequireProviderEffect(
			context,
			s.owner.FullName()+" recovery",
			profileEffect,
		); err != nil {
			return RecoveryInvocation{}, err
		}
		return RecoveryInvocation{
			selection: s,
			requests:  profileRequests,
			provider:  true,
			direct:    true,
		}, nil
	}
	reference, provider, err := context.Names().RecoveryCallable(s.owner)
	if err != nil {
		return RecoveryInvocation{}, err
	}
	return RecoveryInvocation{
		selection: s,
		reference: reference,
		provider:  provider,
		direct:    !provider,
	}, nil
}

func (i RecoveryInvocation) Call(
	context api.Context,
	receiver tsgo.Expression,
	sourceArguments []tsgo.Expression,
	recovery tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	if recovery == nil {
		return nil, nil, invariant(
			context,
			"selected-method recovery invocation has no authority",
		)
	}
	if i.direct {
		call, requests, err := i.selection.DeferredCall(
			context,
			receiver,
			sourceArguments,
			recovery,
		)
		return call, api.CombineRequests(i.requests, requests), err
	}
	if !i.provider {
		return nil, nil, invariant(
			context,
			"selected-method recovery invocation is invalid",
		)
	}
	arguments := append([]tsgo.Expression{receiver}, sourceArguments...)
	arguments = append(arguments, recovery)
	call := context.Factory().CallExpression(
		i.reference.Expression(context.Factory()),
		nil,
		i.selection.typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	)
	return call,
		api.CombineRequests(
			i.selection.Requests(),
			i.reference.Requests(),
			i.requests,
		),
		nil
}
