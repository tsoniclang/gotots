package constant

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// ProjectionKind is the exact target basic representation an untyped-constant
// use projects to: the checker's contextual type at that use, required to be a
// concrete predeclared basic type. A defined or still-untyped contextual type
// is not a supported projection target, so callers fail at that typed boundary
// rather than guessing a representation. The basic kind is canonical and
// comparable, so it is a stable per-representation dedup key.
func ProjectionKind(contextual types.Type) (types.BasicKind, bool) {
	basic, ok := types.Unalias(contextual).(*types.Basic)
	if !ok {
		return types.Invalid, false
	}
	_, ok = api.ConstantProjectionType(basic.Kind())
	return basic.Kind(), ok
}

// ProjectionEmission is one materialized untyped-constant projection: an
// exported binding of the constant at one exact target basic representation,
// plus the requests its value and type annotation raise.
type ProjectionEmission struct {
	declaration tsgo.VariableDeclaration
	requests    []api.RootRequest
}

func (e ProjectionEmission) Declaration() tsgo.VariableDeclaration {
	return e.declaration
}

func (e ProjectionEmission) Requests() []api.RootRequest {
	return slices.Clone(e.requests)
}

func (e ProjectionEmission) ExportedStatement(
	factory tsgo.Factory,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{e.declaration},
			tsgo.NodeFlagsConst,
		),
	)
}

// EmitProjection materializes one untyped constant at one exact target basic
// representation as a named exported binding. The value is the constant's
// canonical go/constant value projected at the target basic type through the
// single value owner; the source value spelling is never evaluated. A typed
// constant has one declared binding (EmitBinding) and is never projected, so a
// typed constant reaching this owner is a caller invariant violation.
func EmitProjection(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	selected *types.Const,
	projectionName string,
	projection types.BasicKind,
	typeRole api.Role,
	valueRole api.Role,
) (ProjectionEmission, error) {
	if selected == nil || projectionName == "" ||
		(source == nil && !context.EnvironmentContract()) {
		return ProjectionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "constant projection input is nil",
		}
	}
	if !IsUntyped(selected.Type()) {
		return ProjectionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "typed constant has a declared binding and is never projected",
		}
	}
	targetType, ok := api.ConstantProjectionType(projection)
	if !ok {
		return ProjectionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "constant projection target representation is invalid",
		}
	}
	value, err := EmitValue(
		context.WithRole(valueRole),
		source,
		targetType,
		selected.Val(),
	)
	if err != nil {
		return ProjectionEmission{}, err
	}
	if len(value.Before()) != 0 {
		return ProjectionEmission{},
			api.Unsupported(context.WithRole(valueRole), api.CategoryExpression, source)
	}
	targetTypeEmission, err := children.RepresentedType(
		context.WithRole(typeRole),
		source,
		targetType,
	)
	if err != nil {
		return ProjectionEmission{}, err
	}
	return ProjectionEmission{
		declaration: context.Factory().VariableDeclaration(
			context.Factory().Identifier(projectionName),
			nil,
			targetTypeEmission.Value(),
			value.Value(),
		),
		requests: api.CombineRequests(
			targetTypeEmission.Requests(),
			value.Requests(),
		),
	}, nil
}
