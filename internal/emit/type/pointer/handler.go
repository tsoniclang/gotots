package pointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
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
	_, element, ok := Resolve(sourceType)
	if !ok {
		return api.TypeEmission{},
			api.Unsupported(context, api.CategoryType, source)
	}
	targetElement, err := children.RepresentedType(
		context,
		source,
		element,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return pointermarker.Type(context, targetElement, true)
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
	elementType, err := children.RepresentedType(context, source, element)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return pointermarker.Type(context, elementType, false)
}

func Observe(
	context api.Context,
	sourceType types.Type,
	demand api.PointerRepresentationDemand,
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
		demand,
	)
}

func ObserveSource(
	context api.Context,
	owner types.Object,
	sourceType types.Type,
	demand api.PointerRepresentationDemand,
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
		demand,
	)
}
