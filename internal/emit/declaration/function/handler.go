package function

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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

	functionObject, ok := context.TypesInfo().DefOf(source.Name).(*types.Func)
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
	context, err := applyLocalConstantProjections(
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
