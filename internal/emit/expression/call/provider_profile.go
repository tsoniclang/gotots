package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitProviderProfileFunction(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	owner *types.Func,
	signature *types.Signature,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, bool, []api.RootRequest, error) {
	selection, selected, err := providerboundary.ResolveCallableProfile(
		context,
		owner,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, nil, err
	}
	if !selected {
		return api.ExpressionEmission{}, false, selection.Requests(), nil
	}
	arguments, before, requests, err := emitArguments(
		context,
		children,
		source,
		signature,
		detached,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, nil, err
	}
	target, err := emitProviderProfileInvocation(
		context,
		children,
		source,
		signature,
		selection,
		arguments,
		before,
		requests,
		discarded,
		detached,
	)
	return target, true, nil, err
}

func emitProviderProfileMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	signature *types.Signature,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, bool, []api.RootRequest, error) {
	profile, selected, err := providerboundary.ResolveCallableProfile(
		context,
		method,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, nil, err
	}
	if !selected {
		return api.ExpressionEmission{}, false, profile.Requests(), nil
	}
	receiver, resolvedMethod, err := selectionvalue.DirectMethodReceiver(
		context,
		children,
		selector,
		selection,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, nil, err
	}
	if resolvedMethod != method {
		return api.ExpressionEmission{}, true, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		detached,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, nil, err
	}
	before := receiver.Before()
	receiverValue := receiver.Value()
	var receiverRequests []api.RootRequest
	if len(argumentBefore) != 0 || detached {
		receiverValue, receiverRequests, before, err = captureReceiver(
			context,
			receiver,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, nil, err
		}
	}
	before = append(before, argumentBefore...)
	arguments = append([]tsgo.Expression{receiverValue}, arguments...)
	target, err := emitProviderProfileInvocation(
		context,
		children,
		source,
		signature,
		profile,
		arguments,
		before,
		api.CombineRequests(
			receiver.Requests(),
			receiverRequests,
			argumentRequests,
		),
		discarded,
		detached,
	)
	return target, true, nil, err
}

func emitProviderProfileInvocation(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	selection providerboundary.CallableProfileSelection,
	arguments []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, error) {
	reference := selection.Reference()
	typeArguments := make([]tsgo.TypeNode, 0, len(reference.CanonicalTypeArguments()))
	for _, sourceType := range reference.CanonicalTypeArguments() {
		represented, err := interfacetype.EmitNonNil(
			context.WithRole(api.RoleCallTypeArgument),
			children,
			source,
			sourceType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		typeArguments = append(typeArguments, represented.Value())
		requests = append(requests, represented.Requests()...)
	}
	for _, object := range reference.CanonicalValues() {
		variable, ok := object.(*types.Var)
		if !ok {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "provider callable-profile canonical value is not a variable",
			}
		}
		value, err := context.Names().PackageVariable(variable)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		raw, err := api.NewExpressionEmission(
			nil,
			value.Expression(context.Factory()),
			value.Requests(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		converted, _, err := providerboundary.FromProviderValue(
			context,
			children,
			nil,
			"",
			variable.Type(),
			raw,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, converted.Before()...)
		arguments = append(arguments, converted.Value())
		requests = append(requests, converted.Requests()...)
	}
	for _, guardType := range reference.Guards() {
		guard, err := context.Names().InterfaceContract(guardType)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		arguments = append(
			arguments,
			context.Factory().Identifier(guard.GuardName()),
		)
		requests = append(requests, guard.Requests()...)
	}
	for _, contractType := range reference.Contracts() {
		contract, err := context.Names().InterfaceContract(contractType)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		arguments = append(
			arguments,
			context.Factory().Identifier(contract.ContractName()),
		)
		requests = append(requests, contract.Requests()...)
	}
	for _, bridge := range reference.FromProviderBridges() {
		arguments = append(arguments, bridge.Expression(context.Factory()))
		requests = append(requests, bridge.Requests()...)
	}
	target, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			requests,
			selection.Requests(),
			reference.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err = cooperativecall.ProviderProfileCall(
		context,
		source,
		target,
		reference.Profile().Effect().MaySuspend(),
		detached,
	)
	if err != nil || discarded {
		return target, err
	}
	return providerboundary.FromProviderProfileResults(
		context,
		children,
		signature.Results(),
		reference.Profile().CanonicalResults(),
		target,
	)
}
