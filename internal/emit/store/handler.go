package store

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericpointer "github.com/tsoniclang/gotots/internal/emit/generic/pointer"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.StoreTargetEmission, error) {
	switch source := source.(type) {
	case *ast.Ident:
		return identifier(context, source)
	case *ast.IndexExpr:
		if _, ok := maprepresentation.Source(
			context,
			context.TypesInfo().TypeOf(source.X),
		); ok {
			return mapIndex(context, children, source)
		}
		if array, ok := arrayvalue.Resolve(
			context,
			context.TypesInfo().TypeOf(source.X),
		); ok {
			return array.EmitStoreTarget(context, children, source)
		}
		if element, ok := pointerArrayElement(
			context,
			context.TypesInfo().TypeOf(source.X),
		); ok {
			return pointerArrayIndex(
				context,
				children,
				source,
				element,
			)
		}
		return sliceIndex(context, children, source)
	case *ast.ParenExpr:
		target, err := children.StoreTarget(context, source.X)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		if !types.Identical(
			context.TypesInfo().TypeOf(source),
			target.SourceType(),
		) {
			return api.StoreTargetEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return target, nil
	case *ast.SelectorExpr:
		return field(context, children, source)
	case *ast.StarExpr:
		return dereference(context, children, source)
	default:
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
}

func dereference(
	context api.Context,
	children api.ChildEmitter,
	source *ast.StarExpr,
) (api.StoreTargetEmission, error) {
	if source == nil || source.X == nil {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	pointerType := context.TypesInfo().TypeOf(source.X)
	_, element, ok := pointertype.Resolve(pointerType)
	defined, definedOK := definedtype.ResolvePointer(pointerType)
	if definedOK {
		pointer, _ := defined.Pointer()
		element = pointer.Elem()
		ok = true
	}
	if !ok || !types.Identical(context.TypesInfo().TypeOf(source), element) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	pointer, err := children.Expression(
		context.
			WithRole(api.RoleAssignmentTarget).
			WithExpectedType(pointerType),
		source.X,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	if definedOK {
		pointer, err = defined.Project(context, pointer)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
	}
	return canonicalPointerTarget(
		context,
		children,
		source,
		pointer,
		element,
	)
}

func pointerArrayElement(
	context api.Context,
	sourceType types.Type,
) (types.Type, bool) {
	_, element, ok := pointertype.Resolve(sourceType)
	if !ok {
		if defined, definedOK := definedtype.ResolvePointer(sourceType); definedOK {
			pointer, _ := defined.Pointer()
			element = pointer.Elem()
			ok = true
		}
	}
	if !ok {
		return nil, false
	}
	array, ok := arrayvalue.Resolve(context, element)
	if !ok {
		return nil, false
	}
	return array.ElementType(), true
}

func pointerArrayIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
	element types.Type,
) (api.StoreTargetEmission, error) {
	if !types.Identical(context.TypesInfo().TypeOf(source), element) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	pointer, err := children.Address(
		context.
			WithRole(api.RoleAssignmentTarget).
			WithExpectedType(types.NewPointer(element)),
		source,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return canonicalPointerTarget(
		context,
		children,
		source,
		pointer,
		element,
	)
}

func canonicalPointerTarget(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	pointer api.ExpressionEmission,
	element types.Type,
) (api.StoreTargetEmission, error) {
	if target, handled, err := genericpointer.StoreTarget(
		context,
		source,
		element,
		pointer,
	); handled || err != nil {
		return target, err
	}
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(element),
		false,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		logical, err := children.RepresentedType(
			context.WithRole(api.RoleAssignmentTarget),
			source,
			element,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		runtime, err := context.Names().Runtime(
			api.RuntimePointer,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		location, err := api.NewExpressionEmission(
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
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		return api.NewStableIdentityStoreTargetEmission(location, element)
	}
	storageType, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
		representation,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		element,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	receiver, err := api.NewExpressionEmission(
		pointer.Before(),
		pointerruntime.Dereference(
			context.Factory(),
			reference.Name(),
			targetElement.Value(),
			storageType.Value(),
			pointer.Value(),
		),
		api.CombineRequests(
			pointer.Requests(),
			targetElement.Requests(),
			storageType.Requests(),
			reference.Requests(),
			representation.Requests(),
		),
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	if representation.Representation() ==
		api.PointerRepresentationCarrierCanonical {
		return api.NewCanonicalStoragePropertyStoreTargetEmission(
			context.Factory(),
			receiver,
			pointerruntime.CellValueName,
			element,
		)
	}
	return api.NewPropertyStoreTargetEmission(
		context.Factory(),
		receiver,
		pointerruntime.CellValueName,
		element,
	)
}

func sliceIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.StoreTargetEmission, error) {
	receiverType := context.TypesInfo().TypeOf(source.X)
	_, elementType, ok := slicevalue.Resolve(receiverType)
	defined, definedOK := definedtype.ResolveSlice(receiverType)
	if definedOK {
		sliceType, _ := defined.Slice()
		elementType = sliceType.Elem()
		ok = true
	}
	indexType := context.TypesInfo().TypeOf(source.Index)
	if !ok ||
		!types.Identical(context.TypesInfo().TypeOf(source), elementType) ||
		!integeroperand.Supports(context.TypesSizes(), indexType) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleSliceReceiver).
			WithExpectedType(receiverType),
		source.X,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	if definedOK {
		receiver, err = defined.Project(context, receiver)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
	}
	index, err := integeroperand.Emit(
		context.WithRole(api.RoleSliceIndex),
		children,
		source.Index,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewContainerStorageAccessorStoreTargetEmission(
		receiver,
		runtimeslice.MemberName(runtimeslice.MemberGet),
		runtimeslice.MemberName(runtimeslice.MemberSet),
		[]api.ExpressionEmission{index},
		elementType,
	)
}

func mapIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.StoreTargetEmission, error) {
	mapType, ok := maprepresentation.Source(
		context,
		context.TypesInfo().TypeOf(source.X),
	)
	if !ok ||
		!types.Identical(context.TypesInfo().TypeOf(source), mapType.Element()) ||
		!types.AssignableTo(
			context.TypesInfo().TypeOf(source.Index),
			mapType.Key(),
		) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleMapReceiver).
			WithExpectedType(mapType.Type()),
		source.X,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	receiver, err = mapType.StoreReceiver(context, source.X, receiver)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	key, err := children.Expression(
		context.
			WithRole(api.RoleMapKey).
			WithExpectedType(mapType.Key()),
		source.Index,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	key, err = context.Values().Transfer(
		context.WithRole(api.RoleMapKey),
		source.Index,
		context.TypesInfo().TypeOf(source.Index),
		mapType.Key(),
		api.ValueTransferRepresentation,
		key,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	member, err := mapruntime.Name(mapruntime.MemberStore)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	getter, err := mapruntime.Name(mapruntime.MemberLookup)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewCopyingAccessorStoreTargetEmission(
		receiver,
		getter,
		member,
		[]api.ExpressionEmission{key},
		mapType.Element(),
	)
}

func identifier(
	context api.Context,
	source *ast.Ident,
) (api.StoreTargetEmission, error) {
	object, ok := context.TypesInfo().UseOf(source).(*types.Var)
	if !ok {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOfObject(object)
	if sourceType == nil {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if !object.IsField() &&
		object.Pkg() != nil &&
		object.Parent() == object.Pkg().Scope() {
		return packageVariable(context, object)
	}
	if selected, ok, err := context.AddressableStorage().StoreTarget(
		context,
		object,
	); ok || err != nil {
		return selected, err
	}
	if receiver, ok := context.ValueReceiver(object); ok {
		request, err := receiver.CopyRequest()
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		return api.NewStoreTargetEmission(
			context.Factory().Identifier(receiver.CopyName()),
			sourceType,
			[]api.RootRequest{request},
		)
	}
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewStoreTargetEmission(
		reference.Expression(context.Factory()),
		sourceType,
		reference.Requests(),
	)
}

func field(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
) (api.StoreTargetEmission, error) {
	selection := context.TypesInfo().SelectionOf(source)
	if selection == nil {
		return packageVariableSelector(context, source)
	}
	if selection.Kind() != types.FieldVal {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return selectionvalue.FieldStoreTarget(
		context,
		children,
		source,
		selection,
	)
}

func packageVariableSelector(
	context api.Context,
	source *ast.SelectorExpr,
) (api.StoreTargetEmission, error) {
	qualifier, ok := source.X.(*ast.Ident)
	if !ok {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	packageName, ok := context.TypesInfo().UseOf(qualifier).(*types.PkgName)
	if !ok {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	variable, ok := context.TypesInfo().UseOf(source.Sel).(*types.Var)
	if !ok ||
		variable.Pkg() != packageName.Imported() ||
		variable.IsField() ||
		variable.Parent() != variable.Pkg().Scope() {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return packageVariable(context, variable)
}

func packageVariable(
	context api.Context,
	variable *types.Var,
) (api.StoreTargetEmission, error) {
	reference, err := context.Names().PackageVariable(variable)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewCanonicalStorageTargetEmission(
		reference.Expression(context.Factory()),
		variable.Type(),
		reference.Requests(),
	)
}
