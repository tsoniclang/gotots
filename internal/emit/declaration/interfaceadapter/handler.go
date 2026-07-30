package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
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
	methodSet, selectedMethods, err := demandedMethods(
		sourceType,
		contracts,
	)
	if err != nil {
		return nil, nil, err
	}
	methods := make(
		[]tsgo.ClassElement,
		0,
		len(selectedMethods)+4,
	)
	tokens := make([]tsgo.Expression, 0, len(selectedMethods))
	requests := api.CombineRequests(
		payload.Requests(),
		runtimeValue.Requests(),
		dynamicType.Requests(),
	)
	for _, index := range selectedMethods {
		selected := methodSet.At(index)
		method, ok := selected.Obj().(*types.Func)
		if !ok {
			return nil, nil, &api.GeneratedArtifactShapeError{
				Artifact: name,
				Reason:   "method set contains a non-method object",
			}
		}
		target, methodRequests, err := emitMethod(
			context,
			children,
			sourceType,
			selected,
			method,
		)
		if err != nil {
			methodError, staged := err.(*MethodError)
			if staged && methodError.Method == nil {
				methodError.Method = method
				return nil, nil, methodError
			}
			return nil, nil, &MethodError{
				Method: method,
				Cause:  err,
			}
		}
		methods = append(methods, target)
		requests = append(requests, methodRequests...)
		token, err := context.Names().InterfaceMethodToken(method)
		if err != nil {
			return nil, nil, err
		}
		tokens = append(
			tokens,
			context.Factory().Identifier(token.Name()),
		)
		requests = append(requests, token.Requests()...)
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

func demandedMethods(
	sourceType types.Type,
	contracts []*types.Interface,
) (*types.MethodSet, []int, error) {
	methodSet := types.NewMethodSet(sourceType)
	required := make(map[*types.Func]struct{})
	for _, contract := range contracts {
		if contract == nil ||
			!contract.Complete().IsMethodSet() ||
			!types.Implements(sourceType, contract) {
			return nil, nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter contract is not implemented by its source type",
			}
		}
		for index := range contract.NumMethods() {
			method := contract.Method(index)
			selected := methodSet.Lookup(method.Pkg(), method.Name())
			if selected == nil {
				return nil, nil, &api.GeneratedArtifactShapeError{
					Reason: "adapter contract method has no concrete selection",
				}
			}
			concrete, ok := selected.Obj().(*types.Func)
			if !ok || !methodidentity.Equivalent(concrete, method) {
				return nil, nil, &api.GeneratedArtifactShapeError{
					Reason: "adapter contract method selection is not exact",
				}
			}
			required[concrete] = struct{}{}
		}
	}
	selected := make([]int, 0, len(required))
	for index := range methodSet.Len() {
		method, ok := methodSet.At(index).Obj().(*types.Func)
		if !ok {
			return nil, nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter method set contains a non-method object",
			}
		}
		if _, ok := required[method]; ok {
			selected = append(selected, index)
		}
	}
	if len(selected) != len(required) {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter contract selection lost a required method",
		}
	}
	return methodSet, selected, nil
}

func emitMethod(
	context api.Context,
	children api.ChildEmitter,
	sourceType types.Type,
	selected *types.Selection,
	method *types.Func,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	sourceSignature, ok := method.Type().(*types.Signature)
	if !ok || sourceSignature.Recv() == nil {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter method has no receiver signature",
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
	if resolvedMethod != method || len(receiver.Before()) != 0 {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter method receiver is not direct",
		}
	}
	interfaceDispatch := false
	if _, selected := interfacetype.Resolve(dispatchType); selected {
		interfaceDispatch = true
	}
	var providerCooperative bool
	var contractCooperative bool
	var contractRequests []api.RootRequest
	if interfaceDispatch {
		contractCooperative, contractRequests, err =
			cooperativecall.ValueContract(context, signature)
		providerCooperative = contractCooperative
	} else {
		providerCooperative, contractCooperative, contractRequests, err =
			cooperativecall.SourceValueContract(
				context,
				method,
				signature,
			)
	}
	if err != nil {
		return nil, nil, methodStageError(MethodStageContract, err)
	}
	parameterReferences := target.ParameterReferences(context.Factory())
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
			parameterReferences,
			nil,
			nil,
		)
		if err != nil {
			return nil, nil, methodStageError(MethodStageInvocation, err)
		}
	} else {
		controlRequest, controlErr := api.NewDirectCallableControlRequest(
			method.Origin(),
			api.CallableControlRecovery,
		)
		if controlErr != nil {
			return nil, nil, controlErr
		}
		var targetCall tsgo.CallExpression
		var targetRequests []api.RootRequest
		targetCall, targetRequests, err = callable.SelectedMethodCall(
			context,
			method,
			"",
			receiver.Value(),
			nil,
			parameterReferences,
		)
		if err != nil {
			return nil, nil, methodStageError(MethodStageInvocation, err)
		}
		call = api.DirectExpression(targetCall, targetRequests...)
		callRequests = api.CombineRequests(
			receiver.Requests(),
			[]api.RootRequest{controlRequest},
		)
	}
	body := call.Before()
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
			callRequests,
			contractRequests,
		), nil
}
