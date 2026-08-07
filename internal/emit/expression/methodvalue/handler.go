package methodvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/emit/methodcall"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
) (api.ExpressionEmission, error) {
	if selected == nil || selected.Kind() != types.MethodVal {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature, ok := selected.Type().(*types.Signature)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetSignatureSource := signature
	generic := genericinstance.ReceiverTypeArguments(selected.Recv()) != nil
	if generic {
		var err error
		targetSignatureSource, err =
			genericinstance.ConcreteCallableSignature(signature)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	} else if !callable.Supports(signature) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, interfaceReceiver := interfacetype.Resolve(
		selected.Recv(),
	); interfaceReceiver {
		return emitInterface(context, children, source, selected, signature)
	}
	receiver, method, err := selectionvalue.MethodReceiver(
		context,
		children,
		source,
		selected,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	owner := method.Origin()
	if generic {
		return emitGenericMethodValue(
			context,
			children,
			source,
			receiver,
			method,
			targetSignatureSource,
		)
	}
	methodSignature, ok := owner.Type().(*types.Signature)
	if !ok || methodSignature.Recv() == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	invocation, err := methodcall.Resolve(
		context,
		children,
		source,
		owner,
		methodSignature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	providerCooperative, abiCooperative, sourceRequests, err :=
		cooperativecall.SourceValueContract(
			context,
			method,
			targetSignatureSource,
		)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetSignature, err := callable.EmitABIAdapter(
		context,
		children,
		source,
		targetSignatureSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceArguments := targetSignature.ParameterReferences(context.Factory())
	selectedABI, _ := context.ResolveCallableABI(owner)
	arguments, projectionBefore, projectionRequests, err :=
		callable.ProjectArguments(
			context,
			source,
			methodSignature,
			sourceArguments,
			selectedABI,
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
	call, err := invocation.Invoke(
		context,
		children,
		context.Factory().Identifier(receiverName),
		arguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err = api.NewExpressionEmission(
		append(projectionBefore, call.Before()...),
		call.Value(),
		api.CombineRequests(projectionRequests, call.Requests()),
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
	call, err = invocation.FromProviderResults(context, children, call)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	recoveryObservation, err := context.ObserveRecoveryCallable(
		invocation.Facet(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var modifiers []tsgo.ModifierLike
	resultType := targetSignature.Result()
	if abiCooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	ordinary := context.Factory().ArrowFunction(
		modifiers,
		nil,
		targetSignature.Parameters(),
		resultType,
		context.Factory().EqualsGreaterThanToken(),
		methodValueBody(context, targetSignatureSource, call),
	)
	if !recoveryObservation.Recovery() {
		return api.NewExpressionEmission(
			before,
			ordinary,
			api.CombineRequests(
				receiver.Requests(),
				targetSignature.Requests(),
				call.Requests(),
				sourceRequests,
				recoveryObservation.Requests(),
			),
		)
	}
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredArguments, deferredProjectionBefore,
		deferredProjectionRequests, err := callable.ProjectArguments(
		context,
		source,
		methodSignature,
		sourceArguments,
		selectedABI,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredCall, err := invocation.InvokeDeferred(
		context,
		children,
		source,
		context.Factory().Identifier(receiverName),
		deferredArguments,
		context.Factory().Identifier(callable.RecoveryAuthorityName),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredCall, err = api.NewExpressionEmission(
		append(deferredProjectionBefore, deferredCall.Before()...),
		deferredCall.Value(),
		api.CombineRequests(
			deferredProjectionRequests,
			deferredCall.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredCall, err = invocation.FromProviderResults(
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
		targetSignatureSource,
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
		methodValueBody(context, targetSignatureSource, deferredCall),
	)
	target, err := api.NewExpressionEmission(
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
			call.Requests(),
			deferredCall.Requests(),
			deferredProjectionRequests,
			recoveryRequests,
			registry.Requests(),
			sourceRequests,
			recoveryObservation.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return target, nil
}

func methodValueBody(
	context api.Context,
	signature *types.Signature,
	call api.ExpressionEmission,
) tsgo.ConciseBody {
	statements := call.Before()
	if signature.Results().Len() == 0 {
		statements = append(
			statements,
			context.Factory().ExpressionStatement(call.Value()),
		)
	} else {
		statements = append(
			statements,
			context.Factory().ReturnStatement(call.Value()),
		)
	}
	return context.Factory().Block(statements, true)
}

func emitInterface(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
	signature *types.Signature,
) (api.ExpressionEmission, error) {
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleReceiverValue).
			WithExpectedType(selected.Recv()),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	method, ok := selected.Obj().(*types.Func)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	callableReference, err :=
		context.Names().InterfaceMethodCallable(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	_, cooperative, contractRequests, err :=
		cooperativecall.InterfaceMethodValueContract(
			context,
			callableReference,
			signature,
		)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetSignature, err := callable.EmitABIAdapter(
		context,
		children,
		source,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	nonNil, err := context.Names().Runtime(
		api.RuntimeInterfaceNonNil,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	member, err := context.Names().InterfaceMethodName(method)
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
						context.Factory().CallExpression(
							context.Factory().Identifier(nonNil.Name()),
							nil,
							nil,
							[]tsgo.Expression{receiver.Value()},
							tsgo.NodeFlagsNone,
						),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	arguments := targetSignature.ParameterReferences(context.Factory())
	ordinaryCall := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(receiverName),
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredCall, err := interfaceoperation.ApplyDeferred(
		context,
		children,
		source,
		selected.Recv(),
		api.DirectExpression(context.Factory().Identifier(receiverName)),
		method,
		signature,
		cooperative,
		arguments,
		context.Factory().Identifier(callable.RecoveryAuthorityName),
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
		ordinaryCall,
	)
	deferred := context.Factory().ArrowFunction(
		modifiers,
		nil,
		append(
			[]tsgo.ParameterDeclaration{recovery},
			targetSignature.Parameters()...,
		),
		resultType,
		context.Factory().EqualsGreaterThanToken(),
		methodValueBody(context, signature, deferredCall),
	)
	registry, err := deferredregistry.Reference(context, source, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
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
			nonNil.Requests(),
			contractRequests,
			deferredCall.Requests(),
			recoveryRequests,
			registry.Requests(),
		),
	)
}
