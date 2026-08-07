package identifier

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.Ident,
) (api.ExpressionEmission, error) {
	object := context.TypesInfo().UseOf(source)
	if object == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	switch object {
	case types.Universe.Lookup("false"), types.Universe.Lookup("true"):
		return emitPredeclaredBoolean(context, source)
	case types.Universe.Lookup("nil"):
		sourceType := context.TypesInfo().TypeOf(source)
		targetType := context.ExpectedType()
		if sourceType == nil ||
			targetType == nil ||
			!types.AssignableTo(sourceType, targetType) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return context.Values().Zero(context, source, targetType)
	case types.Universe.Lookup("iota"):
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if constObject, ok := object.(*types.Const); ok &&
		(constantbinding.IsUntyped(constObject.Type()) ||
			constantbinding.RequiresDeferredBinding(constObject)) {
		return constantbinding.EmitUse(context, source, constObject)
	}
	if variable, ok := object.(*types.Var); ok &&
		!variable.IsField() &&
		variable.Pkg() != nil &&
		variable.Parent() == variable.Pkg().Scope() {
		reference, err := context.Names().PackageVariable(variable)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		target := api.DirectExpression(
			reference.Expression(context.Factory()),
			reference.Requests()...,
		)
		if !reference.ProviderBoundary() {
			return target, nil
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
	if variable, ok := object.(*types.Var); ok {
		if receiver, ok := context.ValueReceiver(variable); ok {
			value := receiver.Value()
			var requests []api.RootRequest
			if context.Role() == api.RoleAssignmentTarget {
				request, err := receiver.CopyRequest()
				if err != nil {
					return api.ExpressionEmission{}, err
				}
				value = context.Factory().Identifier(receiver.CopyName())
				requests = []api.RootRequest{request}
			}
			return api.DirectExpression(value, requests...), nil
		}
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
		target, err = callable.AdaptProjectedSourceValue(
			context,
			children,
			source,
			function,
			target,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if reference.ProviderBoundary() {
			return providerboundary.FromProviderSourceCallable(
				context,
				children,
				source,
				function,
				target,
			)
		}
		return cooperativecall.AdaptSourceValue(
			context,
			children,
			source,
			function,
			target,
		)
	}
	return target, nil
}

// emitPredeclaredBoolean projects the predeclared true/false constant through
// the single constant-value owner. These are literal booleans, not
// source-declared constants, so they are materialized in place rather than
// projected; routing them through EmitValue keeps one value-materialization
// owner rather than a second boolean path.
func emitPredeclaredBoolean(
	context api.Context,
	source *ast.Ident,
) (api.ExpressionEmission, error) {
	typeAndValue, _ := context.TypesInfo().TypeAndValue(source)
	sourceType := context.TypesInfo().TypeOf(source)
	targetType := context.ExpectedType()
	if typeAndValue.Value == nil ||
		sourceType == nil ||
		targetType == nil ||
		!types.AssignableTo(sourceType, targetType) ||
		!isBoolean(targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return constantbinding.EmitValue(
		context,
		source,
		targetType,
		typeAndValue.Value,
	)
}

func isBoolean(source types.Type) bool {
	if source == nil {
		return false
	}
	basic, ok := types.Unalias(source).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Bool
}
