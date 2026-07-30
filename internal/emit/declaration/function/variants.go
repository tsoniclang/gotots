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

type callableVariantEmission struct {
	statement tsgo.Statement
	member    tsgo.ClassElement
	requests  []api.RootRequest
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
	members := make([]tsgo.ClassElement, 0, len(variants))
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
		target, err := emitCallableVariant(
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
		if target.statement != nil {
			declarations = append(declarations, target.statement)
		}
		if target.member != nil {
			members = append(members, target.member)
		}
		requests = append(requests, target.requests...)
	}
	if signature.Recv() != nil {
		if len(declarations) != 0 || len(members) != len(variants) {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "receiver method lost its class-member form",
			}
		}
		return api.ClassMemberContributionEmission(
			api.MethodReceiverTypeName(function),
			members,
			api.CombineRequests(requests),
		)
	}
	if len(members) != 0 || len(declarations) != len(variants) {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "free function acquired a class-member form",
		}
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
) (callableVariantEmission, error) {
	var (
		genericParameters genericdeclaration.Parameters
		err               error
	)
	if signature.Recv() == nil {
		genericParameters, err = genericdeclaration.Enter(
			context,
			children,
			source,
			function,
			requirements,
		)
	} else {
		genericParameters, err = genericdeclaration.EnterClassMethod(
			context,
			children,
			source,
			function,
			requirements,
		)
	}
	if err != nil {
		return callableVariantEmission{}, err
	}
	context = genericParameters.Context()
	name := ""
	if signature.Recv() == nil {
		name, err = context.Names().Declare(function)
	} else {
		name, err = context.Names().InterfaceMethodName(function)
	}
	if err != nil {
		return callableVariantEmission{}, err
	}
	if variant.profile != nil {
		name += variant.profile.Suffix()
	}
	valueReceiver := api.ValueReceiverTypeName(function) != nil
	copySelected := false
	if valueReceiver {
		copySelected, err = valueReceiverCopySelected(
			context,
			function,
			requirements,
		)
		if err != nil {
			return callableVariantEmission{}, err
		}
		receiverName, nameErr := context.Names().Parameter(
			signature.Recv(),
			signature.Params().Len(),
		)
		if nameErr != nil {
			return callableVariantEmission{}, nameErr
		}
		context, err = context.WithValueReceiver(
			function,
			context.Factory().ThisExpression(),
			receiverName,
			copySelected,
		)
		if err != nil {
			return callableVariantEmission{}, err
		}
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
		return callableVariantEmission{}, err
	}
	parameters, parameterRequests, err := emitVariantParameters(
		context,
		children,
		source,
		function,
		signature,
		genericParameters,
		targetSignature,
		valueReceiver,
	)
	if err != nil {
		return callableVariantEmission{}, err
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
		return callableVariantEmission{}, err
	}
	if valueReceiver && copySelected {
		body, err = prependValueReceiverCopy(
			context,
			children,
			source,
			signature,
			body,
		)
		if err != nil {
			return callableVariantEmission{}, err
		}
	}
	var modifiers []tsgo.ModifierLike
	if signature.Recv() == nil {
		moduleExport, moduleErr := context.Names().ModuleExport(function)
		if moduleErr != nil {
			return callableVariantEmission{}, moduleErr
		}
		if moduleExport {
			modifiers = []tsgo.ModifierLike{
				context.Factory().ExportKeyword(),
			}
		}
	} else if !valueReceiver {
		modifiers = []tsgo.ModifierLike{
			context.Factory().StaticKeyword(),
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
	requests := api.CombineRequests(
		parameterRequests,
		body.Requests(),
	)
	if signature.Recv() == nil {
		return callableVariantEmission{
			statement: context.Factory().FunctionDeclaration(
				modifiers,
				nil,
				context.Factory().Identifier(name),
				genericParameters.TypeNodes(),
				parameters,
				resultType,
				body.Value(),
			),
			requests: requests,
		}, nil
	}
	return callableVariantEmission{
		member: context.Factory().MethodDeclaration(
			modifiers,
			nil,
			context.Factory().Identifier(name),
			nil,
			genericParameters.TypeNodes(),
			parameters,
			resultType,
			body.Value(),
		),
		requests: requests,
	}, nil
}

func emitVariantParameters(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	function *types.Func,
	signature *types.Signature,
	genericParameters genericdeclaration.Parameters,
	targetSignature callable.SignatureEmission,
	valueReceiver bool,
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
	if signature.RecvTypeParams().Len() != 0 {
		sourceBindings, bindErr := genericabi.SourceParameters(
			function,
			sourceParameters,
		)
		if bindErr != nil {
			return nil, nil, bindErr
		}
		parameters, err = genericabi.JoinClassMethod(
			function,
			genericParameters.Operations(),
			genericabi.Combine(
				genericParameters.Capabilities(),
				sourceBindings,
			),
		)
		if err != nil {
			return nil, nil, err
		}
	}
	if signature.Recv() != nil && !valueReceiver {
		receiver, receiverRequests, err := emitReceiver(
			context,
			children,
			source,
			signature,
		)
		if err != nil {
			return nil, nil, err
		}
		parameters = append(
			[]tsgo.ParameterDeclaration{receiver},
			parameters...,
		)
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

func valueReceiverCopySelected(
	context api.Context,
	function *types.Func,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	selected := false
	for _, requirement := range requirements {
		if requirement.Kind() !=
			api.DeclarationRequirementValueReceiverCopy {
			continue
		}
		method, ok := requirement.ValueReceiverCopy()
		if !ok || method != function || selected {
			return false, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received an invalid value-receiver copy requirement",
			}
		}
		selected = true
	}
	return selected, nil
}

func prependValueReceiverCopy(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	signature *types.Signature,
	body api.BlockEmission,
) (api.BlockEmission, error) {
	receiver := signature.Recv()
	if source.Recv == nil || len(source.Recv.List) != 1 {
		return api.BlockEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "value-receiver copy prologue has no receiver syntax",
		}
	}
	binding, ok := context.ValueReceiver(receiver)
	if !ok || !binding.CopySelected() {
		return api.BlockEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "value-receiver copy prologue has no selected receiver",
		}
	}
	if _, addressable := context.AddressableStorage().Name(
		context,
		receiver,
	); addressable {
		return body, nil
	}
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleReceiverValue),
		source,
		receiver.Type(),
		receiver.Type(),
		api.ValueTransferCopy,
		api.DirectExpression(binding.OriginalValue()),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	receiverType, err := children.RepresentedType(
		context.WithRole(api.RoleReceiverType),
		source.Recv.List[0].Type,
		receiver.Type(),
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	statements := append([]tsgo.Statement(nil), copied.Before()...)
	statements = append(
		statements,
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(
							binding.CopyName(),
						),
						nil,
						receiverType.Value(),
						copied.Value(),
					),
				},
				tsgo.NodeFlagsLet,
			),
		),
	)
	statements = append(statements, body.Value().Statements()...)
	return api.DirectBlock(
		context.Factory().Block(statements, true),
		api.CombineRequests(
			copied.Requests(),
			receiverType.Requests(),
			body.Requests(),
		)...,
	), nil
}
