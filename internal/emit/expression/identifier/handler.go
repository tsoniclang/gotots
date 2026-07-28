package identifier

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
)

func Emit(
	context api.Context,
	_ api.ChildEmitter,
	source *ast.Ident,
) (api.ExpressionEmission, error) {
	object := context.TypesInfo().Uses[source]
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
		constantbinding.IsUntyped(constObject.Type()) {
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
		return api.DirectExpression(
			reference.Expression(context.Factory()),
			reference.Requests()...,
		), nil
	}
	if variable, ok := object.(*types.Var); ok {
		if selected, exists := context.AddressableStorage().Read(
			context,
			variable,
		); exists {
			return selected, nil
		}
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

// emitPredeclaredBoolean projects the predeclared true/false constant through
// the single constant-value owner. These are literal booleans, not
// source-declared constants, so they are materialized in place rather than
// projected; routing them through EmitValue keeps one value-materialization
// owner rather than a second boolean path.
func emitPredeclaredBoolean(
	context api.Context,
	source *ast.Ident,
) (api.ExpressionEmission, error) {
	typeAndValue := context.TypesInfo().Types[source]
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
	basic, ok := types.Unalias(source).(*types.Basic)
	return ok && basic.Kind() == types.Bool
}
