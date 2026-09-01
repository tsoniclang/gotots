package concretization

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
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
	providerParameters, providerKernel, err :=
		context.GenericCallableParameters(owner)
	if err != nil {
		return nil, nil, err
	}
	var targetSignature callable.SignatureEmission
	if providerKernel {
		targetSignature, err = callable.EmitAdapterWithProviderCallableParameters(
			context,
			children,
			nil,
			wrapperSignature,
			providerParameters,
		)
	} else {
		targetSignature, err = callable.EmitAdapter(
			context,
			children,
			nil,
			wrapperSignature,
		)
	}
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
	capabilities, capabilityRequests, err :=
		genericinstance.EmitCapabilities(
			context,
			children,
			nil,
			operationSet,
			arguments,
		)
	if err != nil {
		return nil, nil, err
	}
	sourceArguments := targetSignature.ParameterReferences(context.Factory())
	var (
		call                   api.ExpressionEmission
		deferredCall           api.ExpressionEmission
		typeRequests           []api.RootRequest
		providerKernelBoundary bool
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
		providerKernelBoundary = kernel.ProviderBoundary()
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
		kernelArguments := sourceArguments
		var kernelBefore []tsgo.Statement
		var kernelRequests []api.RootRequest
		if providerKernelBoundary {
			kernelArguments, kernelBefore, kernelRequests, err =
				providerboundary.ToProviderGenericArgumentsWithCallableParameters(
					context,
					children,
					declaration.Params(),
					wrapperSignature.Params(),
					sourceArguments,
					providerParameters,
				)
			if err != nil {
				return nil, nil, err
			}
		}
		callArguments := append(mechanics, kernelArguments...)
		call, err = api.NewExpressionEmission(
			kernelBefore,
			context.Factory().CallExpression(
				kernel.Expression(context.Factory()),
				nil,
				typeArguments,
				callArguments,
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(kernel.Requests(), kernelRequests),
		)
		if err != nil {
			return nil, nil, err
		}
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
			if deferredReference.ProviderBoundary() != providerKernelBoundary {
				return nil, nil, &api.InvariantError{
					Role:   context.Role(),
					Reason: "generic kernel variants disagree on provider ownership",
				}
			}
			deferredCall, err = api.NewExpressionEmission(
				kernelBefore,
				context.Factory().CallExpression(
					deferredReference.Expression(context.Factory()),
					nil,
					typeArguments,
					deferredArguments,
					tsgo.NodeFlagsNone,
				),
				api.CombineRequests(
					deferredReference.Requests(),
					kernelRequests,
				),
			)
			if err != nil {
				return nil, nil, err
			}
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
		selectedCall, callRequests, callErr := callable.SelectedMethodCall(
			context,
			owner,
			api.GenericKernelSuffix,
			sourceArguments[0],
			typeArguments,
			mechanics,
		)
		if callErr != nil {
			return nil, nil, callErr
		}
		call = api.DirectExpression(selectedCall, callRequests...)
		if deferred {
			selectedDeferred, deferredRequests, deferredErr :=
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
			if deferredErr != nil {
				return nil, nil, deferredErr
			}
			deferredCall = api.DirectExpression(
				selectedDeferred,
				deferredRequests...,
			)
		}
	}
	if providerKernelBoundary {
		call, err = providerboundary.FromProviderGenericResults(
			context,
			children,
			declaration.Results(),
			wrapperSignature.Results(),
			call,
		)
		if err != nil {
			return nil, nil, err
		}
		if deferred {
			deferredCall, err = providerboundary.FromProviderGenericResults(
				context,
				children,
				declaration.Results(),
				wrapperSignature.Results(),
				deferredCall,
			)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	statement := context.Factory().FunctionDeclaration(
		modifiers,
		nil,
		context.Factory().Identifier(artifact.TargetName()),
		nil,
		targetSignature.Parameters(),
		targetSignature.Result(),
		context.Factory().Block(
			returnStatements(context.Factory(), call),
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
				targetSignature.Result(),
				context.Factory().Block(
					returnStatements(context.Factory(), deferredCall),
					true,
				),
			),
		)
	}
	return statements, api.CombineRequests(
		targetSignature.Requests(),
		capabilityRequests,
		typeRequests,
		call.Requests(),
		deferredCall.Requests(),
		recoveryRequests,
	), nil
}

func ExactRequirement(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	if artifact == nil || len(requirements) < 1 || len(requirements) > 2 {
		return false, &api.ContextError{
			Reason: "generic concretization requirements are not exact",
		}
	}
	bound, boundOK := artifact.GenericConcretization()
	base := false
	deferred := false
	for _, requirement := range requirements {
		selected, ok := requirement.GenericConcretization()
		generated, generatedOK := requirement.GeneratedArtifact()
		if !ok || !generatedOK || generated != artifact || !boundOK ||
			selected != bound {
			return false, &api.ContextError{
				Reason: "generic concretization received a foreign requirement",
			}
		}
		if requirement.DeferredGenericConcretization() {
			if deferred {
				return false, &api.ContextError{
					Reason: "generic concretization has duplicate deferred demand",
				}
			}
			deferred = true
			continue
		}
		if base {
			return false, &api.ContextError{
				Reason: "generic concretization has duplicate definition demand",
			}
		}
		base = true
	}
	if !base {
		return false, &api.ContextError{
			Reason: "generic concretization lacks its definition demand",
		}
	}
	return deferred, nil
}

func returnStatements(
	factory tsgo.Factory,
	emission api.ExpressionEmission,
) []tsgo.Statement {
	return append(emission.Before(), factory.ReturnStatement(emission.Value()))
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
