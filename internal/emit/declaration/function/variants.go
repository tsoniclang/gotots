package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	genericdeclaration "github.com/tsoniclang/gotots/internal/emit/generic/declaration"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type callableVariantEmission struct {
	statements []tsgo.Statement
	member     tsgo.ClassElement
	requests   []api.RootRequest
}

func emitCallableVariants(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	function *types.Func,
	signature *types.Signature,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	kernel, err := genericKernelRequired(
		function,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	facet, err := api.NewSourceCallableFacet(function)
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
	target, err := emitCallableVariant(
		context.WithCooperativeCallable(facet, cooperative),
		children,
		source,
		function,
		signature,
		requirements,
		cooperative,
		kernel,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	declarations := target.statements
	var members []tsgo.ClassElement
	if target.member != nil {
		members = append([]tsgo.ClassElement{target.member}, members...)
	}
	if signature.Recv() != nil {
		if len(members) != 1 {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "receiver method lost its class-member form",
			}
		}
		return api.ClassMemberAndSupportContributionEmission(
			api.MethodReceiverTypeName(function),
			members,
			declarations,
			api.CombineRequests(target.requests),
		)
	}
	if len(members) != 0 || len(declarations) < 1 {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "free function acquired a class-member form",
		}
	}
	return api.NewDeclarationEmission(
		declarations,
		api.CombineRequests(target.requests),
	)
}

func genericKernelRequired(
	function *types.Func,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	if function == nil || len(api.GenericDeclarationParameters(function)) == 0 {
		return false, nil
	}
	return api.GenericKernelRequired(function, requirements)
}

func emitCallableVariant(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	function *types.Func,
	signature *types.Signature,
	requirements []api.DeclarationRequirement,
	cooperative bool,
	kernel bool,
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
	supportName := ""
	if signature.Recv() == nil {
		name, err = context.Names().Declare(function)
	} else {
		name, err = context.Names().InterfaceMethodName(function)
		if err == nil {
			supportName, err = context.Names().Declare(function)
		}
	}
	if err != nil {
		return callableVariantEmission{}, err
	}
	if kernel {
		name += api.GenericKernelSuffix
		supportName += api.GenericKernelSuffix
	}
	valueReceiver := api.ValueReceiverTypeName(function) != nil
	copySelected := false
	deferredContext := context
	deferredReceiverName := ""
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
		deferredReceiverName, err = context.Names().Temporary(
			api.TemporaryReceiverValue,
		)
		if err != nil {
			return callableVariantEmission{}, err
		}
		deferredContext, err = deferredContext.WithValueReceiver(
			function,
			deferredContext.Factory().Identifier(deferredReceiverName),
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
	var body api.BlockEmission
	if source.Body == nil {
		body, err = emitExternalBody(context, function)
	} else {
		body, err = callable.EmitBody(
			context,
			children,
			source,
			source.Type,
			source.Body,
			signature,
			api.RoleFunctionBody,
		)
	}
	if err != nil {
		return callableVariantEmission{}, err
	}
	var deferredBody api.BlockEmission
	deferred := source.Body != nil &&
		context.CallableControlFor(source).Recovery()
	if deferred {
		deferredBody, err = callable.EmitBody(
			deferredContext.WithRecoveryAuthority(
				callable.RecoveryAuthorityName,
			),
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
	}
	if source.Body != nil && valueReceiver && copySelected {
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
		if deferred {
			deferredBody, err = prependValueReceiverCopy(
				deferredContext,
				children,
				source,
				signature,
				deferredBody,
			)
			if err != nil {
				return callableVariantEmission{}, err
			}
		}
	}
	var modifiers []tsgo.ModifierLike
	if signature.Recv() == nil {
		moduleExport := kernel
		if !moduleExport {
			var moduleErr error
			moduleExport, moduleErr = context.Names().ModuleExport(function)
			if moduleErr != nil {
				return callableVariantEmission{}, moduleErr
			}
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
		deferredBody.Requests(),
	)
	if signature.Recv() == nil {
		statements := []tsgo.Statement{
			context.Factory().FunctionDeclaration(
				modifiers,
				nil,
				context.Factory().Identifier(name),
				genericParameters.TypeNodes(),
				parameters,
				resultType,
				body.Value(),
			),
		}
		if deferred {
			recovery, recoveryRequests, recoveryErr :=
				callable.RecoveryAuthorityParameter(context)
			if recoveryErr != nil {
				return callableVariantEmission{}, recoveryErr
			}
			deferredParameters := append(
				[]tsgo.ParameterDeclaration{recovery},
				parameters...,
			)
			deferredModifiers := append(
				[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
				modifiersWithoutExport(modifiers)...,
			)
			statements = append(
				statements,
				context.Factory().FunctionDeclaration(
					deferredModifiers,
					nil,
					context.Factory().Identifier(
						name+api.DeferredEntrySuffix,
					),
					genericParameters.TypeNodes(),
					deferredParameters,
					resultType,
					deferredBody.Value(),
				),
			)
			requests = append(requests, recoveryRequests...)
			if signature.TypeParams().Len() == 0 &&
				!api.ContainsGenericTypeParameter(signature) {
				registry, registryErr := deferredregistry.Reference(
					context,
					source,
					signature,
				)
				if registryErr != nil {
					return callableVariantEmission{}, registryErr
				}
				statements = append(
					statements,
					deferredRegistrationStatement(
						context,
						registry.Expression(context.Factory()),
						context.Factory().Identifier(name),
						context.Factory().Identifier(
							name+api.DeferredEntrySuffix,
						),
					),
				)
				requests = append(requests, registry.Requests()...)
			}
		}
		return callableVariantEmission{
			statements: statements,
			requests:   requests,
		}, nil
	}
	target := callableVariantEmission{
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
	}
	if deferred {
		recovery, recoveryRequests, recoveryErr :=
			callable.RecoveryAuthorityParameter(context)
		if recoveryErr != nil {
			return callableVariantEmission{}, recoveryErr
		}
		deferredParameters := append(
			[]tsgo.ParameterDeclaration{recovery},
			parameters...,
		)
		if valueReceiver {
			receiver, receiverRequests, receiverErr := emitReceiverNamed(
				deferredContext,
				children,
				source,
				signature,
				deferredReceiverName,
			)
			if receiverErr != nil {
				return callableVariantEmission{}, receiverErr
			}
			deferredParameters = append(
				[]tsgo.ParameterDeclaration{recovery, receiver},
				parameters...,
			)
			target.requests = append(
				target.requests,
				receiverRequests...,
			)
		}
		deferredModifiers := []tsgo.ModifierLike{
			context.Factory().ExportKeyword(),
		}
		if cooperative {
			deferredModifiers = append(
				deferredModifiers,
				context.Factory().AsyncKeyword(),
			)
		}
		target.statements = []tsgo.Statement{
			context.Factory().FunctionDeclaration(
				deferredModifiers,
				nil,
				context.Factory().Identifier(
					supportName+api.DeferredEntrySuffix,
				),
				genericParameters.DetachedTypeNodes(),
				deferredParameters,
				resultType,
				deferredBody.Value(),
			),
		}
		target.requests = append(target.requests, recoveryRequests...)
	}
	return target, nil
}
