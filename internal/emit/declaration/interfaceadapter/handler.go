package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
	"github.com/tsoniclang/gotots/internal/emit/methodcall"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
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
	contracts []Contract,
	completeMethodSet bool,
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
		api.ImportPhaseValue,
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
		completeMethodSet,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(demanded) == 0 {
		adapterFactory, factoryErr := context.Names().Runtime(
			api.RuntimeInterfaceAdapterFactory,
			api.ImportPhaseValue,
		)
		if factoryErr != nil {
			return nil, nil, factoryErr
		}
		statements, operationRequests, buildErr := buildZeroMethodAdapter(
			context,
			name,
			runtimeValue.Name(),
			adapterFactory.Name(),
			dynamicType.Name(),
			payload.Value(),
			sourceType,
			modifiers,
		)
		if buildErr != nil {
			return nil, nil, buildErr
		}
		return statements, api.CombineRequests(
			payload.Requests(),
			runtimeValue.Requests(),
			adapterFactory.Requests(),
			dynamicType.Requests(),
			operationRequests,
		), nil
	}
	methods := make(
		[]tsgo.ClassElement,
		0,
		len(demanded)+6,
	)
	var support []tsgo.Statement
	tokens := make([]tsgo.Expression, 0, len(demanded))
	tokenNames := make(map[string]struct{})
	implements, implementsRequests, err := contractHeritage(
		context,
		children,
		contracts,
	)
	if err != nil {
		return nil, nil, err
	}
	requests := api.CombineRequests(
		payload.Requests(),
		runtimeValue.Requests(),
		dynamicType.Requests(),
		implementsRequests,
	)
	for _, selected := range demanded {
		for _, contract := range selected.contracts {
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
		target, methodSupport, methodRequests, err := emitMethod(
			context,
			children,
			name,
			dynamicType.Name(),
			sourceType,
			selected.selection,
			selected.method,
			selected.contracts,
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
		methods = append(methods, target...)
		support = append(support, methodSupport...)
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
	format, formatRequests, err := formatMethod(context, sourceType)
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
		formatStringProperty(context.Factory(), sourceType),
		format,
	}
	classMembers = append(classMembers, methods...)
	heritage := []tsgo.HeritageClause{
		context.Factory().HeritageClause(
			tsgo.HeritageClauseTokenKindExtendsKeyword,
			[]tsgo.ExpressionWithTypeArguments{
				context.Factory().ExpressionWithTypeArguments(
					context.Factory().Identifier(
						runtimeValue.Name(),
					),
					nil,
				),
			},
		),
	}
	if implements != nil {
		heritage = append(heritage, implements)
	}
	statements := []tsgo.Statement{
		methodSetDeclaration(context.Factory(), name, tokens),
		typescriptclass.Declaration(context.Factory(),
			modifiers,
			context.Factory().Identifier(name),
			nil,
			heritage,
			classMembers,
		),
	}
	statements = append(statements, support...)
	return statements, api.CombineRequests(
		requests,
		equalRequests,
		hashRequests,
		formatRequests,
	), nil
}

func emitMethod(
	context api.Context,
	children api.ChildEmitter,
	adapterName string,
	dynamicTypeName string,
	sourceType types.Type,
	selected *types.Selection,
	method *types.Func,
	contracts []*types.Func,
) ([]tsgo.ClassElement, []tsgo.Statement, []api.RootRequest, error) {
	sourceSignature, ok := method.Type().(*types.Signature)
	if !ok || sourceSignature.Recv() == nil || len(contracts) == 0 {
		return nil, nil, nil, &api.GeneratedArtifactShapeError{
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
		return nil, nil, nil, methodStageError(MethodStageABI, err)
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
		return nil, nil, nil, methodStageError(MethodStageReceiver, err)
	}
	if resolvedMethod != method {
		return nil, nil, nil, &api.GeneratedArtifactShapeError{
			Reason: "adapter method receiver selection is inconsistent",
		}
	}
	interfaceDispatch := false
	if _, selected := interfacetype.Resolve(dispatchType); selected {
		interfaceDispatch = true
	}
	var invocation methodcall.Selection
	if !interfaceDispatch {
		invocation, err = methodcall.Resolve(
			context,
			children,
			nil,
			method,
			sourceSignature,
		)
		if err != nil {
			return nil, nil, nil, methodStageError(MethodStageContract, err)
		}
		if !types.Identical(invocation.Signature(), signature) {
			return nil, nil, nil, &api.GeneratedArtifactShapeError{
				Reason: "adapter method invocation signature is inconsistent",
			}
		}
	}
	sourceArguments := target.ParameterReferences(context.Factory())
	var call api.ExpressionEmission
	if interfaceDispatch {
		call, err = interfaceoperation.Apply(
			context,
			children,
			nil,
			dispatchType,
			receiver,
			method,
			sourceArguments,
			nil,
			nil,
		)
		if err != nil {
			return nil, nil, nil, methodStageError(MethodStageInvocation, err)
		}
	} else {
		call, err = invocation.Invoke(
			context,
			children,
			receiver.Value(),
			sourceArguments,
		)
		if err != nil {
			return nil, nil, nil, methodStageError(
				MethodStageInvocation,
				err,
			)
		}
		call, err = api.NewExpressionEmission(
			append(receiver.Before(), call.Before()...),
			call.Value(),
			api.CombineRequests(receiver.Requests(), call.Requests()),
		)
		if err != nil {
			return nil, nil, nil, methodStageError(MethodStageInvocation, err)
		}
	}
	if !interfaceDispatch {
		call, err = invocation.FromProviderResults(context, children, call)
		if err != nil {
			return nil, nil, nil, methodStageError(MethodStageInvocation, err)
		}
	}
	recoveryRequired := interfaceDispatch
	var recoveryObservationRequests []api.RootRequest
	if !interfaceDispatch {
		observation, observationErr := context.ObserveRecoveryCallable(
			invocation.Facet(),
		)
		if observationErr != nil {
			return nil, nil, nil, methodStageError(
				MethodStageContract,
				observationErr,
			)
		}
		recoveryRequired = observation.Recovery()
		recoveryObservationRequests = observation.Requests()
	}
	memberName, err := context.Names().InterfaceMethodName(method)
	if err != nil {
		return nil, nil, nil, err
	}
	resultType := target.Result()
	ordinaryMember := context.Factory().MethodDeclaration(
		nil,
		nil,
		context.Factory().Identifier(memberName),
		nil,
		nil,
		target.Parameters(),
		resultType,
		context.Factory().Block(
			adapterCallBody(context, signature, call),
			true,
		),
	)
	if !recoveryRequired {
		return []tsgo.ClassElement{ordinaryMember}, nil, api.CombineRequests(
			target.Requests(),
			call.Requests(),
			recoveryObservationRequests,
		), nil
	}
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return nil, nil, nil, methodStageError(MethodStageABI, err)
	}
	var deferredCall api.ExpressionEmission
	if interfaceDispatch {
		deferredCall, err = interfaceoperation.ApplyDeferred(
			context,
			children,
			nil,
			dispatchType,
			receiver,
			method,
			signature,
			sourceArguments,
			context.Factory().Identifier(callable.RecoveryAuthorityName),
		)
	} else {
		deferredCall, err = invocation.InvokeDeferred(
			context,
			children,
			nil,
			receiver.Value(),
			sourceArguments,
			context.Factory().Identifier(
				callable.RecoveryAuthorityName,
			),
		)
		if err != nil {
			return nil, nil, nil, methodStageError(
				MethodStageInvocation,
				err,
			)
		}
		deferredCall, err = api.NewExpressionEmission(
			append(receiver.Before(), deferredCall.Before()...),
			deferredCall.Value(),
			api.CombineRequests(receiver.Requests(), deferredCall.Requests()),
		)
	}
	if err != nil {
		return nil, nil, nil, methodStageError(MethodStageInvocation, err)
	}
	if !interfaceDispatch {
		deferredCall, err = invocation.FromProviderResults(
			context,
			children,
			deferredCall,
		)
		if err != nil {
			return nil, nil, nil, methodStageError(MethodStageInvocation, err)
		}
	}
	members := []tsgo.ClassElement{
		ordinaryMember,
		context.Factory().MethodDeclaration(
			nil,
			nil,
			context.Factory().Identifier(
				memberName+api.DeferredEntrySuffix,
			),
			nil,
			nil,
			append(
				[]tsgo.ParameterDeclaration{recovery},
				target.Parameters()...,
			),
			resultType,
			context.Factory().Block(
				adapterCallBody(context, signature, deferredCall),
				true,
			),
		),
	}
	deferredSupport, deferredSupportRequests, err := methodDeferredSupport(
		context,
		adapterName,
		dynamicTypeName,
		memberName,
		signature,
		target,
		resultType,
		contracts,
	)
	if err != nil {
		return nil, nil, nil, methodStageError(MethodStageInvocation, err)
	}
	return members, deferredSupport, api.CombineRequests(
		target.Requests(),
		call.Requests(),
		deferredCall.Requests(),
		recoveryRequests,
		deferredSupportRequests,
		recoveryObservationRequests,
	), nil
}
