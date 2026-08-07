package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/emit/value/namedstructstorage"
	"github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func FieldAddress(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
	element types.Type,
) (api.ExpressionEmission, error) {
	resolved, ok := fieldPath(context, selected)
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
	resolved, ok := fieldPath(context, selected)
	if !ok || !Valid(context, source, selected, types.FieldVal) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	var (
		receiver api.ExpressionEmission
		err      error
	)
	facts, hasFacts := context.TypesInfo().TypeAndValue(source.X)
	if hasFacts && facts.Addressable() {
		target, err := children.StoreTarget(
			context.
				WithRole(api.RoleAssignmentTarget).
				WithExpectedType(resolved.root),
			source.X,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		receiver, err = target.MutableValue(
			context.WithRole(api.RoleAssignmentTarget),
			source.X,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
	} else {
		receiver, err = children.Expression(
			context.
				WithRole(api.RoleAssignmentTarget).
				WithExpectedType(resolved.root),
			source.X,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
	}
	receiverType := resolved.root
	for _, field := range resolved.fields[:len(resolved.fields)-1] {
		receiver, receiverType, err = projectMutableFieldValue(
			context,
			children,
			source,
			receiverType,
			receiver,
			field,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
	}
	if _, _, _, pointer := pointerType(receiverType); pointer {
		receiver, receiverType, err = dereferenceValue(
			context,
			children,
			source,
			receiverType,
			receiver,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
	}
	field := resolved.fields[len(resolved.fields)-1]
	if !fieldInType(receiverType, field) ||
		!types.Identical(field.Type(), resolved.effective) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err = joinNominalFieldCallableABI(
		context,
		receiverType,
		field,
		receiver,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	target, providerOwned, err := providerboundary.StructFieldStoreTarget(
		context.WithRole(api.RoleAssignmentTarget),
		children,
		source,
		receiverType,
		field,
		receiver,
	)
	if err != nil || providerOwned {
		return target, err
	}
	target, storageSelected, err := namedstructstorage.FieldTarget(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		receiverType,
		field,
		receiver,
	)
	if err != nil || storageSelected {
		return target, err
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

func addressSource(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	resolved path,
) (api.ExpressionEmission, error) {
	rootSource, resolved, ok := expandAddressPath(context, source.X, resolved)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	root, err := children.Expression(
		context.
			WithRole(api.RoleFieldReceiver).
			WithExpectedType(resolved.root),
		rootSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return projectAddress(context, children, source, resolved, root)
}

func projectAddress(
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
			current, currentType, err = loadAddressParent(
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
		current, err = joinNominalFieldCallableABI(
			context,
			currentType,
			field,
			current,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		providerField, providerOwned, err := providerboundary.StructField(
			context,
			currentType,
			field,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		name := ""
		if providerOwned {
			name = providerField.Member()
		} else {
			name, err = context.Names().Member(field)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
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
	targetType, err := children.RepresentedType(
		context.WithRole(api.RoleUnaryOperand),
		source,
		resolved.effective,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return pointermarker.Operation(
		context,
		tsoniccore.SymbolAddressOf,
		[]api.TypeEmission{targetType},
		[]api.ExpressionEmission{current},
	)
}

func loadAddressParent(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, types.Type, error) {
	_, element, defined, ok := pointerType(sourceType)
	if !ok {
		return api.ExpressionEmission{}, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if defined {
		model, _ := definedtype.ResolvePointer(sourceType)
		var err error
		pointer, err = model.Project(context, pointer)
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
	guarded, err := pointermarker.Guard(context, pointer)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	loaded, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolLoadPointer,
		[]api.TypeEmission{targetElement},
		[]api.ExpressionEmission{guarded},
	)
	return loaded, element, err
}
