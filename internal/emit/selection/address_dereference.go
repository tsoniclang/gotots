package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
)

func rawPointer(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, types.Type, error) {
	_, element, defined, ok := pointerType(sourceType)
	if !ok {
		return api.ExpressionEmission{}, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if defined {
		model, _ := definedtype.ResolvePointer(sourceType)
		var err error
		value, err = model.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, nil, err
		}
	}
	return value, element, nil
}

func dereferencePointer(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	element types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(element),
		api.PointerRepresentationDemandNone,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleFieldReceiver),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	if representation.Representation().DirectClass() {
		emission, err := api.NewExpressionEmission(
			pointer.Before(),
			pointerruntime.Direct(
				context.Factory(),
				runtime.Name(),
				logical.Value(),
				pointer.Value(),
			),
			api.CombineRequests(
				pointer.Requests(),
				logical.Requests(),
				runtime.Requests(),
				representation.Requests(),
			),
		)
		return emission, true, err
	}
	storage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
		representation,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	emission, err := api.NewExpressionEmission(
		pointer.Before(),
		pointerruntime.Dereference(
			context.Factory(),
			runtime.Name(),
			logical.Value(),
			storage.Value(),
			pointer.Value(),
		),
		api.CombineRequests(
			pointer.Requests(),
			logical.Requests(),
			storage.Requests(),
			runtime.Requests(),
			representation.Requests(),
		),
	)
	return emission, false, err
}
