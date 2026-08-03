package concretization

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
	modifiers []tsgo.ModifierLike,
	deferred bool,
) ([]tsgo.Statement, []api.RootRequest, error) {
	concretization, ok := artifact.GenericConcretization()
	if !ok || artifact.TargetName() == "" {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic concretization artifact is invalid",
		}
	}
	owner := concretization.Owner()
	declaration, ok := owner.Type().(*types.Signature)
	if !ok {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic concretization owner has no signature",
		}
	}
	wrapperSignature, err := receiverFreeSignature(
		concretization.Signature(),
	)
	if err != nil {
		return nil, nil, err
	}
	targetSignature, err := callable.EmitAdapter(
		context,
		children,
		nil,
		wrapperSignature,
	)
	if err != nil {
		return nil, nil, err
	}
	arguments, err := api.NewTypeArgumentList(concretization.Arguments())
	if err != nil {
		return nil, nil, err
	}
	operationSet, resolved, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return nil, nil, err
	}
	if !resolved || arguments.Len() != len(operationSet.Parameters()) {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic concretization operation set is unresolved",
		}
	}
	facet, err := api.NewSourceCallableFacet(owner)
	if err != nil {
		return nil, nil, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return nil, nil, err
	}
	capabilities, capabilityRequests, err :=
		genericinstance.EmitCapabilities(
			context,
			nil,
			operationSet,
			arguments,
		)
	if err != nil {
		return nil, nil, err
	}
	sourceArguments := targetSignature.ParameterReferences(context.Factory())
	var (
		call             tsgo.CallExpression
		deferredCall     tsgo.CallExpression
		callRequests     []api.RootRequest
		deferredRequests []api.RootRequest
		typeRequests     []api.RootRequest
	)
	if declaration.Recv() == nil {
		kernelNames, ok := context.Names().(api.GenericKernelNames)
		if !ok {
			return nil, nil, &api.ContextError{
				Reason: "generic kernel names are unavailable",
			}
		}
		kernel, nameErr := kernelNames.GenericKernel(owner)
		if nameErr != nil {
			return nil, nil, nameErr
		}
		typeArguments, requests, typeErr :=
			genericinstance.EmitFunctionTypeArguments(
				context,
				children,
				nil,
				owner,
				arguments,
			)
		if typeErr != nil {
			return nil, nil, typeErr
		}
		typeRequests = requests
		mechanics, joinErr := genericabi.JoinCapabilities(
			owner,
			operationSet.Operations(),
			capabilities,
		)
		if joinErr != nil {
			return nil, nil, joinErr
		}
		callArguments := append(mechanics, sourceArguments...)
		call = context.Factory().CallExpression(
			kernel.Expression(context.Factory()),
			nil,
			typeArguments,
			callArguments,
			tsgo.NodeFlagsNone,
		)
		callRequests = kernel.Requests()
		if deferred {
			deferredKernel, deferredErr :=
				kernelNames.DeferredGenericKernel(owner)
			if deferredErr != nil {
				return nil, nil, deferredErr
			}
			deferredArguments, deferredErr :=
				deferredKernel.CallArguments(
					context.Factory().Identifier(
						callable.RecoveryAuthorityName,
					),
					callArguments,
				)
			if deferredErr != nil {
				return nil, nil, deferredErr
			}
			deferredReference := deferredKernel.Reference()
			deferredCall = context.Factory().CallExpression(
				deferredReference.Expression(context.Factory()),
				nil,
				typeArguments,
				deferredArguments,
				tsgo.NodeFlagsNone,
			)
			deferredRequests = deferredReference.Requests()
		}
	} else {
		if len(sourceArguments) == 0 {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic method concretization has no receiver",
			}
		}
		typeArguments, requests, typeErr :=
			genericinstance.EmitTypeArguments(
				context,
				children,
				nil,
				owner,
				arguments,
			)
		if typeErr != nil {
			return nil, nil, typeErr
		}
		if api.ValueReceiverTypeName(owner) != nil {
			typeArguments = nil
		}
		typeRequests = requests
		sourceBindings, bindErr := genericabi.SourceParameters(
			owner,
			sourceArguments[1:],
		)
		if bindErr != nil {
			return nil, nil, bindErr
		}
		mechanics, joinErr := genericabi.JoinClassMethod(
			owner,
			operationSet.Operations(),
			genericabi.Combine(capabilities, sourceBindings),
		)
		if joinErr != nil {
			return nil, nil, joinErr
		}
		call, callRequests, err = callable.SelectedMethodCall(
			context,
			owner,
			api.GenericKernelSuffix,
			sourceArguments[0],
			typeArguments,
			mechanics,
		)
		if err != nil {
			return nil, nil, err
		}
		if deferred {
			deferredCall, deferredRequests, err =
				callable.SelectedDeferredMethodCall(
					context,
					owner,
					api.GenericKernelSuffix,
					sourceArguments[0],
					typeArguments,
					context.Factory().Identifier(
						callable.RecoveryAuthorityName,
					),
					mechanics,
				)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	resultType := targetSignature.Result()
	if observation.Cooperative() {
		modifiers = append(modifiers, context.Factory().AsyncKeyword())
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	statement := context.Factory().FunctionDeclaration(
		modifiers,
		nil,
		context.Factory().Identifier(artifact.TargetName()),
		nil,
		targetSignature.Parameters(),
		resultType,
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ReturnStatement(call),
			},
			true,
		),
	)
	statements := []tsgo.Statement{statement}
	var recoveryRequests []api.RootRequest
	if deferred {
		recovery, requests, recoveryErr :=
			callable.RecoveryAuthorityParameter(context)
		if recoveryErr != nil {
			return nil, nil, recoveryErr
		}
		recoveryRequests = requests
		statements = append(
			statements,
			context.Factory().FunctionDeclaration(
				modifiers,
				nil,
				context.Factory().Identifier(
					artifact.TargetName()+api.DeferredEntrySuffix,
				),
				nil,
				append(
					[]tsgo.ParameterDeclaration{recovery},
					targetSignature.Parameters()...,
				),
				resultType,
				context.Factory().Block(
					[]tsgo.Statement{
						context.Factory().ReturnStatement(deferredCall),
					},
					true,
				),
			),
		)
	}
	return statements, api.CombineRequests(
		targetSignature.Requests(),
		capabilityRequests,
		typeRequests,
		callRequests,
		deferredRequests,
		recoveryRequests,
		observation.Requests(),
	), nil
}

func receiverFreeSignature(
	signature *types.Signature,
) (*types.Signature, error) {
	if signature == nil {
		return nil, &api.InvariantError{
			Role:   api.RoleCallableParameter,
			Reason: "generic concretization signature is nil",
		}
	}
	if signature.Recv() == nil {
		return signature, nil
	}
	parameters := make([]*types.Var, 0, signature.Params().Len()+1)
	parameters = append(parameters, types.NewVar(
		token.NoPos,
		signature.Recv().Pkg(),
		"receiver",
		signature.Recv().Type(),
	))
	for index := range signature.Params().Len() {
		parameters = append(parameters, signature.Params().At(index))
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(parameters...),
		signature.Results(),
		signature.Variadic(),
	), nil
}
