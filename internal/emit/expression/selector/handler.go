package selector

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	if constObject, ok := object.(*types.Const); ok &&
		constantbinding.IsUntyped(constObject.Type()) {
		typeAndValue := context.TypesInfo().Types[source]
		if typeAndValue.Value == nil {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		projection, ok := constantbinding.ProjectionKind(typeAndValue.Type)
		if !ok {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		reference, err := context.Names().ConstantProjection(constObject, projection)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().Identifier(reference.Name()),
			reference.Requests()...,
		), nil
	}
	switch object.(type) {
	case *types.Const, *types.Func:
	default:
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
	receiverValue := receiver.Value()
	requests := receiver.Requests()
	if selection.Indirect() {
		_, element, ok := pointertype.Resolve(receiverType)
		if !ok {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		targetElement, err := children.RepresentedType(
			context.WithRole(api.RoleFieldReceiver),
			source.X,
			element,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		runtime, err := context.Names().Runtime(
			api.RuntimePointer,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		receiverValue = pointerruntime.CellValue(
			context.Factory(),
			runtime.Name(),
			targetElement.Value(),
			receiverValue,
		)
		requests = api.CombineRequests(
			requests,
			targetElement.Requests(),
			runtime.Requests(),
		)
	}
	name, err := context.Names().Member(field)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		receiver.Before(),
		context.Factory().PropertyAccessExpression(
			receiverValue,
			nil,
			context.Factory().Identifier(name),
			tsgo.NodeFlagsNone,
		),
		requests,
	)
}
