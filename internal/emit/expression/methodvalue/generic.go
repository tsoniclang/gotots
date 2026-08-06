package methodvalue

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

func emitGenericMethodValue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	receiver api.ExpressionEmission,
	method *types.Func,
	valueSignature *types.Signature,
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
	providerCooperative, cooperative, contractRequests, err :=
		cooperativecall.GenericValueContract(
			context,
			selection.Facet(),
			valueSignature,
		)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetSignature, err := callable.EmitABIAdapter(
		context,
		children,
		source,
		valueSignature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverName, err := context.Names().Temporary(
		api.TemporaryReceiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		receiver.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(receiverName),
						nil,
						nil,
						receiver.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	arguments := targetSignature.ParameterReferences(context.Factory())
	call, err := selection.Invoke(
		context,
		children,
		context.Factory().Identifier(receiverName),
		arguments,
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
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	ordinary := context.Factory().ArrowFunction(
		modifiers,
		nil,
		targetSignature.Parameters(),
		resultType,
		context.Factory().EqualsGreaterThanToken(),
		methodValueBody(context, valueSignature, call),
	)
	if !recoveryObservation.Recovery() {
		return api.NewExpressionEmission(
			before,
			ordinary,
			api.CombineRequests(
				receiver.Requests(),
				targetSignature.Requests(),
				selection.Requests(),
				call.Requests(),
				contractRequests,
				recoveryObservation.Requests(),
			),
		)
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
		context.Factory().Identifier(receiverName),
		arguments,
		context.Factory().Identifier(callable.RecoveryAuthorityName),
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
		methodValueBody(context, valueSignature, deferredCall),
	)
	return api.NewExpressionEmission(
		before,
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
			receiver.Requests(),
			targetSignature.Requests(),
			selection.Requests(),
			call.Requests(),
			deferredCall.Requests(),
			recoveryRequests,
			registry.Requests(),
			contractRequests,
			recoveryObservation.Requests(),
		),
	)
}
