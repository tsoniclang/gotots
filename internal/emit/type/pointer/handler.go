package pointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Resolve(
	sourceType types.Type,
) (*types.Pointer, types.Type, bool) {
	if sourceType == nil {
		return nil, nil, false
	}
	pointer, ok := types.Unalias(sourceType).(*types.Pointer)
	if !ok {
		return nil, nil, false
	}
	element := pointer.Elem()
	if element == nil {
		return nil, nil, false
	}
	return pointer, element, true
}

func EmitSyntax(
	context api.Context,
	children api.ChildEmitter,
	source *ast.StarExpr,
	sourceType types.Type,
) (api.TypeEmission, error) {
	pointer, element, ok := Resolve(sourceType)
	if !ok ||
		source == nil ||
		source.X == nil ||
		!types.Identical(context.TypesInfo().TypeOf(source), pointer) ||
		!types.Identical(context.TypesInfo().TypeOf(source.X), element) {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	return EmitRepresented(context, children, source, pointer)
}

func EmitRepresented(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	cell, err := EmitNonNilRepresented(
		context,
		children,
		source,
		sourceType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().UnionTypeNode([]tsgo.TypeNode{
			cell.Value(),
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		}),
		cell.Requests()...,
	), nil
}

func EmitNonNilRepresented(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	_, element, ok := Resolve(sourceType)
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	if parameter, generic := api.GenericTypeParameter(element); generic {
		facetRequests, err :=
			context.RequireGenericParameterRepresentation(
				parameter,
				api.GenericRepresentationPointer,
			)
		if err != nil {
			return api.TypeEmission{}, err
		}
		logical, err := children.RepresentedType(context, source, element)
		if err != nil {
			return api.TypeEmission{}, err
		}
		projection, err := context.Names().Runtime(
			api.RuntimePointerType,
			api.ImportPhaseType,
		)
		if err != nil {
			return api.TypeEmission{}, err
		}
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				projection.EntityName(context.Factory()),
				[]tsgo.TypeNode{logical.Value()},
			),
			api.CombineRequests(
				facetRequests,
				logical.Requests(),
				projection.Requests(),
			)...,
		), nil
	}
	representation, err := Observe(context, sourceType, false)
	if err != nil {
		return api.TypeEmission{}, err
	}
	elementType, err := children.RepresentedType(context, source, element)
	if err != nil {
		return api.TypeEmission{}, err
	}
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		return api.DirectType(
			elementType.Value(),
			api.CombineRequests(
				elementType.Requests(),
				representation.Requests(),
			)...,
		), nil
	}
	storageType, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
		representation,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			reference.EntityName(context.Factory()),
			[]tsgo.TypeNode{elementType.Value(), storageType.Value()},
		),
		api.CombineRequests(
			elementType.Requests(),
			storageType.Requests(),
			reference.Requests(),
			representation.Requests(),
		)...,
	), nil
}

func Observe(
	context api.Context,
	sourceType types.Type,
	carrierDemand bool,
) (api.PointerRepresentationObservation, error) {
	pointer, _, ok := Resolve(sourceType)
	if !ok {
		return api.PointerRepresentationObservation{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer representation source is invalid",
		}
	}
	values := context.PointerRepresentationValues()
	if values == nil {
		return api.PointerRepresentationObservation{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer representation service is unavailable",
		}
	}
	return values.PointerRepresentation(
		context,
		pointer,
		carrierDemand,
	)
}

func ObserveSource(
	context api.Context,
	owner types.Object,
	sourceType types.Type,
	carrierDemand bool,
) (api.PointerRepresentationObservation, error) {
	pointer, _, ok := Resolve(sourceType)
	if !ok || owner == nil {
		return api.PointerRepresentationObservation{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "source pointer representation input is invalid",
		}
	}
	values := context.PointerRepresentationValues()
	if values == nil {
		return api.PointerRepresentationObservation{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer representation service is unavailable",
		}
	}
	return values.SourcePointerRepresentation(
		context,
		owner,
		pointer,
		carrierDemand,
	)
}
