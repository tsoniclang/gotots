package generic

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericdeclaration "github.com/tsoniclang/gotots/internal/emit/generic/declaration"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, bool, error) {
	spec, ok := sourceSpec(context, declaration, typeName)
	if !ok {
		return api.DeclarationEmission{}, false, nil
	}
	named, namedOK := types.Unalias(typeName.Type()).(*types.Named)
	if namedOK {
		if constraint, interfaceOK := named.Underlying().(*types.Interface); interfaceOK &&
			!constraint.Complete().IsMethodSet() {
			if len(requirements) != 0 ||
				spec.TypeParams != nil ||
				spec.Assign.IsValid() {
				return api.DeclarationEmission{}, true,
					api.Unsupported(
						context,
						api.CategoryDeclaration,
						declaration,
					)
			}
			return api.CoverageOnlyDeclarationEmission(), true, nil
		}
	}
	if spec.TypeParams == nil {
		return api.DeclarationEmission{}, false, nil
	}
	if !typeName.IsAlias() || !spec.Assign.IsValid() {
		return api.DeclarationEmission{}, false, nil
	}
	if len(requirements) != 0 {
		return api.DeclarationEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic alias received declaration requirements",
		}
	}
	parameters, err := genericdeclaration.EnterType(context, spec, typeName)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	context = parameters.Context()
	target, err := children.RepresentedType(
		context.WithRole(api.RoleDefinedUnderlyingType),
		spec.Type,
		context.TypesInfo().TypeOf(spec.Type),
	)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	name, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	moduleExport, err := context.Names().ModuleExport(typeName)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	return api.DirectDeclaration(
		context.Factory().TypeAliasDeclaration(
			modifiers,
			context.Factory().Identifier(name),
			parameters.Nodes(),
			target.Value(),
		),
		target.Requests()...,
	), true, nil
}

func sourceSpec(
	context api.Context,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
) (*ast.TypeSpec, bool) {
	if declaration == nil || typeName == nil {
		return nil, false
	}
	for _, candidate := range declaration.Specs {
		spec, ok := candidate.(*ast.TypeSpec)
		if ok && context.TypesInfo().Defs[spec.Name] == typeName {
			return spec, true
		}
	}
	return nil, false
}
