package methodexpression

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
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
	if selected == nil || selected.Kind() != types.MethodExpr {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, interfaceReceiver := interfacetype.Resolve(
		selected.Recv(),
	); interfaceReceiver {
		return emitInterface(context, children, source, selected)
	}
	method, direct := selectionvalue.DirectMethodExpression(
		context,
		source,
		selected,
	)
	typeArguments := genericinstance.ReceiverTypeArguments(selected.Recv())
	generic := typeArguments != nil
	signature, ok := selected.Type().(*types.Signature)
	if !ok ||
		signature.Params().Len() == 0 ||
		!types.Identical(
			signature.Params().At(0).Type(),
			selected.Recv(),
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetSignatureSource := signature
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
	targetSignature, err := callable.EmitABIAdapter(
		context,
		children,
		source,
		targetSignatureSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if direct {
		methodTarget, targetErr := context.Names().MethodTarget(method)
		if targetErr != nil {
			return api.ExpressionEmission{}, targetErr
		}
		if methodTarget.Kind() == api.MethodTargetSourceFunction {
			reference, referenceErr := context.Names().Reference(method)
			if referenceErr != nil {
				return api.ExpressionEmission{}, referenceErr
			}
			return api.DirectExpression(
				reference.Expression(context.Factory()),
				api.CombineRequests(
					methodTarget.Requests(),
					reference.Requests(),
				)...,
			), nil
		}
	}
	parameters := targetSignature.ParameterReferences(context.Factory())
	receiver, method, err := selectionvalue.MethodExpressionReceiver(
		context,
		children,
		source,
		selected,
		api.DirectExpression(parameters[0]),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	owner := method.Origin()
	if generic {
		return emitGenericMethodExpression(
			context,
			children,
			source,
			receiver,
			method,
			targetSignatureSource,
			targetSignature,
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
	sourceArguments := parameters[1:]
	call, err := invocation.Invoke(
		context,
		children,
		receiver.Value(),
		sourceArguments,
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
	ordinary := context.Factory().ArrowFunction(
		nil,
		nil,
		targetSignature.Parameters(),
		targetSignature.Result(),
		context.Factory().EqualsGreaterThanToken(),
		methodExpressionBody(context, targetSignatureSource, call),
	)
	if !recoveryObservation.Recovery() {
		return api.DirectExpression(
			ordinary,
			api.CombineRequests(
				targetSignature.Requests(),
				call.Requests(),
				recoveryObservation.Requests(),
			)...,
		), nil
	}
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	deferredCall, err := invocation.InvokeDeferred(
		context,
		children,
		source,
		receiver.Value(),
		sourceArguments,
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
		nil,
		nil,
		append(
			[]tsgo.ParameterDeclaration{recovery},
			targetSignature.Parameters()...,
		),
		targetSignature.Result(),
		context.Factory().EqualsGreaterThanToken(),
		methodExpressionBody(context, targetSignatureSource, deferredCall),
	)
	target := api.DirectExpression(
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
			call.Requests(),
			deferredCall.Requests(),
			recoveryRequests,
			registry.Requests(),
			recoveryObservation.Requests(),
		)...,
	)
	return target, nil
}

func methodExpressionBody(
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
) (api.ExpressionEmission, error) {
	signature, ok := selected.Type().(*types.Signature)
	if !ok ||
		!callable.Supports(signature) ||
		signature.Params().Len() == 0 ||
		!types.Identical(
			signature.Params().At(0).Type(),
			selected.Recv(),
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := callable.EmitABIAdapter(
		context,
		children,
		source,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := target.ParameterReferences(context.Factory())
	method, ok := selected.Obj().(*types.Func)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	member, err := context.Names().InterfaceMethodName(method)
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
	receiver := context.Factory().CallExpression(
		context.Factory().Identifier(nonNil.Name()),
		nil,
		nil,
		[]tsgo.Expression{arguments[0]},
		tsgo.NodeFlagsNone,
	)
	call := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments[1:],
		tsgo.NodeFlagsNone,
	)
	methodSignature, ok := method.Type().(*types.Signature)
	if !ok || methodSignature.Recv() == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	valueSignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		methodSignature.Params(),
		methodSignature.Results(),
		methodSignature.Variadic(),
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
		api.DirectExpression(arguments[0]),
		method,
		valueSignature,
		arguments[1:],
		context.Factory().Identifier(callable.RecoveryAuthorityName),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	ordinary := context.Factory().ArrowFunction(
		nil,
		nil,
		target.Parameters(),
		target.Result(),
		context.Factory().EqualsGreaterThanToken(),
		call,
	)
	deferredStatements := deferredCall.Before()
	if signature.Results().Len() == 0 {
		deferredStatements = append(
			deferredStatements,
			context.Factory().ExpressionStatement(deferredCall.Value()),
		)
	} else {
		deferredStatements = append(
			deferredStatements,
			context.Factory().ReturnStatement(deferredCall.Value()),
		)
	}
	deferred := context.Factory().ArrowFunction(
		nil,
		nil,
		append(
			[]tsgo.ParameterDeclaration{recovery},
			target.Parameters()...,
		),
		target.Result(),
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(deferredStatements, true),
	)
	registry, err := deferredregistry.Reference(context, source, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
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
			target.Requests(),
			nonNil.Requests(),
			deferredCall.Requests(),
			recoveryRequests,
			registry.Requests(),
		)...,
	), nil
}
