package constant

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type BindingEmission struct {
	declaration tsgo.VariableDeclaration
	requests    []api.RootRequest
}

func (e BindingEmission) Declaration() tsgo.VariableDeclaration {
	return e.declaration
}

func (e BindingEmission) Requests() []api.RootRequest {
	return slices.Clone(e.requests)
}

// EmitBinding emits one TYPED constant binding from its checker evidence: the
// target type is the constant's own concrete type, and the value is its
// canonical go/constant value (never the source value spelling). Untyped
// constants have no single type and therefore no binding — each of their uses
// projects the value at its exact contextual type through EmitValue — so an
// untyped constant reaching this owner is a caller invariant violation.
func EmitBinding(
	context api.Context,
	children api.ChildEmitter,
	sourceName *ast.Ident,
	selected *types.Const,
	typeRole api.Role,
	valueRole api.Role,
) (BindingEmission, error) {
	if sourceName == nil || selected == nil {
		return BindingEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "constant binding input is nil",
		}
	}
	if IsUntyped(selected.Type()) {
		return BindingEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "untyped constant has no binding; each use projects its contextual type",
		}
	}
	if sourceName.Name == "_" ||
		context.TypesInfo().Defs[sourceName] != selected {
		return BindingEmission{},
			api.Unsupported(context, api.CategoryDeclaration, sourceName)
	}
	targetType := selected.Type()
	value, err := EmitValue(
		context.WithRole(valueRole),
		sourceName,
		targetType,
		selected.Val(),
	)
	if err != nil {
		return BindingEmission{}, err
	}
	if len(value.Before()) != 0 {
		return BindingEmission{},
			api.Unsupported(context.WithRole(valueRole), api.CategoryExpression, sourceName)
	}
	targetTypeEmission, err := children.RepresentedType(
		context.WithRole(typeRole),
		sourceName,
		targetType,
	)
	if err != nil {
		return BindingEmission{}, err
	}
	targetName, err := context.Names().Declare(selected)
	if err != nil {
		return BindingEmission{}, err
	}
	return BindingEmission{
		declaration: context.Factory().VariableDeclaration(
			context.Factory().Identifier(targetName),
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
