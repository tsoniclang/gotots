package providerinterfacebridge

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func profileContractMethod(
	context api.Context,
	children api.ChildEmitter,
	method *types.Func,
	certificate gostdlib.ProviderInterfaceMethod,
) (tsgo.MethodSignatureDeclaration, []api.RootRequest, error) {
	signature, ok := receiverFreeSignature(method)
	if !ok || certificate.Kind() != gostdlib.ProviderInterfaceMethodCallable ||
		certificate.Member() == "" || !certificate.Effect().Valid() {
		return nil, nil, shapeError("", "profile contract method is invalid")
	}
	target, err := callable.EmitABIAdapter(context, children, nil, signature)
	if err != nil {
		return nil, nil, err
	}
	result := api.DirectType(target.Result())
	if certificate.Effect().MaySuspend() {
		result, err = callable.AwaitableResult(context, target.Result())
		if err != nil {
			return nil, nil, err
		}
	}
	member := context.Factory().MethodSignatureDeclaration(
		nil,
		context.Factory().Identifier(certificate.Member()),
		nil,
		nil,
		target.Parameters(),
		result.Value(),
	)
	return member, api.CombineRequests(target.Requests(), result.Requests()), nil
}

func profileForwardMethod(
	context api.Context,
	children api.ChildEmitter,
	bridgeName string,
	bridgeType *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
	method *types.Func,
	certificate gostdlib.ProviderInterfaceMethod,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	signature, ok := receiverFreeSignature(method)
	if !ok {
		return nil, nil, shapeError(bridgeName, "profile forward method is invalid")
	}
	target, err := callable.EmitABIAdapter(context, children, nil, signature)
	if err != nil {
		return nil, nil, err
	}
	canonicalCooperative, contractRequests, err := profileGeneratedMethodContract(
		context,
		method,
	)
	if err != nil {
		return nil, nil, err
	}
	providerAsync := certificate.Effect().MaySuspend()
	if providerAsync && !canonicalCooperative {
		return nil, nil, shapeError(
			bridgeName,
			"asynchronous profile method cannot satisfy a synchronous Go contract",
		)
	}
	arguments, before, argumentRequests, err :=
		providerboundary.ToProviderProfileArgumentsForBridge(
			context,
			children,
			signature.Params(),
			bridgeType,
			bridgeName,
			profile,
			target.SourceParameterReferences(context.Factory()),
		)
	if err != nil {
		return nil, nil, err
	}
	call, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				payload(context.Factory()),
				nil,
				context.Factory().Identifier(certificate.Member()),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		argumentRequests,
	)
	if err != nil {
		return nil, nil, err
	}
	call, err = cooperativecall.GeneratedInterfaceProviderCall(
		context,
		call,
		providerAsync,
	)
	if err != nil {
		return nil, nil, err
	}
	converted, err := providerboundary.FromProviderProfileResultsForBridge(
		context,
		children,
		bridgeType,
		bridgeName,
		signature.Results(),
		profile,
		call,
	)
	if err != nil {
		return nil, nil, err
	}
	memberName, err := context.Names().InterfaceMethodName(method)
	if err != nil {
		return nil, nil, err
	}
	resultType := target.Result()
	var modifiers []tsgo.ModifierLike
	if canonicalCooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return context.Factory().MethodDeclaration(
			modifiers,
			nil,
			context.Factory().Identifier(memberName),
			nil,
			nil,
			target.Parameters(),
			resultType,
			context.Factory().Block(profileMethodBody(context.Factory(), signature, converted), true),
		), api.CombineRequests(
			target.Requests(),
			contractRequests,
			converted.Requests(),
		), nil
}

func profileReverseMethod(
	context api.Context,
	providerContext api.Context,
	children api.ChildEmitter,
	bridgeName string,
	bridgeType *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
	method *types.Func,
	certificate gostdlib.ProviderInterfaceMethod,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	emission, err := prepareProfileReverseMethod(
		context,
		providerContext,
		children,
		bridgeName,
		bridgeType,
		profile,
		method,
		certificate,
		payload(context.Factory()),
	)
	if err != nil {
		return nil, nil, err
	}
	return emission.declaration(context.Factory()), emission.requests, nil
}

func prepareProfileReverseMethod(
	context api.Context,
	providerContext api.Context,
	children api.ChildEmitter,
	bridgeName string,
	bridgeType *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
	method *types.Func,
	certificate gostdlib.ProviderInterfaceMethod,
	receiver tsgo.Expression,
) (methodEmission, error) {
	signature, ok := receiverFreeSignature(method)
	if !ok {
		return methodEmission{}, shapeError(bridgeName, "profile reverse method is invalid")
	}
	target, err := callable.EmitABIAdapter(
		providerContext,
		children,
		nil,
		signature,
	)
	if err != nil {
		return methodEmission{}, err
	}
	arguments, before, argumentRequests, err :=
		providerboundary.FromProviderProfileArgumentsForBridge(
			context,
			children,
			signature.Params(),
			bridgeType,
			bridgeName,
			profile,
			target.SourceParameterReferences(context.Factory()),
		)
	if err != nil {
		return methodEmission{}, err
	}
	memberName, err := context.Names().InterfaceMethodName(method)
	if err != nil {
		return methodEmission{}, err
	}
	value := tsgo.Expression(context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(memberName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	))
	providerAsync := certificate.Effect().MaySuspend()
	if providerAsync {
		value = context.Factory().AwaitExpression(value)
	}
	call, err := api.NewExpressionEmission(before, value, argumentRequests)
	if err != nil {
		return methodEmission{}, err
	}
	converted, err := providerboundary.ToProviderProfileResultsForBridge(
		context,
		children,
		bridgeType,
		bridgeName,
		signature.Results(),
		profile,
		call,
	)
	if err != nil {
		return methodEmission{}, err
	}
	resultType := target.Result()
	var modifiers []tsgo.ModifierLike
	if providerAsync {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return methodEmission{
		name:        certificate.Member(),
		signature:   signature,
		parameters:  target.Parameters(),
		resultValue: target.Result(),
		result:      resultType,
		modifiers:   modifiers,
		body:        profileMethodBody(context.Factory(), signature, converted),
		requests: api.CombineRequests(
			target.Requests(),
			converted.Requests(),
		),
		cooperative: providerAsync,
	}, nil
}

func profileGeneratedMethodContract(
	context api.Context,
	method *types.Func,
) (bool, []api.RootRequest, error) {
	reference, err := context.Names().InterfaceMethodCallable(method)
	if err != nil {
		return false, nil, err
	}
	return cooperativecall.InterfaceMethodContract(context, reference)
}

func profileMethodBody(
	factory tsgo.Factory,
	signature *types.Signature,
	emission api.ExpressionEmission,
) []tsgo.Statement {
	body := emission.Before()
	if signature.Results().Len() == 0 {
		return append(body, factory.ExpressionStatement(emission.Value()))
	}
	return append(body, factory.ReturnStatement(emission.Value()))
}
