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

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	if source.Doc != nil ||
		source.Type == nil ||
		source.Type.Params == nil ||
		source.Body == nil {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}

	functionObject, ok := context.TypesInfo().Defs[source.Name].(*types.Func)
	if !ok {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	functionObject = functionObject.Origin()
	signature, ok := functionObject.Type().(*types.Signature)
	if !ok ||
		(source.Recv == nil) != (signature.Recv() == nil) {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
	context, err := applyAddressableStorage(
		context,
		source,
		functionObject,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context, err = applyLocalConstantProjections(
		context,
		source,
		functionObject,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context, err = context.WithCallableControls(
		api.MustSourceArtifactOwner(functionObject),
		source,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	genericParameters, err := genericdeclaration.Enter(
		context,
		children,
		source,
		functionObject,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context = genericParameters.Context()
	name, err := context.Names().Declare(functionObject)
	if err != nil {
		return api.DeclarationEmission{}, err
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
		return api.DeclarationEmission{}, err
	}
	sourceParameters := targetSignature.Parameters()
	var capabilityParameters []tsgo.ParameterDeclaration
	if len(api.GenericDeclarationParameters(functionObject)) != 0 {
		capabilityParameters, err = genericabi.JoinCapabilities(
			functionObject,
			genericParameters.Operations(),
			genericParameters.Capabilities(),
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
	}
	parameters := append(
		capabilityParameters,
		sourceParameters...,
	)
	parameterRequests := api.CombineRequests(
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
			return api.DeclarationEmission{}, err
		}
		if signature.RecvTypeParams().Len() != 0 {
			receiverBinding, bindErr := genericabi.Receiver(
				functionObject,
				receiver,
			)
			if bindErr != nil {
				return api.DeclarationEmission{}, bindErr
			}
			sourceBindings, bindErr := genericabi.SourceParameters(
				functionObject,
				sourceParameters,
			)
			if bindErr != nil {
				return api.DeclarationEmission{}, bindErr
			}
			parameters, err = genericabi.JoinMethod(
				functionObject,
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
				return api.DeclarationEmission{}, err
			}
		} else {
			parameters = append(
				[]tsgo.ParameterDeclaration{receiver},
				sourceParameters...,
			)
		}
		parameterRequests = append(receiverRequests, parameterRequests...)
	}
	if context.CallableControlFor(source).Recovery() {
		recovery, requests, err := callable.RecoveryAuthorityParameter(context)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		parameters = append(parameters, recovery)
		parameterRequests = append(parameterRequests, requests...)
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
		return api.DeclarationEmission{}, err
	}
	moduleExport, err := context.Names().ModuleExport(functionObject)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	target := context.Factory().FunctionDeclaration(
		modifiers,
		nil,
		context.Factory().Identifier(name),
		genericParameters.TypeNodes(),
		parameters,
		targetSignature.Result(),
		body.Value(),
	)
	return api.DirectDeclaration(
		target,
		api.CombineRequests(
			parameterRequests,
			body.Requests(),
		)...,
	), nil
}

func applyLocalConstantProjections(
	context api.Context,
	source *ast.FuncDecl,
	owner *types.Func,
	requirements []api.DeclarationRequirement,
) (api.Context, error) {
	if source == nil || owner == nil {
		return api.Context{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "local-constant projection artifact identity is nil",
		}
	}
	projections := make(map[*types.Const][]types.BasicKind)
	for _, requirement := range requirements {
		if requirement.Kind() != api.DeclarationRequirementLocalConstantProjection {
			continue
		}
		requirementOwner, selected, projection, ok :=
			requirement.LocalConstantProjection()
		if !ok ||
			requirementOwner != owner ||
			selected.Pkg() != owner.Pkg() ||
			selected.Pos() < source.Pos() ||
			selected.Pos() > source.End() {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received foreign local-constant projection requirement",
			}
		}
		for _, existing := range projections[selected] {
			if existing == projection {
				return api.Context{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "function received duplicate local-constant projection requirement",
				}
			}
		}
		projections[selected] = append(projections[selected], projection)
	}
	return context.WithLocalConstantProjections(owner, projections)
}
