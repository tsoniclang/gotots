package store

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.StoreTargetEmission, error) {
	switch source := source.(type) {
	case *ast.Ident:
		return identifier(context, source)
	case *ast.SelectorExpr:
		return field(context, children, source)
	default:
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
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
	receiver, err := children.StoreTarget(
		context.WithRole(api.RoleAssignmentTarget),
		source.X,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	name, err := context.Names().Member(field)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewStoreTargetEmission(
		context.Factory().PropertyAccessExpression(
			receiver.Value(),
			nil,
			context.Factory().Identifier(name),
			tsgo.NodeFlagsNone,
		),
		field.Type(),
		receiver.Requests(),
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
	return api.NewStoreTargetEmission(
		reference.Expression(context.Factory()),
		variable.Type(),
		reference.Requests(),
	)
}

func selectedField(selection *types.Selection) (*types.Var, bool) {
	if selection == nil ||
		selection.Kind() != types.FieldVal ||
		selection.Indirect() ||
		len(selection.Index()) != 1 {
		return nil, false
	}
	field, ok := selection.Obj().(*types.Var)
	return field, ok && field.IsField() && !field.Embedded()
}
