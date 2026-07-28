package store

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
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
	storageType, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
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
		!basictype.SupportsInteger(context.TypesSizes(), indexType) {
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
	index, err := children.Expression(
		context.
			WithRole(api.RoleSliceIndex).
			WithExpectedType(indexType),
		source.Index,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewAccessorStoreTargetEmission(
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
	key, err = maprepresentation.ProjectKey(
		context.WithRole(api.RoleMapKey),
		source.Index,
		mapType.Key(),
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
	object, ok := context.TypesInfo().Uses[source].(*types.Var)
	if !ok {
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
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewStoreTargetEmission(
		context.Factory().Identifier(reference.Name()),
		object.Type(),
		reference.Requests(),
	)
}

func field(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
) (api.StoreTargetEmission, error) {
	selection := context.TypesInfo().Selections[source]
	if selection == nil {
		return packageVariableSelector(context, source)
	}
	field, ok := selectedField(selection)
	if !ok {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	name, err := context.Names().Member(field)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	if selection.Indirect() {
		receiverType := context.TypesInfo().TypeOf(source.X)
		_, element, ok := pointertype.Resolve(receiverType)
		defined, definedOK := definedtype.ResolvePointer(receiverType)
		if definedOK {
			pointer, _ := defined.Pointer()
			element = pointer.Elem()
			ok = true
		}
		if !ok {
			return api.StoreTargetEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		receiver, err := children.Expression(
			context.
				WithRole(api.RoleAssignmentTarget).
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
		targetElement, err := children.RepresentedType(
			context.WithRole(api.RoleAssignmentTarget),
			source.X,
			element,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		storageType, err := context.Values().StorageType(
			context.WithRole(api.RoleStorageType),
			source.X,
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
		stored, err := api.NewExpressionEmission(
			receiver.Before(),
			pointerruntime.CellValue(
				context.Factory(),
				reference.Name(),
				targetElement.Value(),
				storageType.Value(),
				receiver.Value(),
			),
			api.CombineRequests(
				receiver.Requests(),
				targetElement.Requests(),
				storageType.Requests(),
				reference.Requests(),
			),
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		fieldReceiver, err := context.Values().FromStorage(
			context,
			source.X,
			element,
			stored,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		return api.NewPropertyStoreTargetEmission(
			context.Factory(),
			fieldReceiver,
			name,
			field.Type(),
		)
	}
	receiverType := context.TypesInfo().TypeOf(source.X)
	if receiverType == nil {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleAssignmentTarget).
			WithExpectedType(receiverType),
		source.X,
	)
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

func packageVariableSelector(
	context api.Context,
	source *ast.SelectorExpr,
) (api.StoreTargetEmission, error) {
	qualifier, ok := source.X.(*ast.Ident)
	if !ok {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	packageName, ok := context.TypesInfo().Uses[qualifier].(*types.PkgName)
	if !ok {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	variable, ok := context.TypesInfo().Uses[source.Sel].(*types.Var)
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

func selectedField(selection *types.Selection) (*types.Var, bool) {
	if selection == nil ||
		selection.Kind() != types.FieldVal ||
		len(selection.Index()) != 1 {
		return nil, false
	}
	field, ok := selection.Obj().(*types.Var)
	return field, ok && field.IsField() && !field.Embedded()
}
