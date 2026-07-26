package selector

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
) (api.ExpressionEmission, error) {
	if selection := context.TypesInfo().Selections[source]; selection != nil {
		return emitField(context, children, source, selection)
	}
	qualifier, ok := source.X.(*ast.Ident)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	packageName, ok := context.TypesInfo().Uses[qualifier].(*types.PkgName)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	object := context.TypesInfo().Uses[source.Sel]
	if object == nil || object.Pkg() != packageName.Imported() {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if variable, ok := object.(*types.Var); ok &&
		!variable.IsField() &&
		variable.Parent() == variable.Pkg().Scope() {
		reference, err := context.Names().PackageVariable(variable)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			reference.Expression(context.Factory()),
			reference.Requests()...,
		), nil
	}
	if _, ok := object.(*types.Const); !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().Identifier(reference.Name()),
		reference.Requests()...,
	), nil
}

func emitField(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selection *types.Selection,
) (api.ExpressionEmission, error) {
	if selection.Kind() != types.FieldVal ||
		selection.Indirect() ||
		len(selection.Index()) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	field, ok := selection.Obj().(*types.Var)
	if !ok || !field.IsField() || field.Embedded() {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiverType := context.TypesInfo().TypeOf(source.X)
	if receiverType == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleFieldReceiver).
			WithExpectedType(receiverType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := context.Names().Member(field)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		receiver.Before(),
		context.Factory().PropertyAccessExpression(
			receiver.Value(),
			nil,
			context.Factory().Identifier(name),
			tsgo.NodeFlagsNone,
		),
		receiver.Requests(),
	)
}
