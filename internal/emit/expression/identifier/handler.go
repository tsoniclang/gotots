package identifier

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
	case types.Universe.Lookup("false"):
		return emitBooleanConstant(context, source, context.Factory().FalseLiteral())
	case types.Universe.Lookup("true"):
		return emitBooleanConstant(context, source, context.Factory().TrueLiteral())
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
		// An untyped constant is a compile-time value with no runtime
		// declaration: project its canonical value at this use's exact
		// contextual type, which the checker recorded on the identifier.
		typeAndValue := context.TypesInfo().Types[source]
		if typeAndValue.Value == nil {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return constantbinding.EmitValue(
			context,
			source,
			typeAndValue.Type,
			typeAndValue.Value,
		)
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

func emitBooleanConstant(
	context api.Context,
	source *ast.Ident,
	literal tsgo.Expression,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	targetType := context.ExpectedType()
	if sourceType == nil ||
		targetType == nil ||
		!types.AssignableTo(sourceType, targetType) ||
		!isBoolean(targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.DirectExpression(literal), nil
}

func isBoolean(source types.Type) bool {
	basic, ok := types.Unalias(source).(*types.Basic)
	return ok && basic.Kind() == types.Bool
}
