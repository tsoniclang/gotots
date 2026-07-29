package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func FieldAddress(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
	element types.Type,
) (api.ExpressionEmission, error) {
	resolved, ok := fieldPath(selected)
	if !ok ||
		!Valid(context, source, selected, types.FieldVal) ||
		!types.Identical(resolved.effective, element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return addressSource(context, children, source, resolved)
}

func FieldStoreTarget(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
) (api.StoreTargetEmission, error) {
	resolved, ok := fieldPath(selected)
	if !ok || !Valid(context, source, selected, types.FieldVal) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if field, direct := resolved.directField(); direct {
		receiver, err := children.Expression(
			context.
				WithRole(api.RoleAssignmentTarget).
				WithExpectedType(resolved.root),
			source.X,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		name, err := context.Names().Member(field)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		return api.NewPropertyStoreTargetEmission(
			context.Factory(),
			receiver,
			name,
			field.Type(),
		)
	}
	pointer, err := addressSource(context, children, source, resolved)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return canonicalPointerTarget(
		context,
		children,
		source,
		pointer,
		resolved.effective,
	)
}

func addressSource(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	resolved path,
) (api.ExpressionEmission, error) {
	if _, _, _, pointer := pointerType(resolved.root); pointer {
		root, err := children.Expression(
			context.
				WithRole(api.RoleFieldReceiver).
				WithExpectedType(resolved.root),
			source.X,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return projectAddress(
			context,
			children,
			source,
			resolved,
			root,
			false,
		)
	}
	root, err := children.Address(
		context.
			WithRole(api.RoleFieldReceiver).
			WithExpectedType(types.NewPointer(resolved.root)),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return projectAddress(
		context,
		children,
		source,
		resolved,
		root,
		true,
	)
}

func projectAddress(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	resolved path,
	root api.ExpressionEmission,
	rootIsAddress bool,
) (api.ExpressionEmission, error) {
	current := root
	currentType := resolved.root
	if !rootIsAddress {
		var err error
		current, currentType, err = rawPointer(
			context,
			source,
			currentType,
			current,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	for index, field := range resolved.fields {
		parent, err := dereferencePointer(
			context,
			children,
			source,
			currentType,
			current,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		fieldPointer, err := projectFieldPointer(
			context,
			children,
			source,
			currentType,
			parent,
			field,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if index == len(resolved.fields)-1 {
			return fieldPointer, nil
		}
		if _, _, _, pointer := pointerType(field.Type()); pointer {
			logical, _, err := dereferenceValue(
				context,
				children,
				source,
				types.NewPointer(field.Type()),
				fieldPointer,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			current, currentType, err = rawPointer(
				context,
				source,
				field.Type(),
				logical,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			continue
		}
		current = fieldPointer
		currentType = field.Type()
	}
	return current, nil
}

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
) (api.ExpressionEmission, error) {
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleFieldReceiver),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
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
		),
	)
}

func projectFieldPointer(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	parentType types.Type,
	parent api.ExpressionEmission,
	field *types.Var,
) (api.ExpressionEmission, error) {
	if !fieldInType(parentType, field) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	fieldType, err := children.RepresentedType(
		context.WithRole(api.RoleFieldReceiver),
		source,
		field.Type(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parentLogical, err := children.RepresentedType(
		context.WithRole(api.RoleFieldReceiver),
		source,
		parentType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parentStorage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		parentType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := context.Names().Member(field)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		parent.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.FieldName),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{
				fieldType.Value(),
				parentLogical.Value(),
				parentStorage.Value(),
				context.Factory().LiteralTypeNode(
					context.Factory().StringLiteral(name, tsgo.TokenFlagsNone),
				),
			},
			[]tsgo.Expression{
				parent.Value(),
				context.Factory().StringLiteral(name, tsgo.TokenFlagsNone),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			parent.Requests(),
			fieldType.Requests(),
			parentLogical.Requests(),
			parentStorage.Requests(),
			runtime.Requests(),
		),
	)
}

func canonicalPointerTarget(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	pointer api.ExpressionEmission,
	element types.Type,
) (api.StoreTargetEmission, error) {
	storage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		element,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	receiver, err := api.NewExpressionEmission(
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
		),
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewCanonicalStoragePropertyStoreTargetEmission(
		context.Factory(),
		receiver,
		pointerruntime.CellValueName,
		element,
	)
}
