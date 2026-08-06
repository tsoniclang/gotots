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

type methodEmission struct {
	name        string
	signature   *types.Signature
	parameters  []tsgo.ParameterDeclaration
	resultValue tsgo.TypeNode
	result      tsgo.TypeNode
	modifiers   []tsgo.ModifierLike
	body        []tsgo.Statement
	requests    []api.RootRequest
	cooperative bool
}

func emitMethod(
	context api.Context,
	children api.ChildEmitter,
	bridgeName string,
	bridgeType *types.Named,
	method *types.Func,
	certificate gostdlib.ProviderInterfaceMethod,
) (tsgo.ClassElement, []api.RootRequest, error) {
	emission, err := prepareMethod(
		context,
		children,
		bridgeName,
		bridgeType,
		nil,
		method,
		certificate,
		payload(context.Factory()),
	)
	if err != nil {
		return nil, nil, err
	}
	return emission.declaration(context.Factory()), emission.requests, nil
}

func prepareMethod(
	context api.Context,
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
		return methodEmission{}, shapeError(bridgeName, "bridge method signature is invalid")
	}
	target, err := callable.EmitABIAdapter(context, children, nil, signature)
	if err != nil {
		return methodEmission{}, err
	}
	callableReference, err := context.Names().InterfaceMethodCallable(method)
	if err != nil {
		return methodEmission{}, err
	}
	canonicalCooperative, contractRequests, err :=
		cooperativecall.InterfaceMethodContract(context, callableReference)
	if err != nil {
		return methodEmission{}, err
	}
	memberName, err := context.Names().InterfaceMethodName(method)
	if err != nil {
		return methodEmission{}, err
	}
	if certificate.Kind() == gostdlib.ProviderInterfaceMethodRuntimeOnly {
		return runtimeOnlyMethod(
			context,
			memberName,
			signature,
			target,
			canonicalCooperative,
			contractRequests,
		)
	}
	if certificate.Kind() != gostdlib.ProviderInterfaceMethodCallable ||
		certificate.Member() == "" ||
		!certificate.Effect().Valid() {
		return methodEmission{}, shapeError(bridgeName, "provider method certificate is invalid")
	}
	providerAsync := providerCooperative(certificate)
	if providerAsync && !canonicalCooperative {
		return methodEmission{}, shapeError(
			bridgeName,
			"asynchronous provider method cannot satisfy a synchronous Go contract",
		)
	}
	sourceParameters := target.SourceParameterReferences(context.Factory())
	if len(sourceParameters) != signature.Params().Len() {
		return methodEmission{}, shapeError(
			bridgeName,
			"provider method parameter count drifted",
		)
	}
	arguments := make([]tsgo.Expression, 0, len(sourceParameters))
	var before []tsgo.Statement
	var argumentRequests []api.RootRequest
	if len(profile) != 0 {
		arguments, before, argumentRequests, err =
			providerboundary.ToProviderProfileArgumentsForBridge(
				context,
				children,
				signature.Params(),
				bridgeType,
				bridgeName,
				profile,
				sourceParameters,
			)
		if err != nil {
			return methodEmission{}, err
		}
	} else {
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
				return methodEmission{}, convertErr
			}
			before = append(before, converted.Before()...)
			arguments = append(arguments, converted.Value())
			argumentRequests = append(
				argumentRequests,
				converted.Requests()...,
			)
		}
	}
	call := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
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
		return methodEmission{}, err
	}
	emission, err = cooperativecall.GeneratedInterfaceProviderCall(
		context,
		emission,
		providerAsync,
	)
	if err != nil {
		return methodEmission{}, err
	}
	var converted api.ExpressionEmission
	if len(profile) != 0 {
		converted, err = providerboundary.FromProviderProfileResultsForBridge(
			context,
			children,
			bridgeType,
			bridgeName,
			signature.Results(),
			profile,
			emission,
		)
	} else {
		converted, err = providerboundary.FromProviderResults(
			context,
			children,
			bridgeType,
			bridgeName,
			signature.Results(),
			emission,
		)
	}
	if err != nil {
		return methodEmission{}, err
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
	return methodEmission{
		name:        memberName,
		signature:   signature,
		parameters:  target.Parameters(),
		resultValue: target.Result(),
		result:      resultType,
		modifiers:   modifiers,
		body:        body,
		requests: api.CombineRequests(
			target.Requests(),
			contractRequests,
			converted.Requests(),
		),
		cooperative: canonicalCooperative,
	}, nil
}

func runtimeOnlyMethod(
	context api.Context,
	name string,
	signature *types.Signature,
	target callable.SignatureEmission,
	cooperative bool,
	contractRequests []api.RootRequest,
) (methodEmission, error) {
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return methodEmission{}, err
	}
	resultType := target.Result()
	var modifiers []tsgo.ModifierLike
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return methodEmission{
		name:        name,
		signature:   signature,
		parameters:  target.Parameters(),
		resultValue: target.Result(),
		result:      resultType,
		modifiers:   modifiers,
		body: []tsgo.Statement{
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
		requests: api.CombineRequests(
			target.Requests(),
			contractRequests,
			panicReference.Requests(),
		),
		cooperative: cooperative,
	}, nil
}

func (e methodEmission) declaration(factory tsgo.Factory) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		e.modifiers,
		nil,
		factory.Identifier(e.name),
		nil,
		nil,
		e.parameters,
		e.result,
		factory.Block(e.body, true),
	)
}

func (e methodEmission) declarationWithoutBody(
	factory tsgo.Factory,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(e.name),
		nil,
		nil,
		e.parameters,
		e.result,
		nil,
	)
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
