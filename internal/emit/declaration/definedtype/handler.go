package definedtype

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	typefacet "github.com/tsoniclang/gotots/internal/emit/declaration/typefacet"
	genericdeclaration "github.com/tsoniclang/gotots/internal/emit/generic/declaration"
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
	var genericRequirements []api.DeclarationRequirement
	var representationFacets []api.TypeRepresentationFacet
	for _, requirement := range requirements {
		if owner, _, _, ok := requirement.GenericRepresentation(); ok {
			if owner != typeName {
				return api.DeclarationEmission{}, true, &api.InvariantError{
					Role:   context.Role(),
					Reason: "defined type received a foreign generic representation",
				}
			}
			genericRequirements = append(genericRequirements, requirement)
			continue
		}
		owner, artifact, facet, ok := requirement.TypeRepresentation()
		if !ok || owner != typeName || artifact != nil {
			return api.DeclarationEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "defined type received a foreign declaration requirement",
			}
		}
		representationFacets = append(representationFacets, facet)
	}
	representation, err := model.Representation(context)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	if representation.Kind() == api.DefinedValueRepresentationGeneratedNumeric {
		if len(genericRequirements) != 0 || model.Type().TypeParams().Len() != 0 {
			return api.DeclarationEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "native defined numeric received generic requirements",
			}
		}
		name, modifiers, identityErr := declarationIdentity(context, typeName)
		if identityErr != nil {
			return api.DeclarationEmission{}, true, identityErr
		}
		underlying, underlyingErr := children.RepresentedType(
			context.WithRole(api.RoleDefinedUnderlyingType),
			source.Type,
			model.Underlying(),
		)
		if underlyingErr != nil {
			return api.DeclarationEmission{}, true, underlyingErr
		}
		brandIdentity, brandErr := context.Names().DefinedTypeIdentity(typeName)
		if brandErr != nil {
			return api.DeclarationEmission{}, true, brandErr
		}
		brand := context.Factory().TypeLiteralNode([]tsgo.TypeElement{
			context.Factory().PropertySignatureDeclaration(
				[]tsgo.ModifierLike{context.Factory().ReadonlyKeyword()},
				context.Factory().Identifier(definedtype.BrandMember),
				context.Factory().QuestionToken(),
				context.Factory().LiteralTypeNode(
					context.Factory().StringLiteral(
						brandIdentity,
						tsgo.TokenFlagsNone,
					),
				),
				context.Factory().OmittedExpression(),
			),
		})
		return api.DirectDeclaration(
			context.Factory().TypeAliasDeclaration(
				modifiers,
				context.Factory().Identifier(name),
				nil,
				context.Factory().IntersectionTypeNode([]tsgo.TypeNode{
					underlying.Value(),
					brand,
				}),
			),
			underlying.Requests()...,
		), true, nil
	}
	parameters, err := genericdeclaration.EnterType(
		context,
		source,
		typeName,
		genericRequirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, true, err
	}
	context = parameters.Context()
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
	typeParameters := parameters.Nodes()
	valueType := underlying.Value()
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
			nil,
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
					valueType,
					nil,
				),
			},
			nil,
			context.Factory().Block(nil, true),
		),
	}
	markers := typefacet.Emission{}
	var markerRequests []api.RootRequest
	if len(representationFacets) != 0 {
		storage, storageErr := context.Values().StorageType(
			context.WithRole(api.RoleStorageType),
			source.Type,
			model.Underlying(),
		)
		if storageErr != nil {
			return api.DeclarationEmission{}, true, storageErr
		}
		markers, err = typefacet.Build(
			context,
			model.Type(),
			storage.Value(),
			representationFacets,
			false,
		)
		if err != nil {
			return api.DeclarationEmission{}, true, err
		}
		members = append(members, markers.Members()...)
		markerRequests = api.CombineRequests(
			storage.Requests(),
			markers.Requests(),
		)
	}
	return api.DirectDeclaration(
		context.Factory().ClassDeclaration(
			modifiers,
			context.Factory().Identifier(name),
			typeParameters,
			markers.Heritage(),
			members,
		),
		api.CombineRequests(
			underlying.Requests(),
			markerRequests,
		)...,
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
		source, ok := candidate.(*ast.TypeSpec)
		if !ok || context.TypesInfo().DefOf(source.Name) != typeName {
			continue
		}
		if source.Assign.IsValid() != typeName.IsAlias() {
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
