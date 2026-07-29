package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
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
	parameters := targetSignature.Parameters()
	parameters = append(genericParameters.Capabilities(), parameters...)
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
		parameters = append([]tsgo.ParameterDeclaration{receiver}, parameters...)
		parameterRequests = append(receiverRequests, parameterRequests...)
	}
	body, err := callable.EmitBody(
		context,
		children,
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
	return context.WithLocalConstantProjections(owner, projections), nil
}
