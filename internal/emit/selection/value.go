package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/emit/value/namedstructstorage"
	"github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
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

func projectMutableValue(
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
		current, currentType, err = projectMutableFieldValue(
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
	projected, providerOwned, err := providerboundary.ReadStructField(
		context,
		children,
		source,
		currentType,
		field,
		current,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	if providerOwned {
		return projected, field.Type(), nil
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

func projectMutableFieldValue(
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
	if !selected {
		target, selected, err = providerboundary.StructFieldStoreTarget(
			context.WithRole(api.RoleStructField),
			children,
			source,
			currentType,
			field,
			current,
		)
		if err != nil {
			return api.ExpressionEmission{}, nil, err
		}
	}
	if selected {
		current, err = target.MutableValue(
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
	return loadAddressParent(
		context,
		children,
		source,
		sourceType,
		value,
	)
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
