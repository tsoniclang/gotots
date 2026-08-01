package providerinterfacebridge

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitMethod(
	context api.Context,
	children api.ChildEmitter,
	bridgeName string,
	bridgeType *types.Named,
	method *types.Func,
	certificate gostdlib.ProviderInterfaceMethod,
) (tsgo.ClassElement, []api.RootRequest, error) {
	signature, ok := receiverFreeSignature(method)
	if !ok {
		return nil, nil, shapeError(bridgeName, "bridge method signature is invalid")
	}
	target, err := callable.EmitABIAdapter(context, children, nil, signature)
	if err != nil {
		return nil, nil, err
	}
	callableReference, err := context.Names().InterfaceMethodCallable(method)
	if err != nil {
		return nil, nil, err
	}
	canonicalCooperative, contractRequests, err :=
		cooperativecall.InterfaceMethodContract(context, callableReference)
	if err != nil {
		return nil, nil, err
	}
	memberName, err := context.Names().InterfaceMethodName(method)
	if err != nil {
		return nil, nil, err
	}
	if certificate.Kind() == gostdlib.ProviderInterfaceMethodRuntimeOnly {
		return runtimeOnlyMethod(
			context,
			memberName,
			target,
			canonicalCooperative,
			contractRequests,
		)
	}
	if certificate.Kind() != gostdlib.ProviderInterfaceMethodCallable ||
		certificate.Member() == "" ||
		!certificate.Effect().Valid() {
		return nil, nil, shapeError(bridgeName, "provider method certificate is invalid")
	}
	providerAsync := providerCooperative(certificate)
	if providerAsync && !canonicalCooperative {
		return nil, nil, shapeError(
			bridgeName,
			"asynchronous provider method cannot satisfy a synchronous Go contract",
		)
	}
	sourceParameters := target.SourceParameterReferences(context.Factory())
	if len(sourceParameters) != signature.Params().Len() {
		return nil, nil, shapeError(
			bridgeName,
			"provider method parameter count drifted",
		)
	}
	arguments := make([]tsgo.Expression, 0, len(sourceParameters))
	var before []tsgo.Statement
	var argumentRequests []api.RootRequest
	for index, parameter := range sourceParameters {
		converted, _, convertErr := providerboundary.ToProviderValue(
			context,
			children,
			bridgeType,
			bridgeName,
			signature.Params().At(index).Type(),
			api.DirectExpression(parameter),
		)
		if convertErr != nil {
			return nil, nil, convertErr
		}
		before = append(before, converted.Before()...)
		arguments = append(arguments, converted.Value())
		argumentRequests = append(
			argumentRequests,
			converted.Requests()...,
		)
	}
	call := context.Factory().CallExpression(
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
	)
	emission, err := api.NewExpressionEmission(
		before,
		call,
		api.CombineRequests(argumentRequests),
	)
	if err != nil {
		return nil, nil, err
	}
	emission, err = cooperativecall.GeneratedInterfaceProviderCall(
		context,
		emission,
		providerAsync,
	)
	if err != nil {
		return nil, nil, err
	}
	converted, err := providerboundary.FromProviderResults(
		context,
		children,
		bridgeType,
		bridgeName,
		signature.Results(),
		emission,
	)
	if err != nil {
		return nil, nil, err
	}
	body := converted.Before()
	if signature.Results().Len() == 0 {
		body = append(body, context.Factory().ExpressionStatement(converted.Value()))
	} else {
		body = append(body, context.Factory().ReturnStatement(converted.Value()))
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
			context.Factory().Block(body, true),
		), api.CombineRequests(
			target.Requests(),
			contractRequests,
			converted.Requests(),
		), nil
}

func runtimeOnlyMethod(
	context api.Context,
	name string,
	target callable.SignatureEmission,
	cooperative bool,
	contractRequests []api.RootRequest,
) (tsgo.ClassElement, []api.RootRequest, error) {
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	resultType := target.Result()
	var modifiers []tsgo.ModifierLike
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return context.Factory().MethodDeclaration(
			modifiers,
			nil,
			context.Factory().Identifier(name),
			nil,
			nil,
			target.Parameters(),
			resultType,
			context.Factory().Block(
				[]tsgo.Statement{
					context.Factory().ReturnStatement(
						panicruntime.Call(
							context.Factory(),
							panicReference.Name(),
							context.Factory().StringLiteral(
								"inaccessible provider-interface method",
								tsgo.TokenFlagsNone,
							),
						),
					),
				},
				true,
			),
		), api.CombineRequests(
			target.Requests(),
			contractRequests,
			panicReference.Requests(),
		), nil
}

func receiverFreeSignature(method *types.Func) (*types.Signature, bool) {
	if method == nil {
		return nil, false
	}
	source, ok := method.Type().(*types.Signature)
	if !ok || source.TypeParams().Len() != 0 || source.RecvTypeParams().Len() != 0 {
		return nil, false
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		source.Params(),
		source.Results(),
		source.Variadic(),
	), true
}
