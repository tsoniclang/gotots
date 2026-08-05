package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
		receiver, receiverType, err = projectFieldValue(
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
		parent, parentDirect, err := dereferencePointer(
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
			parentDirect,
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
) (api.ExpressionEmission, bool, error) {
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(element),
		false,
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
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
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

func projectFieldPointer(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	parentType types.Type,
	parent api.ExpressionEmission,
	field *types.Var,
	parentDirect bool,
) (api.ExpressionEmission, error) {
	if !fieldInType(parentType, field) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	parent, err := joinNominalFieldCallableABI(
		context,
		parentType,
		field,
		parent,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	providerField, providerOwned, err := providerboundary.StructField(
		context,
		parentType,
		field,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	fieldContext := context
	if providerOwned {
		fieldContext, err = context.WithProviderScalarRepresentation()
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	fieldRepresentation, err := pointertype.Observe(
		fieldContext,
		types.NewPointer(field.Type()),
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	fieldType, err := children.RepresentedType(
		fieldContext.WithRole(api.RoleFieldReceiver),
		source,
		field.Type(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	fieldStorage, err := fieldContext.Values().StorageType(
		fieldContext.WithRole(api.RoleStorageType),
		source,
		field.Type(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parentLogical, err := children.RepresentedType(
		fieldContext.WithRole(api.RoleFieldReceiver),
		source,
		parentType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var parentStorage api.TypeEmission
	var parentRepresentationRequests []api.RootRequest
	if parentDirect {
		parentStorage, err = fieldContext.Values().StorageType(
			fieldContext.WithRole(api.RoleStorageType),
			source,
			parentType,
		)
	} else {
		parentRepresentation, representationErr := pointertype.Observe(
			fieldContext,
			types.NewPointer(parentType),
			true,
		)
		if representationErr != nil {
			return api.ExpressionEmission{}, representationErr
		}
		parentStorage, err = fieldContext.ContainerStorage().PointerStorageType(
			fieldContext.WithRole(api.RoleStorageType),
			source,
			parentType,
			parentRepresentation,
		)
		parentRepresentationRequests = parentRepresentation.Requests()
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name := ""
	method := pointerruntime.FieldName
	receiver := parent
	if parentDirect && !providerOwned {
		receiver, name, err = namedstructstorage.DemandFieldOwner(
			context.WithRole(api.RoleStorageType),
			parentType,
			field,
			parent,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		method = pointerruntime.ObjectFieldName
	} else if providerOwned {
		name = providerField.Member()
		if parentDirect {
			method = pointerruntime.ObjectFieldName
		}
	} else {
		name, err = context.Names().Member(field)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	typeArguments := []tsgo.TypeNode{
		fieldType.Value(),
		parentLogical.Value(),
		parentStorage.Value(),
		context.Factory().LiteralTypeNode(
			context.Factory().StringLiteral(name, tsgo.TokenFlagsNone),
		),
	}
	if parentDirect {
		typeArguments = []tsgo.TypeNode{
			fieldType.Value(),
			parentStorage.Value(),
			context.Factory().LiteralTypeNode(
				context.Factory().StringLiteral(name, tsgo.TokenFlagsNone),
			),
		}
	}
	raw, err := api.NewExpressionEmission(
		receiver.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(method),
				tsgo.NodeFlagsNone,
			),
			nil,
			typeArguments,
			[]tsgo.Expression{
				receiver.Value(),
				context.Factory().StringLiteral(name, tsgo.TokenFlagsNone),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			receiver.Requests(),
			fieldType.Requests(),
			fieldStorage.Requests(),
			parentLogical.Requests(),
			parentStorage.Requests(),
			parentRepresentationRequests,
			runtime.Requests(),
			fieldRepresentation.Requests(),
		),
	)
	if err != nil || !providerOwned {
		return raw, err
	}
	return providerboundary.ProjectStructFieldPointer(
		context,
		fieldContext,
		children,
		source,
		field.Type(),
		raw,
	)
}
