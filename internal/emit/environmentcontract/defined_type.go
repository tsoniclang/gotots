package environmentcontract

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	typefacet "github.com/tsoniclang/gotots/internal/emit/declaration/typefacet"
	definedmodel "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func definedDeclaration(
	context api.Context,
	children api.ChildEmitter,
	typeName *types.TypeName,
	underlying types.Type,
	requirements []api.DeclarationRequirement,
	representationFacets []api.TypeRepresentationFacet,
) (api.DeclarationEmission, error) {
	generic, err := enterGeneric(context, typeName, requirements)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context = generic.context
	target, err := children.RepresentedType(
		context.WithRole(api.RoleDefinedUnderlyingType),
		nil,
		underlying,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	name, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	typeParameters := generic.parameters
	valueType := target.Value()
	members := []tsgo.ClassElement{
		context.Factory().PropertyDeclaration(
			[]tsgo.ModifierLike{
				context.Factory().PrivateKeyword(),
				context.Factory().ReadonlyKeyword(),
			},
			context.Factory().Identifier(definedmodel.BrandMember),
			nil,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			nil,
		),
		context.Factory().PropertyDeclaration(
			[]tsgo.ModifierLike{
				context.Factory().PublicKeyword(),
				context.Factory().ReadonlyKeyword(),
			},
			context.Factory().Identifier(definedmodel.ValueMember),
			nil,
			valueType,
			nil,
		),
		context.Factory().ConstructorDeclaration(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				parameter(context, definedmodel.ValueMember, valueType),
			},
			nil,
			nil,
		),
	}
	markers := typefacet.Emission{}
	var markerRequests []api.RootRequest
	if len(representationFacets) != 0 {
		storage, storageErr := context.Values().StorageType(
			context.WithRole(api.RoleStorageType),
			nil,
			underlying,
		)
		if storageErr != nil {
			return api.DeclarationEmission{}, storageErr
		}
		markers, err = typefacet.Build(
			context,
			typeName.Type(),
			storage.Value(),
			representationFacets,
			true,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		members = append(members, markers.Members()...)
		markerRequests = api.CombineRequests(
			storage.Requests(),
			markers.Requests(),
		)
	}
	return api.DirectDeclaration(
		context.Factory().ClassDeclaration(
			exportDeclare(context),
			context.Factory().Identifier(name),
			typeParameters,
			markers.Heritage(),
			members,
		),
		api.CombineRequests(target.Requests(), markerRequests)...,
	), nil
}
