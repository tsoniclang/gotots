package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericdeclaration "github.com/tsoniclang/gotots/internal/emit/generic/declaration"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type callableVariant struct {
	profile *api.GenericCallableProfile
}

func emitCallableVariants(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	function *types.Func,
	signature *types.Signature,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	variants, err := selectCallableVariants(
		context,
		function,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	declarations := make([]tsgo.Statement, 0, len(variants))
	var requests []api.RootRequest
	for _, variant := range variants {
		facet, err := variant.facet(function)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		cooperative, err := cooperativeRequirement(
			context,
			facet,
			requirements,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		variantContext := context.WithCooperativeCallable(
			facet,
			cooperative,
		)
		if variant.profile != nil {
			variantContext = variantContext.
				WithGenericCallableProfile(variant.profile)
		}
		declaration, variantRequests, err := emitCallableVariant(
			variantContext,
			children,
			source,
			function,
			signature,
			requirements,
			variant,
			cooperative,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		declarations = append(declarations, declaration)
		requests = append(requests, variantRequests...)
	}
	return api.NewDeclarationEmission(
		declarations,
		api.CombineRequests(requests),
	)
}

func selectCallableVariants(
	context api.Context,
	function *types.Func,
	requirements []api.DeclarationRequirement,
) ([]callableVariant, error) {
	ordered, err := api.SelectGenericCallableProfiles(
		function,
		requirements,
	)
	if err != nil {
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: err.Error(),
		}
	}
	variants := make([]callableVariant, 1, len(ordered)+1)
	for _, profile := range ordered {
		variants = append(variants, callableVariant{profile: profile})
	}
	return variants, nil
}

func (v callableVariant) facet(
	function *types.Func,
) (api.CallableFacet, error) {
	if v.profile != nil {
		return api.NewGenericCallableProfileFacet(v.profile)
	}
	return api.NewSourceCallableFacet(function)
}

func emitCallableVariant(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	function *types.Func,
	signature *types.Signature,
	requirements []api.DeclarationRequirement,
	variant callableVariant,
	cooperative bool,
) (tsgo.Statement, []api.RootRequest, error) {
	genericParameters, err := genericdeclaration.Enter(
		context,
		children,
		source,
		function,
		requirements,
	)
	if err != nil {
		return nil, nil, err
	}
	context = genericParameters.Context()
	name, err := context.Names().Declare(function)
	if err != nil {
		return nil, nil, err
	}
	if variant.profile != nil {
		name += variant.profile.Suffix()
	}
	targetSignature, err := callable.EmitDeclaration(
		context,
		children,
		source.Type,
		signature,
		api.RoleParameterType,
		api.RoleResultType,
	)
	if err != nil {
		return nil, nil, err
	}
	parameters, parameterRequests, err := emitVariantParameters(
		context,
		children,
		source,
		function,
		signature,
		genericParameters,
		targetSignature,
	)
	if err != nil {
		return nil, nil, err
	}
	body, err := callable.EmitBody(
		context,
		children,
		source,
		source.Type,
		source.Body,
		signature,
		api.RoleFunctionBody,
	)
	if err != nil {
		return nil, nil, err
	}
	moduleExport, err := context.Names().ModuleExport(function)
	if err != nil {
		return nil, nil, err
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{
			context.Factory().ExportKeyword(),
		}
	}
	resultType := targetSignature.Result()
	if cooperative {
		modifiers = append(
			modifiers,
			context.Factory().AsyncKeyword(),
		)
		resultType = callable.PromiseResult(
			context.Factory(),
			resultType,
		)
	}
	return context.Factory().FunctionDeclaration(
			modifiers,
			nil,
			context.Factory().Identifier(name),
			genericParameters.TypeNodes(),
			parameters,
			resultType,
			body.Value(),
		),
		api.CombineRequests(
			parameterRequests,
			body.Requests(),
		),
		nil
}

func emitVariantParameters(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	function *types.Func,
	signature *types.Signature,
	genericParameters genericdeclaration.Parameters,
	targetSignature callable.SignatureEmission,
) (
	[]tsgo.ParameterDeclaration,
	[]api.RootRequest,
	error,
) {
	sourceParameters := targetSignature.Parameters()
	var capabilityParameters []tsgo.ParameterDeclaration
	var err error
	if len(api.GenericDeclarationParameters(function)) != 0 {
		capabilityParameters, err = genericabi.JoinCapabilities(
			function,
			genericParameters.Operations(),
			genericParameters.Capabilities(),
		)
		if err != nil {
			return nil, nil, err
		}
	}
	parameters := append(capabilityParameters, sourceParameters...)
	requests := api.CombineRequests(
		genericParameters.Requests(),
		targetSignature.Requests(),
	)
	if signature.Recv() != nil {
		receiver, receiverRequests, err := emitReceiver(
			context,
			children,
			source,
			signature,
		)
		if err != nil {
			return nil, nil, err
		}
		if signature.RecvTypeParams().Len() != 0 {
			receiverBinding, err := genericabi.Receiver(
				function,
				receiver,
			)
			if err != nil {
				return nil, nil, err
			}
			sourceBindings, err := genericabi.SourceParameters(
				function,
				sourceParameters,
			)
			if err != nil {
				return nil, nil, err
			}
			parameters, err = genericabi.JoinMethod(
				function,
				genericParameters.Operations(),
				genericabi.Combine(
					genericParameters.Capabilities(),
					[]genericabi.Binding[tsgo.ParameterDeclaration]{
						receiverBinding,
					},
					sourceBindings,
				),
			)
			if err != nil {
				return nil, nil, err
			}
		} else {
			parameters = append(
				[]tsgo.ParameterDeclaration{receiver},
				sourceParameters...,
			)
		}
		requests = append(receiverRequests, requests...)
	}
	if context.CallableControlFor(source).Recovery() {
		recovery, recoveryRequests, err :=
			callable.RecoveryAuthorityParameter(context)
		if err != nil {
			return nil, nil, err
		}
		parameters = append(parameters, recovery)
		requests = append(requests, recoveryRequests...)
	}
	return parameters, requests, nil
}
