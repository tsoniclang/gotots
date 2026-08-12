package methodexpression

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	"github.com/tsoniclang/gotots/internal/emit/methodcall"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitGenericMethodExpression(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	receiver api.ExpressionEmission,
	method *types.Func,
	valueSignature *types.Signature,
	targetSignature callable.SignatureEmission,
) (api.ExpressionEmission, error) {
	methodSignature, ok := context.TypesInfo().
		TypeOfObject(method).(*types.Signature)
	if !ok || methodSignature.Recv() == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	selection, err := methodcall.Resolve(
		context,
		children,
		source,
		method,
		methodSignature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	providerCooperative, _, contractRequests, err :=
		cooperativecall.GenericValueContract(
			context,
			selection.Facet(),
			valueSignature,
		)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parameters := targetSignature.ParameterReferences(context.Factory())
	if len(parameters) == 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic method expression has no receiver parameter",
		}
	}
	call, err := selection.Invoke(
		context,
		children,
		receiver.Value(),
		parameters[1:],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err = api.NewExpressionEmission(
		append(receiver.Before(), call.Before()...),
		call.Value(),
		api.CombineRequests(receiver.Requests(), call.Requests()),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err = cooperativecall.SourceInterfaceProviderCall(
		context,
		source,
		call,
		providerCooperative,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err = selection.FromProviderResults(context, children, call)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	recoveryObservation, err := context.ObserveRecoveryCallable(
		selection.Facet(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var modifiers []tsgo.ModifierLike
	resultType := targetSignature.Result()
	if providerCooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	ordinary := context.Factory().ArrowFunction(
		modifiers,
		nil,
		targetSignature.Parameters(),
		resultType,
		context.Factory().EqualsGreaterThanToken(),
		methodExpressionBody(context, valueSignature, call),
	)
	if !recoveryObservation.Recovery() {
		return api.DirectExpression(
			ordinary,
			api.CombineRequests(
				targetSignature.Requests(),
				selection.Requests(),
				call.Requests(),
				contractRequests,
				recoveryObservation.Requests(),
			)...,
		), nil
	}
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredCall, err := selection.InvokeDeferred(
		context,
		children,
		source,
		receiver.Value(),
		parameters[1:],
		context.Factory().Identifier(callable.RecoveryAuthorityName),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredCall, err = api.NewExpressionEmission(
		append(receiver.Before(), deferredCall.Before()...),
		deferredCall.Value(),
		api.CombineRequests(receiver.Requests(), deferredCall.Requests()),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredCall, err = selection.FromProviderResults(
		context,
		children,
		deferredCall,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	registry, err := deferredregistry.Reference(
		context,
		source,
		valueSignature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferred := context.Factory().ArrowFunction(
		modifiers,
		nil,
		append(
			[]tsgo.ParameterDeclaration{recovery},
			targetSignature.Parameters()...,
		),
		resultType,
		context.Factory().EqualsGreaterThanToken(),
		methodExpressionBody(context, valueSignature, deferredCall),
	)
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				registry.Expression(context.Factory()),
				nil,
				context.Factory().Identifier(
					api.DeferredRegistryRegisterName,
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{ordinary, deferred},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			targetSignature.Requests(),
			selection.Requests(),
			call.Requests(),
			deferredCall.Requests(),
			recoveryRequests,
			registry.Requests(),
			contractRequests,
			recoveryObservation.Requests(),
		)...,
	), nil
}
