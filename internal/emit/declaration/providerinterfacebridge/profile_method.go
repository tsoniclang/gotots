package providerinterfacebridge

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
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
	if err := providerboundary.RequireProviderEffect(
		context,
		method.FullName(),
		certificate.Effect(),
	); err != nil {
		return nil, nil, err
	}
	target, err := callable.EmitABIAdapter(context, children, nil, signature)
	if err != nil {
		return nil, nil, err
	}
	member := context.Factory().MethodSignatureDeclaration(
		nil,
		context.Factory().Identifier(certificate.Member()),
		nil,
		nil,
		target.Parameters(),
		target.Result(),
	)
	return member, target.Requests(), nil
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
	if err := providerboundary.RequireProviderEffect(
		context,
		method.FullName(),
		certificate.Effect(),
	); err != nil {
		return nil, nil, err
	}
	target, err := callable.EmitABIAdapter(context, children, nil, signature)
	if err != nil {
		return nil, nil, err
	}
	contractRequests, err := profileGeneratedMethodContract(
		context,
		method,
	)
	if err != nil {
		return nil, nil, err
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
	return context.Factory().MethodDeclaration(
			nil,
			nil,
			context.Factory().Identifier(memberName),
			nil,
			nil,
			target.Parameters(),
			target.Result(),
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
	if err := providerboundary.RequireProviderEffect(
		context,
		method.FullName(),
		certificate.Effect(),
	); err != nil {
		return methodEmission{}, err
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
	return methodEmission{
		name:        certificate.Member(),
		signature:   signature,
		parameters:  target.Parameters(),
		resultValue: target.Result(),
		result:      target.Result(),
		body:        profileMethodBody(context.Factory(), signature, converted),
		requests: api.CombineRequests(
			target.Requests(),
			converted.Requests(),
		),
	}, nil
}

func profileGeneratedMethodContract(
	context api.Context,
	method *types.Func,
) ([]api.RootRequest, error) {
	reference, err := context.Names().InterfaceMethodCallable(method)
	if err != nil {
		return nil, err
	}
	return reference.Requests(), nil
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
