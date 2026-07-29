package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
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
	methodSet := types.NewMethodSet(sourceType)
	methods := make(
		[]tsgo.ClassElement,
		0,
		methodSet.Len()+4,
	)
	tokens := make([]tsgo.Expression, 0, methodSet.Len())
	requests := api.CombineRequests(
		payload.Requests(),
		runtimeValue.Requests(),
		dynamicType.Requests(),
	)
	for index := range methodSet.Len() {
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
			return nil, nil, err
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
		return nil, nil, err
	}
	providerCooperative, contractCooperative, contractRequests, err :=
		cooperativecall.SourceValueContract(
			context,
			method,
			signature,
		)
	if err != nil {
		return nil, nil, err
	}
	root := api.DirectExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().ThisExpression(),
			nil,
			context.Factory().Identifier(ValueMember),
			tsgo.NodeFlagsNone,
		),
	)
	receiver, resolvedMethod, err := selectionvalue.MethodSetReceiver(
		context,
		children,
		nil,
		selected,
		root,
	)
	if err != nil {
		return nil, nil, err
	}
	if resolvedMethod != method || len(receiver.Before()) != 0 {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter method receiver is not direct",
		}
	}
	reference, err := context.Names().Reference(method)
	if err != nil {
		return nil, nil, err
	}
	controlRequest, err := api.NewDirectCallableControlRequest(
		method.Origin(),
		api.CallableControlRecovery,
	)
	if err != nil {
		return nil, nil, err
	}
	arguments := append(
		[]tsgo.Expression{receiver.Value()},
		target.ParameterReferences(context.Factory())...,
	)
	call := context.Factory().CallExpression(
		context.Factory().Identifier(reference.Name()),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	var body []tsgo.Statement
	if signature.Results().Len() == 0 {
		body = []tsgo.Statement{
			context.Factory().ExpressionStatement(call),
		}
	} else {
		body = []tsgo.Statement{
			context.Factory().ReturnStatement(call),
		}
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
			receiver.Requests(),
			reference.Requests(),
			[]api.RootRequest{controlRequest},
			contractRequests,
		), nil
}
