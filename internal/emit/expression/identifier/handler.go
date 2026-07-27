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
		return emitUntypedConstantProjection(context, source, constObject)
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

// emitUntypedConstantProjection emits a use of a source-declared untyped
// constant as a constant-size reference to its projection at this use's exact
// contextual basic representation, which the checker recorded on the identifier.
// A package-level constant projects to a module-level binding owned by the
// constant; a function-local constant projects to a prologue binding owned by
// the enclosing function. Neither inlines the value, so output stays
// O(value-size + uses).
func emitUntypedConstantProjection(
	context api.Context,
	source *ast.Ident,
	constObject *types.Const,
) (api.ExpressionEmission, error) {
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
	if constObject.Pkg() != nil &&
		constObject.Parent() == constObject.Pkg().Scope() {
		reference, err := context.Names().ConstantProjection(
			constObject,
			projection,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().Identifier(reference.Name()),
			reference.Requests()...,
		), nil
	}
	owner := context.ArtifactOwner()
	if owner == nil {
		return api.ExpressionEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "local constant use has no enclosing function owner",
			}
	}
	request, err := api.NewLocalConstantProjectionRequest(
		owner,
		constObject,
		projection,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	base, err := context.Names().Declare(constObject)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().Identifier(
			api.ConstantProjectionName(base, projection),
		),
		request,
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
