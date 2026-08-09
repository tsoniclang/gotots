package cooperative

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	runtimescheduler "github.com/tsoniclang/gotots/internal/emit/runtime/scheduler"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func SourceCall(
	context api.Context,
	source ast.Node,
	provider *types.Func,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	facet, err := api.NewSourceCallableFacet(provider)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return facetCall(context, source, facet, target, true)
}

func LiteralCall(
	context api.Context,
	source ast.Node,
	provider *ast.FuncLit,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	facet, err := context.FunctionLiteralCallableFacet(provider)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return facetCall(context, source, facet, target, true)
}

func GenericOperationCall(
	context api.Context,
	source ast.Node,
	operation *api.GenericOperationContract,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	facet, err := api.NewGenericOperationCallableFacet(operation)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return facetCall(context, source, facet, target, true)
}

func DetachedSourceCall(
	context api.Context,
	source ast.Node,
	provider *types.Func,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	facet, err := api.NewSourceCallableFacet(provider)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return facetCall(context, source, facet, target, false)
}

func DetachedLiteralCall(
	context api.Context,
	source ast.Node,
	provider *ast.FuncLit,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	facet, err := context.FunctionLiteralCallableFacet(provider)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return facetCall(context, source, facet, target, false)
}

func ValueCall(
	context api.Context,
	source ast.Node,
	signature *types.Signature,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return valueCall(context, source, signature, target, true, false)
}

func GeneratedValueCall(
	context api.Context,
	signature *types.Signature,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return valueCall(context, nil, signature, target, true, true)
}

func DetachedValueCall(
	context api.Context,
	source ast.Node,
	signature *types.Signature,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return valueCall(context, source, signature, target, false, false)
}

func valueCall(
	context api.Context,
	source ast.Node,
	signature *types.Signature,
	target api.ExpressionEmission,
	propagate bool,
	generated bool,
) (api.ExpressionEmission, error) {
	reference, observation, err := observeABI(context, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err = api.NewExpressionEmission(
		target.Before(),
		target.Value(),
		api.CombineRequests(
			target.Requests(),
			reference.Requests(),
			observation.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if context.ConcurrencySemantics() !=
		api.ConcurrencySemanticsCooperative {
		return target, nil
	}
	return await(context, source, target, propagate, generated)
}

func TransportSourceValue(
	context api.Context,
	source ast.Expr,
	provider *types.Func,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	facet, err := api.NewSourceCallableFacet(provider)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return transportProviderValue(
		context,
		source,
		facet,
		target,
	)
}

func TransportLiteralValue(
	context api.Context,
	source *ast.FuncLit,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	facet, err := context.FunctionLiteralCallableFacet(source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return transportProviderValue(
		context,
		source,
		facet,
		target,
	)
}

func Operation(
	context api.Context,
	source ast.Node,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	scheduler, err := context.Names().Runtime(
		api.RuntimeScheduler,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err = api.NewExpressionEmission(
		target.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(scheduler.Name()),
				nil,
				context.Factory().Identifier(runtimescheduler.BlockMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{target.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			target.Requests(),
			scheduler.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return await(context, source, target, true, false)
}

func ProviderProfileCall(
	context api.Context,
	source ast.Node,
	target api.ExpressionEmission,
	cooperative bool,
	detached bool,
) (api.ExpressionEmission, error) {
	if !cooperative {
		return target, nil
	}
	return await(context, source, target, !detached, false)
}

func ValueContract(
	context api.Context,
	signature *types.Signature,
) (bool, []api.RootRequest, error) {
	reference, observation, err := observeABI(context, signature)
	if err != nil {
		return false, nil, err
	}
	return observation.Cooperative(),
		api.CombineRequests(
			reference.Requests(),
			observation.Requests(),
		),
		nil
}

func SourceContract(
	context api.Context,
	provider *types.Func,
) (bool, []api.RootRequest, error) {
	facet, err := api.NewSourceCallableFacet(provider)
	if err != nil {
		return false, nil, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return false, nil, err
	}
	return observation.Cooperative(), observation.Requests(), nil
}

func LiteralContract(
	context api.Context,
	provider *ast.FuncLit,
) (bool, []api.RootRequest, error) {
	facet, err := context.FunctionLiteralCallableFacet(provider)
	if err != nil {
		return false, nil, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return false, nil, err
	}
	return observation.Cooperative(), observation.Requests(), nil
}

func SourceValueContract(
	context api.Context,
	provider *types.Func,
	signature *types.Signature,
) (bool, bool, []api.RootRequest, error) {
	facet, err := api.NewSourceCallableFacet(provider)
	if err != nil {
		return false, false, nil, err
	}
	return providerContract(context, facet, signature)
}

func GenericValueContract(
	context api.Context,
	provider api.CallableFacet,
	signature *types.Signature,
) (bool, bool, []api.RootRequest, error) {
	if _, source := provider.SourceFunction(); !source {
		return false, false, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic value provider facet is invalid",
		}
	}
	return providerContract(context, provider, signature)
}

func facetCall(
	context api.Context,
	source ast.Node,
	facet api.CallableFacet,
	target api.ExpressionEmission,
	propagate bool,
) (api.ExpressionEmission, error) {
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err = api.NewExpressionEmission(
		target.Before(),
		target.Value(),
		api.CombineRequests(
			target.Requests(),
			observation.Requests(),
		),
	)
	if err != nil || !observation.Cooperative() {
		return target, err
	}
	return await(context, source, target, propagate, false)
}

func transportProviderValue(
	context api.Context,
	source ast.Expr,
	provider api.CallableFacet,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	signature, ok := callable.Signature(context.TypesInfo().TypeOf(source))
	if ok {
		signature, ok = callable.ValueSignature(signature)
	}
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	_, _, contractRequests, err :=
		providerContract(context, provider, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	requests := api.CombineRequests(
		target.Requests(),
		contractRequests,
	)
	return api.NewExpressionEmission(
		target.Before(),
		target.Value(),
		requests,
	)
}

func providerContract(
	context api.Context,
	provider api.CallableFacet,
	signature *types.Signature,
) (bool, bool, []api.RootRequest, error) {
	providerObservation, err :=
		context.ObserveCooperativeCallable(provider)
	if err != nil {
		return false, false, nil, err
	}
	reference, abiObservation, err := observeABI(context, signature)
	if err != nil {
		return false, false, nil, err
	}
	requests := api.CombineRequests(
		providerObservation.Requests(),
		reference.Requests(),
		abiObservation.Requests(),
	)
	if providerObservation.Cooperative() {
		abiFacet, facetErr := context.CallableABIFacet(reference)
		if facetErr != nil {
			return false, false, nil, facetErr
		}
		selection, selectionErr :=
			api.NewCooperativeCallableRequest(abiFacet)
		if selectionErr != nil {
			return false, false, nil, selectionErr
		}
		requests = append(requests, selection)
	}
	return providerObservation.Cooperative(),
		abiObservation.Cooperative() ||
			providerObservation.Cooperative(),
		requests,
		nil
}

func observeABI(
	context api.Context,
	signature *types.Signature,
) (api.CallableABIReference, api.CooperativeCallableObservation, error) {
	signature, ok := callable.ValueSignature(signature)
	if !ok {
		return api.CallableABIReference{},
			api.CooperativeCallableObservation{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "callable ABI signature is unsupported",
			}
	}
	reference, err := callable.ABIReference(context, signature)
	if err != nil {
		return api.CallableABIReference{},
			api.CooperativeCallableObservation{},
			err
	}
	facet, err := context.CallableABIFacet(reference)
	if err != nil {
		return api.CallableABIReference{},
			api.CooperativeCallableObservation{},
			err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	return reference, observation, err
}

func await(
	context api.Context,
	source ast.Node,
	target api.ExpressionEmission,
	propagate bool,
	generated bool,
) (api.ExpressionEmission, error) {
	_, generatedOwner := context.ArtifactOwner().Generated()
	if source == nil && (!generated || !generatedOwner) {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "cooperative operation source is nil",
		}
	}
	requests := target.Requests()
	if propagate {
		requirement, err := context.CooperativeRequest()
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		requests = append(requests, requirement)
	}
	return api.NewExpressionEmission(
		target.Before(),
		context.Factory().AwaitExpression(target.Value()),
		requests,
	)
}
