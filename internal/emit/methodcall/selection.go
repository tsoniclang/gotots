package methodcall

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Selection struct {
	owner           *types.Func
	signature       *types.Signature
	facet           api.CallableFacet
	target          api.MethodTarget
	memberSuffix    string
	typeArguments   []tsgo.TypeNode
	concretized     bool
	concretization  api.GenericConcretizationReference
	openKernel      bool
	operations      []*api.GenericOperationContract
	capabilities    []genericabi.Binding[tsgo.Expression]
	statefulProfile bool
	profile         []gostdlib.ProviderCallableProfileInterface
	requests        []api.RootRequest
}

func Resolve(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	method *types.Func,
	signature *types.Signature,
) (Selection, error) {
	if method == nil ||
		signature == nil ||
		signature.Recv() == nil ||
		signature.TypeParams().Len() != 0 {
		return Selection{}, invariant(
			context,
			"selected-method signature is invalid",
		)
	}
	owner := method.Origin()
	declaration, ok := owner.Type().(*types.Signature)
	if !ok || declaration.Recv() == nil {
		return Selection{}, invariant(
			context,
			"selected-method owner has no receiver signature",
		)
	}
	selected := signature
	if selected.Recv() == nil {
		return Selection{}, invariant(
			context,
			"selected-method contextual signature is invalid",
		)
	}
	concrete, err := genericinstance.ConcreteCallableSignature(selected)
	if err != nil {
		return Selection{}, err
	}
	target, err := context.Names().MethodTarget(owner)
	if err != nil {
		return Selection{}, err
	}
	if declaration.RecvTypeParams().Len() == 0 {
		facet, facetErr := api.NewSourceCallableFacet(owner)
		if facetErr != nil {
			return Selection{}, facetErr
		}
		return resolveStatefulBoundary(context, Selection{
			owner:     owner,
			signature: concrete,
			facet:     facet,
			target:    target,
		})
	}
	operationSet, resolved, err :=
		context.ResolveGenericCallable(owner)
	if err != nil {
		return Selection{}, err
	}
	arguments := genericinstance.ReceiverTypeArguments(
		selected.Recv().Type(),
	)
	if !resolved ||
		arguments == nil ||
		arguments.Len() != len(operationSet.Parameters()) {
		return Selection{}, invariant(
			context,
			"generic selected-method representation is unresolved",
		)
	}
	facet, err := callable.SelectGenericMethod(context, owner)
	if err != nil {
		return Selection{}, err
	}
	typeArgumentList := api.TypeArgumentsFromGo(arguments)
	requiresConcretization, err :=
		context.GenericCallableRequiresConcretization(owner)
	if err != nil {
		return Selection{}, err
	}
	if requiresConcretization &&
		!typeArgumentList.ContainsGenericTypeParameter() {
		concretization, concreteErr := context.ResolveGenericConcretization(
			owner,
			typeArgumentList,
			selected,
		)
		if concreteErr != nil {
			return Selection{}, concreteErr
		}
		return resolveStatefulBoundary(context, Selection{
			owner:          owner,
			signature:      concrete,
			facet:          facet,
			target:         target,
			memberSuffix:   concretization.Suffix(),
			concretized:    true,
			concretization: concretization,
			requests:       concretization.Requests(),
		})
	}
	typeArguments, typeRequests, err :=
		genericinstance.EmitTypeArguments(
			context,
			children,
			source,
			method,
			typeArgumentList,
		)
	if err != nil {
		return Selection{}, err
	}
	var (
		capabilities       []genericabi.Binding[tsgo.Expression]
		projectionRequests []api.RootRequest
	)
	if requiresConcretization {
		capabilities, projectionRequests, err = genericinstance.EmitCapabilities(
			context,
			children,
			source,
			operationSet,
			typeArgumentList,
		)
		if err != nil {
			return Selection{}, err
		}
	}
	if api.ValueReceiverTypeName(owner) != nil {
		typeArguments = nil
	}
	return resolveStatefulBoundary(context, Selection{
		owner:         owner,
		signature:     concrete,
		facet:         facet,
		target:        target,
		typeArguments: slices.Clone(typeArguments),
		openKernel:    requiresConcretization,
		operations:    slices.Clone(operationSet.Operations()),
		capabilities:  slices.Clone(capabilities),
		requests: api.CombineRequests(
			typeRequests,
			projectionRequests,
		),
	})
}

func resolveStatefulBoundary(
	context api.Context,
	selection Selection,
) (Selection, error) {
	if !selection.target.ProviderBoundary() {
		return selection, nil
	}
	boundary, selected, err := providerboundary.ResolveStatefulMethodBoundary(
		context,
		selection.owner,
		selection.signature,
	)
	selection.requests = api.CombineRequests(
		selection.requests,
		boundary.Requests(),
	)
	if err != nil || !selected {
		if err != nil {
			return selection, err
		}
		if err := providerboundary.RequireProviderCallable(
			context,
			selection.owner,
		); err != nil {
			return Selection{}, err
		}
		return selection, nil
	}
	selection.statefulProfile = true
	selection.profile = boundary.Interfaces()
	return selection, nil
}

func (s Selection) Signature() *types.Signature {
	return s.signature
}

func (s Selection) Facet() api.CallableFacet {
	return s.facet
}

func (s Selection) Requests() []api.RootRequest {
	return api.CombineRequests(s.requests, s.target.Requests())
}

func (s Selection) Invoke(
	context api.Context,
	children api.ChildEmitter,
	receiver tsgo.Expression,
	sourceArguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	arguments, before, requests, err := s.providerArguments(
		context,
		children,
		sourceArguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, callRequests, err := s.Call(context, receiver, arguments)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		call,
		api.CombineRequests(requests, callRequests),
	)
}

func (s Selection) InvokeDeferred(
	context api.Context,
	children api.ChildEmitter,
	_ ast.Node,
	receiver tsgo.Expression,
	sourceArguments []tsgo.Expression,
	recovery tsgo.Expression,
) (api.ExpressionEmission, error) {
	if recovery == nil {
		return api.ExpressionEmission{}, invariant(
			context,
			"selected-method deferred invocation has no authority",
		)
	}
	call, err := s.RecoveryCall(
		context,
		children,
		receiver,
		sourceArguments,
		recovery,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return call, nil
}

func (s Selection) FromProviderResults(
	context api.Context,
	children api.ChildEmitter,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !s.target.ProviderBoundary() {
		return emission, nil
	}
	if s.statefulProfile {
		return providerboundary.FromProviderProfileResultsForBridge(
			context,
			children,
			nil,
			"",
			s.signature.Results(),
			s.profile,
			emission,
		)
	}
	return providerboundary.FromProviderResults(
		context,
		children,
		nil,
		"",
		s.signature.Results(),
		emission,
	)
}

func (s Selection) providerArguments(
	context api.Context,
	children api.ChildEmitter,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	if s.owner == nil || s.signature == nil ||
		s.signature.Params().Len() != len(sourceArguments) {
		return nil, nil, nil, invariant(
			context,
			"selected-method arguments do not match its plan",
		)
	}
	if !s.target.ProviderBoundary() {
		return slices.Clone(sourceArguments), nil, nil, nil
	}
	if s.statefulProfile {
		return providerboundary.ToProviderProfileArgumentsForBridge(
			context,
			children,
			s.signature.Params(),
			nil,
			"",
			s.profile,
			sourceArguments,
		)
	}
	return providerboundary.ToProviderArguments(
		context,
		children,
		s.signature.Params(),
		sourceArguments,
	)
}

func (s Selection) Call(
	context api.Context,
	receiver tsgo.Expression,
	sourceArguments []tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	if s.owner == nil ||
		s.signature == nil ||
		!s.facet.Valid() ||
		receiver == nil ||
		s.signature.Params().Len() != len(sourceArguments) {
		return nil, nil, invariant(
			context,
			"selected-method invocation does not match its plan",
		)
	}
	if s.concretized {
		arguments := append(
			[]tsgo.Expression{receiver},
			sourceArguments...,
		)
		return context.Factory().CallExpression(
			context.Factory().Identifier(s.concretization.Name()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		), s.Requests(), nil
	}
	arguments := sourceArguments
	suffix := s.memberSuffix
	if s.openKernel {
		sourceBindings, err := genericabi.SourceParameters(
			s.owner,
			sourceArguments,
		)
		if err != nil {
			return nil, nil, err
		}
		arguments, err = genericabi.JoinClassMethod(
			s.owner,
			s.operations,
			genericabi.Combine(s.capabilities, sourceBindings),
		)
		if err != nil {
			return nil, nil, err
		}
		suffix += api.GenericKernelSuffix
	}
	call, callRequests, err := callable.SelectedMethodCall(
		context,
		s.owner,
		suffix,
		receiver,
		s.typeArguments,
		arguments,
	)
	return call, api.CombineRequests(s.Requests(), callRequests), err
}

func (s Selection) DeferredCall(
	context api.Context,
	receiver tsgo.Expression,
	sourceArguments []tsgo.Expression,
	recovery tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	if recovery == nil {
		return nil, nil, invariant(
			context,
			"selected-method deferred invocation has no authority",
		)
	}
	if s.statefulProfile {
		arguments := append(slices.Clone(sourceArguments), recovery)
		call, callRequests, err := callable.SelectedMethodCall(
			context,
			s.owner,
			s.memberSuffix,
			receiver,
			s.typeArguments,
			arguments,
		)
		return call, api.CombineRequests(s.Requests(), callRequests), err
	}
	if s.concretized {
		concretizationNames, available :=
			context.Names().(api.GenericConcretizationNames)
		if !available {
			return nil, nil, invariant(
				context,
				"generic concretization names are unavailable",
			)
		}
		deferred, err :=
			concretizationNames.DeferredGenericConcretization(
				s.concretization.Concretization(),
			)
		if err != nil {
			return nil, nil, err
		}
		arguments := []tsgo.Expression{recovery, receiver}
		arguments = append(arguments, sourceArguments...)
		call := context.Factory().CallExpression(
			deferred.Expression(context.Factory()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		)
		return call, api.CombineRequests(
			s.Requests(),
			deferred.Requests(),
		), nil
	}
	arguments := sourceArguments
	suffix := s.memberSuffix
	if s.openKernel {
		sourceBindings, bindErr := genericabi.SourceParameters(
			s.owner,
			sourceArguments,
		)
		if bindErr != nil {
			return nil, nil, bindErr
		}
		arguments, bindErr = genericabi.JoinClassMethod(
			s.owner,
			s.operations,
			genericabi.Combine(s.capabilities, sourceBindings),
		)
		if bindErr != nil {
			return nil, nil, bindErr
		}
		suffix += api.GenericKernelSuffix
	}
	call, callRequests, err := callable.SelectedDeferredMethodCall(
		context,
		s.owner,
		suffix,
		receiver,
		s.typeArguments,
		recovery,
		arguments,
	)
	return call, api.CombineRequests(
		s.Requests(),
		callRequests,
	), err
}

func (s Selection) RecoveryCall(
	context api.Context,
	children api.ChildEmitter,
	receiver tsgo.Expression,
	sourceArguments []tsgo.Expression,
	recovery tsgo.Expression,
) (
	api.ExpressionEmission,
	error,
) {
	invocation, err := s.ResolveRecovery(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := slices.Clone(sourceArguments)
	var before []tsgo.Statement
	var argumentRequests []api.RootRequest
	if invocation.provider {
		arguments, before, argumentRequests, err = s.providerArguments(
			context,
			children,
			sourceArguments,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	call, requests, err := invocation.Call(
		context,
		receiver,
		arguments,
		recovery,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := api.NewExpressionEmission(
		before,
		call,
		api.CombineRequests(argumentRequests, requests),
	)
	return target, err
}

func invariant(context api.Context, reason string) error {
	return &api.InvariantError{
		Role:   context.Role(),
		Reason: reason,
	}
}
