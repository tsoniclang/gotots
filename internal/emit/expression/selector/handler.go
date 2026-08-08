package selector

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	methodexpression "github.com/tsoniclang/gotots/internal/emit/expression/methodexpression"
	methodvalue "github.com/tsoniclang/gotots/internal/emit/expression/methodvalue"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
) (api.ExpressionEmission, error) {
	if selection := context.TypesInfo().SelectionOf(source); selection != nil {
		switch selection.Kind() {
		case types.FieldVal:
			return selectionvalue.FieldValue(
				context,
				children,
				source,
				selection,
			)
		case types.MethodVal:
			return methodvalue.Emit(context, children, source, selection)
		case types.MethodExpr:
			return methodexpression.Emit(context, children, source, selection)
		default:
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
	}
	qualifier, ok := source.X.(*ast.Ident)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	packageName, ok := context.TypesInfo().UseOf(qualifier).(*types.PkgName)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	object := context.TypesInfo().UseOf(source.Sel)
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
		target, err := context.Values().FromStorage(
			context,
			source,
			variable.Type(),
			api.DirectExpression(
				reference.Expression(context.Factory()),
				reference.Requests()...,
			),
		)
		if err != nil || !reference.ProviderBoundary() {
			return target, err
		}
		target, _, err = providerboundary.FromProviderValue(
			context,
			children,
			nil,
			"",
			variable.Type(),
			target,
		)
		return target, err
	}
	if constObject, ok := object.(*types.Const); ok &&
		(constantbinding.IsUntyped(constObject.Type()) ||
			constantbinding.RequiresDeferredBinding(constObject)) {
		return constantbinding.EmitUse(context, source, constObject)
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
	target := api.DirectExpression(
		reference.Expression(context.Factory()),
		reference.Requests()...,
	)
	if function, ok := object.(*types.Func); ok {
		if reference.ProviderBoundary() {
			return providerboundary.FromProviderSourceCallable(
				context,
				children,
				source,
				function,
				target,
			)
		}
		return cooperativecall.TransportSourceValue(
			context,
			source,
			function,
			target,
		)
	}
	return target, nil
}
