package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	genericpointer "github.com/tsoniclang/gotots/internal/emit/generic/pointer"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/emit/value/namedstructstorage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func FieldValue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
) (api.ExpressionEmission, error) {
	resolved, ok := fieldPath(context, selected)
	if !ok || !Valid(context, source, selected, types.FieldVal) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	role := api.RoleFieldReceiver
	if context.Role() == api.RoleAssignmentTarget {
		role = api.RoleAssignmentTarget
	}
	root, err := children.Expression(
		context.
			WithRole(role).
			WithExpectedType(resolved.root),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return projectValue(context, children, source, resolved, root)
}

func projectValue(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	resolved path,
	root api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	current := root
	currentType := resolved.root
	for _, field := range resolved.fields {
		var err error
		current, currentType, err = projectFieldValue(
			context,
			children,
			source,
			currentType,
			current,
			field,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	if !types.Identical(currentType, resolved.effective) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return current, nil
}

func projectFieldValue(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	currentType types.Type,
	current api.ExpressionEmission,
	field *types.Var,
) (api.ExpressionEmission, types.Type, error) {
	var err error
	if _, _, _, pointer := pointerType(currentType); pointer {
		current, currentType, err = dereferenceValue(
			context,
			children,
			source,
			currentType,
			current,
		)
		if err != nil {
			return api.ExpressionEmission{}, nil, err
		}
	}
	if !fieldInType(currentType, field) {
		return api.ExpressionEmission{}, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	current, err = joinNominalFieldCallableABI(
		context,
		currentType,
		field,
		current,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	target, selected, err := namedstructstorage.FieldTarget(
		context.WithRole(api.RoleStructField),
		source,
		currentType,
		field,
		current,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	if selected {
		current, err = target.ReadValue(
			context.WithRole(api.RoleStructField),
			source,
		)
		if err != nil {
			return api.ExpressionEmission{}, nil, err
		}
		return current, field.Type(), nil
	}
	name, err := context.Names().Member(field)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	current, err = api.NewExpressionEmission(
		current.Before(),
		context.Factory().PropertyAccessExpression(
			current.Value(),
			nil,
			context.Factory().Identifier(name),
			tsgo.NodeFlagsNone,
		),
		current.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	return current, field.Type(), nil
}

func joinNominalFieldCallableABI(
	context api.Context,
	container types.Type,
	field *types.Var,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	requests, err := cooperative.JoinNominalFieldCallableABIs(
		context,
		container,
		field,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(requests) == 0 {
		return value, nil
	}
	return api.NewExpressionEmission(
		value.Before(),
		value.Value(),
		api.CombineRequests(value.Requests(), requests),
	)
}

func dereferenceValue(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, types.Type, error) {
	raw, element, defined, ok := pointerType(sourceType)
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
	if logical, handled, err := genericpointer.Load(
		context,
		source,
		element,
		value,
	); handled || err != nil {
		return logical, element, err
	}
	representation, err := pointertype.Observe(context, raw, false)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleFieldReceiver),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		logical, err := api.NewExpressionEmission(
			value.Before(),
			pointerruntime.Direct(
				context.Factory(),
				runtime.Name(),
				targetElement.Value(),
				value.Value(),
			),
			api.CombineRequests(
				value.Requests(),
				targetElement.Requests(),
				runtime.Requests(),
				representation.Requests(),
			),
		)
		return logical, element, err
	}
	storage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
		representation,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	stored, err := api.NewExpressionEmission(
		value.Before(),
		pointerruntime.CellValue(
			context.Factory(),
			runtime.Name(),
			targetElement.Value(),
			storage.Value(),
			value.Value(),
		),
		api.CombineRequests(
			value.Requests(),
			targetElement.Requests(),
			storage.Requests(),
			runtime.Requests(),
			representation.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	logical, err := context.ContainerStorage().FromPointerStorage(
		context.WithRole(api.RoleFieldReceiver),
		source,
		element,
		representation,
		stored,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	return logical, element, nil
}

func pointerType(
	sourceType types.Type,
) (*types.Pointer, types.Type, bool, bool) {
	if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
		return pointer, pointer.Elem(), false, true
	}
	if model, ok := definedtype.ResolvePointer(sourceType); ok {
		pointer, _ := model.Pointer()
		return pointer, pointer.Elem(), true, true
	}
	return nil, nil, false, false
}

func fieldInType(sourceType types.Type, field *types.Var) bool {
	if field == nil {
		return false
	}
	if named, ok := types.Unalias(sourceType).(*types.Named); ok {
		sourceType = named.Underlying()
	}
	structType, ok := types.Unalias(sourceType).(*types.Struct)
	if !ok {
		return false
	}
	for index := range structType.NumFields() {
		if structType.Field(index) == field {
			return true
		}
	}
	return false
}

func pointerRuntime(context api.Context) (api.NameReference, error) {
	return context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
}
