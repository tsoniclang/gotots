package definedtype

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
	requirements []api.DeclarationRequirement,
) (api.DeclarationEmission, bool, error) {
	source, ok := sourceSpec(context, declaration, typeName)
	if !ok {
		return api.DeclarationEmission{}, false, nil
	}
	var model definedtype.Model
	if !typeName.IsAlias() {
		model, ok = definedtype.Resolve(typeName.Type())
		if !ok || model.TypeName() != typeName {
			return api.DeclarationEmission{}, false, nil
		}
	}
	if typeName.IsAlias() {
		if len(requirements) != 0 {
			return api.DeclarationEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "type alias received declaration requirements",
			}
		}
		target, err := children.RepresentedType(
			context.WithRole(api.RoleDefinedUnderlyingType),
			source.Type,
			typeName.Type(),
		)
		if err != nil {
			return api.DeclarationEmission{}, true, err
		}
		name, modifiers, err := declarationIdentity(context, typeName)
		if err != nil {
			return api.DeclarationEmission{}, true, err
		}
		return api.DirectDeclaration(
			context.Factory().TypeAliasDeclaration(
				modifiers,
				context.Factory().Identifier(name),
				nil,
				target.Value(),
			),
			target.Requests()...,
		), true, nil
	}
	underlying, err := children.RepresentedType(
		context.WithRole(api.RoleDefinedUnderlyingType),
		source.Type,
		model.Underlying(),
	)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	name, modifiers, err := declarationIdentity(context, typeName)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	members := []tsgo.ClassElement{
		context.Factory().PropertyDeclaration(
			[]tsgo.ModifierLike{
				context.Factory().DeclareKeyword(),
				context.Factory().PrivateKeyword(),
				context.Factory().ReadonlyKeyword(),
			},
			context.Factory().Identifier(definedtype.BrandMember),
			nil,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			nil,
		),
		context.Factory().ConstructorDeclaration(
			constructorModifiers(context, model),
			nil,
			[]tsgo.ParameterDeclaration{
				context.Factory().ParameterDeclaration(
					[]tsgo.ModifierLike{
						context.Factory().PublicKeyword(),
						context.Factory().ReadonlyKeyword(),
					},
					nil,
					context.Factory().Identifier(definedtype.ValueMember),
					nil,
					underlying.Value(),
					nil,
				),
			},
			nil,
			context.Factory().Block(nil, true),
		),
	}
	familyMembers, familyRequests, err := emitFamilyMembers(
		context,
		children,
		source,
		model,
		name,
		underlying.Value(),
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	members = append(members, familyMembers...)
	return api.DirectDeclaration(
		context.Factory().ClassDeclaration(
			modifiers,
			context.Factory().Identifier(name),
			nil,
			nil,
			members,
		),
		api.CombineRequests(
			underlying.Requests(),
			familyRequests,
		)...,
	), true, nil
}

func constructorModifiers(
	context api.Context,
	model definedtype.Model,
) []tsgo.ModifierLike {
	if !model.NilCapable() {
		return nil
	}
	return []tsgo.ModifierLike{context.Factory().PrivateKeyword()}
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
		source, ok := candidate.(*ast.TypeSpec)
		if !ok || context.TypesInfo().Defs[source.Name] != typeName {
			continue
		}
		if source.TypeParams != nil ||
			source.Assign.IsValid() != typeName.IsAlias() {
			return nil, false
		}
		return source, true
	}
	return nil, false
}

func declarationIdentity(
	context api.Context,
	typeName *types.TypeName,
) (string, []tsgo.ModifierLike, error) {
	name, err := context.Names().Declare(typeName)
	if err != nil {
		return "", nil, err
	}
	moduleExport, err := context.Names().ModuleExport(typeName)
	if err != nil {
		return "", nil, err
	}
	if !moduleExport {
		return name, nil, nil
	}
	return name, []tsgo.ModifierLike{context.Factory().ExportKeyword()}, nil
}
