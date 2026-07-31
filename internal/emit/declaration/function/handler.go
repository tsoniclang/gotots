package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	emitstorage "github.com/tsoniclang/gotots/internal/emit/storage"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.FuncDecl,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, error) {
	if source.Type == nil ||
		source.Type.Params == nil ||
		source.Name == nil {
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
	if source.Body == nil {
		return api.DeclarationEmission{},
			api.ExternalFunctionObligation(
				context,
				source,
				functionObject,
				signature,
			)
	}
	context, err := emitstorage.ApplyRequirements(
		context,
		source,
		api.MustSourceArtifactOwner(functionObject),
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
	return emitCallableVariants(
		context,
		children,
		source,
		functionObject,
		signature,
		requirements,
	)
}

func cooperativeRequirement(
	context api.Context,
	facet api.CallableFacet,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	selected := false
	for _, requirement := range requirements {
		if requirement.Kind() != api.DeclarationRequirementCooperativeCallable {
			continue
		}
		requirementFacet, ok := requirement.CooperativeCallable()
		if !ok || requirementFacet.Owner() != facet.Owner() {
			return false, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received an invalid cooperative requirement",
			}
		}
		if requirementFacet != facet {
			continue
		}
		if selected {
			return false, &api.InvariantError{
				Role:   context.Role(),
				Reason: "function received a duplicate cooperative requirement",
			}
		}
		selected = true
	}
	return selected, nil
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
