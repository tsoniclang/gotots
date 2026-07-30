package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
	"github.com/tsoniclang/gotots/internal/emit/methodcall"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/emit/type/methodidentity"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	ValueMember = interfacecontract.PayloadMember
	GuardMember = "$is"
)

func Build(
	context api.Context,
	children api.ChildEmitter,
	name string,
	sourceType types.Type,
	contracts []*types.Interface,
	modifiers []tsgo.ModifierLike,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if name == "" || sourceType == nil {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Artifact: name,
			Reason:   "interface-adapter identity is invalid",
		}
	}
	payload, err := children.RepresentedType(
		context,
		nil,
		sourceType,
	)
	if err != nil {
		return nil, nil, err
	}
	runtimeValue, err := context.Names().Runtime(
		api.RuntimeInterfaceValue,
		api.ImportPhaseType,
	)
	if err != nil {
		return nil, nil, err
	}
	dynamicType, err := context.Names().InterfaceDynamicType(sourceType)
	if err != nil {
		return nil, nil, err
	}
	demanded, err := demandedMethods(
		sourceType,
		contracts,
	)
	if err != nil {
		return nil, nil, err
	}
	methods := make(
		[]tsgo.ClassElement,
		0,
		len(demanded)+4,
	)
	tokens := make([]tsgo.Expression, 0, len(demanded))
	tokenNames := make(map[string]struct{})
	requests := api.CombineRequests(
		payload.Requests(),
		runtimeValue.Requests(),
		dynamicType.Requests(),
	)
	for _, selected := range demanded {
		targets := make(
			[]api.InterfaceMethodCallableReference,
			0,
			len(selected.contracts),
		)
		for _, contract := range selected.contracts {
			callableReference, callableErr :=
				context.Names().InterfaceMethodCallable(contract)
			if callableErr != nil {
				return nil, nil, callableErr
			}
			targets = append(targets, callableReference)
			token, tokenErr :=
				context.Names().InterfaceMethodToken(contract)
			if tokenErr != nil {
				return nil, nil, tokenErr
			}
			requests = append(requests, token.Requests()...)
			if _, duplicate := tokenNames[token.Name()]; duplicate {
				continue
			}
			tokenNames[token.Name()] = struct{}{}
			tokens = append(
				tokens,
				context.Factory().Identifier(token.Name()),
			)
		}
		target, methodRequests, err := emitMethod(
			context,
			children,
			sourceType,
			selected.selection,
			selected.method,
			targets,
		)
		if err != nil {
			methodError, staged := err.(*MethodError)
			if staged && methodError.Method == nil {
				methodError.Method = selected.method
				return nil, nil, methodError
			}
			return nil, nil, &MethodError{
				Method: selected.method,
				Cause:  err,
			}
		}
		methods = append(methods, target)
		requests = append(requests, methodRequests...)
	}
	equal, equalRequests, err := equalMethod(
		context,
		name,
		runtimeValue.Name(),
		sourceType,
	)
	if err != nil {
		return nil, nil, err
	}
	hash, hashRequests, err := hashMethod(
		context,
		dynamicType.Name(),
		sourceType,
	)
	if err != nil {
		return nil, nil, err
	}
	classMembers := []tsgo.ClassElement{
		constructor(context.Factory(), payload.Value()),
		dynamicTypeProperty(context.Factory(), dynamicType.Name()),
		guardMethod(
			context.Factory(),
			name,
			runtimeValue.Name(),
			dynamicType.Name(),
		),
		methodSetProperty(context.Factory(), name),
		implementsMethod(context.Factory()),
		equal,
		hash,
	}
	classMembers = append(classMembers, methods...)
	return []tsgo.Statement{
			methodSetDeclaration(context.Factory(), name, tokens),
			context.Factory().ClassDeclaration(
				modifiers,
				context.Factory().Identifier(name),
				nil,
				[]tsgo.HeritageClause{
					context.Factory().HeritageClause(
						tsgo.HeritageClauseTokenKindImplementsKeyword,
						[]tsgo.ExpressionWithTypeArguments{
							context.Factory().ExpressionWithTypeArguments(
								context.Factory().Identifier(
									runtimeValue.Name(),
								),
								nil,
							),
						},
					),
				},
				classMembers,
			),
		}, api.CombineRequests(
			requests,
			equalRequests,
			hashRequests,
		), nil
}

type demandedMethod struct {
	selection *types.Selection
	method    *types.Func
	contracts []*types.Func
}

func demandedMethods(
	sourceType types.Type,
	contracts []*types.Interface,
) ([]demandedMethod, error) {
	methodSet := types.NewMethodSet(sourceType)
	required := make(map[*types.Func][]*types.Func)
	for _, contract := range contracts {
		if contract == nil ||
			!contract.Complete().IsMethodSet() ||
			!types.Implements(sourceType, contract) {
			return nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter contract is not implemented by its source type",
			}
		}
		for index := range contract.NumMethods() {
			method := contract.Method(index)
			selected := methodSet.Lookup(method.Pkg(), method.Name())
			if selected == nil {
				return nil, &api.GeneratedArtifactShapeError{
					Reason: "adapter contract method has no concrete selection",
				}
			}
			concrete, ok := selected.Obj().(*types.Func)
			if !ok || !methodidentity.Equivalent(concrete, method) {
				return nil, &api.GeneratedArtifactShapeError{
					Reason: "adapter contract method selection is not exact",
				}
			}
			required[concrete] = append(required[concrete], method)
		}
	}
	demanded := make([]demandedMethod, 0, len(required))
	for index := range methodSet.Len() {
		selection := methodSet.At(index)
		method, ok := selection.Obj().(*types.Func)
		if !ok {
			return nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter method set contains a non-method object",
			}
		}
		if targetContracts := required[method]; len(targetContracts) != 0 {
			demanded = append(demanded, demandedMethod{
				selection: selection,
				method:    method,
				contracts: targetContracts,
			})
		}
	}
	if len(demanded) != len(required) {
		return nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter contract selection lost a required method",
		}
	}
	return demanded, nil
}

func emitMethod(
	context api.Context,
	children api.ChildEmitter,
	sourceType types.Type,
	selected *types.Selection,
	method *types.Func,
	targets []api.InterfaceMethodCallableReference,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	sourceSignature, ok := method.Type().(*types.Signature)
	if !ok || sourceSignature.Recv() == nil || len(targets) == 0 {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter method contract is incomplete",
		}
	}
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		sourceSignature.Params(),
		sourceSignature.Results(),
		sourceSignature.Variadic(),
	)
	target, err := callable.EmitABIAdapter(
		context,
		children,
		nil,
		signature,
	)
	if err != nil {
		return nil, nil, methodStageError(MethodStageABI, err)
	}
	root := api.DirectExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().ThisExpression(),
			nil,
			context.Factory().Identifier(ValueMember),
			tsgo.NodeFlagsNone,
		),
	)
	receiver, dispatchType, resolvedMethod, err :=
		selectionvalue.DirectMethodSetReceiver(
			context,
			children,
			nil,
			selected,
			root,
		)
	if err != nil {
		return nil, nil, methodStageError(MethodStageReceiver, err)
	}
	if resolvedMethod != method {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter method receiver selection is inconsistent",
		}
	}
	interfaceDispatch := false
	if _, selected := interfacetype.Resolve(dispatchType); selected {
		interfaceDispatch = true
	}
	var providerCooperative bool
	var contractCooperative bool
	var contractRequests []api.RootRequest
	var invocation methodcall.Selection
	if interfaceDispatch {
		provider, providerErr :=
			context.Names().InterfaceMethodCallable(method)
		if providerErr != nil {
			return nil, nil, methodStageError(
				MethodStageContract,
				providerErr,
			)
		}
		providerCooperative,
			contractCooperative,
			contractRequests,
			err = cooperativecall.InterfaceProviderMethodContracts(
			context,
			provider,
			targets,
		)
	} else {
		invocation, err = methodcall.Resolve(
			context,
			children,
			nil,
			method,
			sourceSignature,
		)
		if err != nil {
			return nil, nil, methodStageError(MethodStageContract, err)
		}
		if !types.Identical(invocation.Signature(), signature) {
			return nil, nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter method invocation signature is inconsistent",
			}
		}
		providerCooperative, contractCooperative, contractRequests, err =
			cooperativecall.ProviderInterfaceMethodContracts(
				context,
				invocation.Facet(),
				targets,
			)
	}
	if err != nil {
		return nil, nil, methodStageError(MethodStageContract, err)
	}
	var call api.ExpressionEmission
	var callRequests []api.RootRequest
	if interfaceDispatch {
		call, err = interfaceoperation.Apply(
			context,
			children,
			nil,
			dispatchType,
			receiver,
			method,
			target.ParameterReferences(context.Factory()),
			nil,
			nil,
		)
		if err != nil {
			return nil, nil, methodStageError(MethodStageInvocation, err)
		}
	} else {
		recovery, ok := target.RecoveryAuthorityReference(
			context.Factory(),
		)
		if !ok {
			return nil, nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter method lacks recovery authority",
			}
		}
		targetCall, targetRequests, callErr := invocation.Call(
			context,
			receiver.Value(),
			target.SourceParameterReferences(context.Factory()),
			recovery,
		)
		if callErr != nil {
			return nil, nil, methodStageError(
				MethodStageInvocation,
				callErr,
			)
		}
		call = api.DirectExpression(targetCall, targetRequests...)
		callRequests = receiver.Requests()
	}
	call, err = api.NewExpressionEmission(
		call.Before(),
		call.Value(),
		api.CombineRequests(
			call.Requests(),
			callRequests,
			contractRequests,
		),
	)
	if err != nil {
		return nil, nil, methodStageError(MethodStageInvocation, err)
	}
	call, err = cooperativecall.GeneratedInterfaceProviderCall(
		context,
		call,
		providerCooperative,
	)
	if err != nil {
		return nil, nil, methodStageError(MethodStageInvocation, err)
	}
	body := append(receiver.Before(), call.Before()...)
	if signature.Results().Len() == 0 {
		body = append(
			body,
			context.Factory().ExpressionStatement(call.Value()),
		)
	} else {
		body = append(
			body,
			context.Factory().ReturnStatement(call.Value()),
		)
	}
	memberName, err := context.Names().InterfaceMethodName(method)
	if err != nil {
		return nil, nil, err
	}
	var modifiers []tsgo.ModifierLike
	resultType := target.Result()
	if contractCooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	if providerCooperative && !contractCooperative {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "cooperative adapter provider has a synchronous contract",
		}
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
			call.Requests(),
		), nil
}
