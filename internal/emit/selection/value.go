package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func FieldValue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
) (api.ExpressionEmission, error) {
	resolved, ok := fieldPath(selected)
	if !ok || !Valid(context, source, selected, types.FieldVal) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	root, err := children.Expression(
		context.
			WithRole(api.RoleFieldReceiver).
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
		if _, _, _, pointer := pointerType(currentType); pointer {
			current, currentType, err = dereferenceValue(
				context,
				children,
				source,
				currentType,
				current,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		}
		if !fieldInType(currentType, field) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		name, err := context.Names().Member(field)
		if err != nil {
			return api.ExpressionEmission{}, err
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
			return api.ExpressionEmission{}, err
		}
		currentType = field.Type()
	}
	if !types.Identical(currentType, resolved.effective) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return current, nil
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
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleFieldReceiver),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	storage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
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
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	logical, err := context.Values().FromStorage(
		context.WithRole(api.RoleFieldReceiver),
		source,
		element,
		stored,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	_ = raw
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
