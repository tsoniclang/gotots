package constant

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

// EmitUse emits one bare use of a source-declared untyped constant. Identifier,
// package-selector, and dot-import handlers only identify the object and
// delegate here; this is the sole owner of contextual projection selection.
func EmitUse(
	context api.Context,
	source ast.Expr,
	selected *types.Const,
) (api.ExpressionEmission, error) {
	if source == nil || selected == nil || !IsUntyped(selected.Type()) {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "untyped constant use identity is invalid",
		}
	}
	facts, ok := context.TypesInfo().Types[source]
	if !ok || facts.Value == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	projection, defined, wrapsDefined, ok := effectiveProjection(
		context,
		facts,
	)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if selected.Pkg() != nil &&
		selected.Parent() == selected.Pkg().Scope() {
		reference, err := context.Names().ConstantProjection(
			selected,
			projection,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		target := api.DirectExpression(
			context.Factory().Identifier(reference.Name()),
			reference.Requests()...,
		)
		if wrapsDefined {
			return defined.Wrap(context, target)
		}
		return target, nil
	}
	owner, ok := context.FunctionArtifactOwner()
	if !ok {
		return api.ExpressionEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "local constant use has no enclosing artifact owner",
			}
	}
	request, err := api.NewLocalConstantProjectionRequest(
		owner,
		selected,
		projection,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	base, err := context.Names().Declare(selected)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	projectionName, err := api.ConstantProjectionName(base, projection)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := api.DirectExpression(
		context.Factory().Identifier(projectionName),
		request,
	)
	if wrapsDefined {
		return defined.Wrap(context, target)
	}
	return target, nil
}

// EffectiveProjection selects the concrete basic representation for one
// untyped constant occurrence. go/types may leave a child occurrence untyped
// even when its parent supplies the conversion, so a validated parent
// expectation is the second and only other source.
func EffectiveProjection(
	context api.Context,
	facts types.TypeAndValue,
) (types.BasicKind, bool) {
	projection, _, _, ok := effectiveProjection(context, facts)
	return projection, ok
}

func effectiveProjection(
	context api.Context,
	facts types.TypeAndValue,
) (types.BasicKind, definedtype.Model, bool, bool) {
	if facts.Type == nil || facts.Value == nil {
		return types.Invalid, definedtype.Model{}, false, false
	}
	if projection, defined, wraps, ok := projectionType(facts.Type); ok {
		expected := context.ExpectedType()
		if expected != nil && !types.AssignableTo(facts.Type, expected) {
			return types.Invalid, definedtype.Model{}, false, false
		}
		return projection, defined, wraps, true
	}
	expected := context.ExpectedType()
	projection, defined, wraps, ok := projectionType(expected)
	if !ok || !types.AssignableTo(facts.Type, expected) {
		return types.Invalid, definedtype.Model{}, false, false
	}
	return projection, defined, wraps, true
}

func projectionType(
	sourceType types.Type,
) (types.BasicKind, definedtype.Model, bool, bool) {
	if projection, ok := ProjectionKind(sourceType); ok {
		return projection, definedtype.Model{}, false, true
	}
	defined, ok := definedtype.ResolveBasic(sourceType)
	if !ok {
		return types.Invalid, definedtype.Model{}, false, false
	}
	projection, ok := ProjectionKind(defined.Underlying())
	if !ok {
		return types.Invalid, definedtype.Model{}, false, false
	}
	return projection, defined, true, true
}

// EmitFolded materializes an admitted non-reference constant expression from
// the enclosing node's checker value. The syntax owner decides that its form is
// admitted; this helper only selects the exact concrete result type and routes
// the canonical value through EmitValue.
func EmitFolded(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, bool, error) {
	if source == nil {
		return api.ExpressionEmission{}, false, nil
	}
	facts, ok := context.TypesInfo().Types[source]
	if !ok || facts.Value == nil {
		return api.ExpressionEmission{}, false, nil
	}
	projection, defined, wrapsDefined, ok := effectiveProjection(
		context,
		facts,
	)
	if !ok {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	projectedType, ok := api.ConstantProjectionType(projection)
	if !ok {
		return api.ExpressionEmission{}, true,
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "effective constant projection is invalid",
			}
	}
	var targetType types.Type = projectedType
	if wrapsDefined {
		targetType = defined.Type()
	}
	target, err := EmitValue(
		context,
		source,
		targetType,
		facts.Value,
	)
	return target, true, err
}
