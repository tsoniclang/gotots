package methodexpression

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
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
	method, _ := selectionvalue.DirectMethodExpression(
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
	var (
		memberSuffix      string
		abiCooperative    bool
		sourceRequests    []api.RootRequest
		selectionRequests []api.RootRequest
	)
	if generic {
		providerSignature, signatureErr :=
			genericinstance.ConcreteMethodExpressionProviderSignature(
				signature,
			)
		if signatureErr != nil {
			return api.ExpressionEmission{}, signatureErr
		}
		declarationSignature, ok := owner.Type().(*types.Signature)
		if !ok {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		var facet api.CallableFacet
		memberSuffix, facet, _, selectionRequests, err =
			cooperativecall.SelectGenericClassMethod(
				context,
				owner,
				declarationSignature,
				providerSignature,
			)
		if err == nil {
			_, abiCooperative, sourceRequests, err =
				cooperativecall.GenericValueContract(
					context,
					facet,
					targetSignatureSource,
				)
		}
	} else {
		_, abiCooperative, sourceRequests, err =
			cooperativecall.SourceValueContract(
				context,
				method,
				targetSignatureSource,
			)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var targetTypeArguments []tsgo.TypeNode
	var typeRequests []api.RootRequest
	var capabilities []genericabi.Binding[tsgo.Expression]
	var capabilityRequests []api.RootRequest
	var operationSet api.GenericOperationSet
	if generic {
		var resolved bool
		var resolveErr error
		operationSet, resolved, resolveErr =
			context.ResolveGenericCallable(owner)
		if resolveErr != nil {
			return api.ExpressionEmission{}, resolveErr
		}
		if !resolved ||
			typeArguments.Len() != len(operationSet.Parameters()) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		targetTypeArguments, typeRequests, err =
			genericinstance.EmitTypeArguments(
				context,
				children,
				source,
				owner,
				typeArguments,
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		capabilities, capabilityRequests, err =
			genericinstance.EmitCapabilities(
				context,
				source,
				operationSet,
				typeArguments,
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	controlRequest, err := api.NewDirectCallableControlRequest(
		owner,
		api.CallableControlRecovery,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := parameters[1:]
	if generic {
		sourceParameters := targetSignature.SourceParameterReferences(
			context.Factory(),
		)
		recoveryAuthority, recoveryOK :=
			targetSignature.RecoveryAuthorityReference(context.Factory())
		if !recoveryOK {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic method expression lacks recovery authority",
			}
		}
		sourceBindings, bindErr := genericabi.SourceParameters(
			owner,
			sourceParameters[1:],
		)
		if bindErr != nil {
			return api.ExpressionEmission{}, bindErr
		}
		arguments, err = genericabi.JoinClassMethod(
			owner,
			operationSet.Operations(),
			genericabi.Combine(
				capabilities,
				sourceBindings,
			),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		arguments = append(arguments, recoveryAuthority)
	}
	if api.ValueReceiverTypeName(owner) != nil {
		targetTypeArguments = nil
	}
	call, callRequests, err := callable.SelectedMethodCall(
		context,
		owner,
		memberSuffix,
		receiver.Value(),
		targetTypeArguments,
		arguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var body tsgo.ConciseBody = call
	if len(receiver.Before()) != 0 {
		statements := receiver.Before()
		if signature.Results().Len() == 0 {
			statements = append(
				statements,
				context.Factory().ExpressionStatement(call),
			)
		} else {
			statements = append(
				statements,
				context.Factory().ReturnStatement(call),
			)
		}
		body = context.Factory().Block(statements, true)
	}
	var modifiers []tsgo.ModifierLike
	resultType := targetSignature.Result()
	if abiCooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	target := api.DirectExpression(
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			targetSignature.Parameters(),
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			body,
		),
		api.CombineRequests(
			receiver.Requests(),
			targetSignature.Requests(),
			typeRequests,
			capabilityRequests,
			selectionRequests,
			callRequests,
			[]api.RootRequest{controlRequest},
			sourceRequests,
		)...,
	)
	return target, nil
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
	var modifiers []tsgo.ModifierLike
	resultType := target.Result()
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return api.DirectExpression(
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			target.Parameters(),
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			call,
		),
		api.CombineRequests(
			target.Requests(),
			nonNil.Requests(),
			contractRequests,
		)...,
	), nil
}
