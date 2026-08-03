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

const deferredBindingSuffix = "$constant"

type DeferredBindingEmission struct {
	declaration tsgo.FunctionDeclaration
	requests    []api.RootRequest
}

func (e DeferredBindingEmission) Declaration() tsgo.FunctionDeclaration {
	return e.declaration
}

func (e DeferredBindingEmission) Requests() []api.RootRequest {
	return slices.Clone(e.requests)
}

func DeferredBindingName(base string) (string, error) {
	if base == "" {
		return "", &api.NameError{Reason: "deferred constant base name is empty"}
	}
	return base + deferredBindingSuffix, nil
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
		context.TypesInfo().DefOf(sourceName) != selected {
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

// EmitDeferredBinding emits one cycle-safe value thunk for a package constant
// whose defined basic representation constructs a package-local class. The
// function declaration is initialized during ESM instantiation; its value body
// executes only after the package module graph has initialized.
func EmitDeferredBinding(
	context api.Context,
	children api.ChildEmitter,
	sourceName *ast.Ident,
	selected *types.Const,
	typeRole api.Role,
	valueRole api.Role,
) (DeferredBindingEmission, error) {
	if sourceName == nil || !RequiresDeferredBinding(selected) {
		return DeferredBindingEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "deferred constant binding identity is invalid",
		}
	}
	if sourceName.Name == "_" ||
		context.TypesInfo().DefOf(sourceName) != selected {
		return DeferredBindingEmission{},
			api.Unsupported(context, api.CategoryDeclaration, sourceName)
	}
	targetName, err := context.Names().Declare(selected)
	if err != nil {
		return DeferredBindingEmission{}, err
	}
	targetName, err = DeferredBindingName(targetName)
	if err != nil {
		return DeferredBindingEmission{}, err
	}
	value, err := EmitValue(
		context.WithRole(valueRole),
		sourceName,
		selected.Type(),
		selected.Val(),
	)
	if err != nil {
		return DeferredBindingEmission{}, err
	}
	if len(value.Before()) != 0 {
		return DeferredBindingEmission{},
			api.Unsupported(context.WithRole(valueRole), api.CategoryExpression, sourceName)
	}
	targetType, err := children.RepresentedType(
		context.WithRole(typeRole),
		sourceName,
		selected.Type(),
	)
	if err != nil {
		return DeferredBindingEmission{}, err
	}
	return DeferredBindingEmission{
		declaration: context.Factory().FunctionDeclaration(
			[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
			nil,
			context.Factory().Identifier(targetName),
			nil,
			nil,
			targetType.Value(),
			context.Factory().Block(
				[]tsgo.Statement{context.Factory().ReturnStatement(value.Value())},
				true,
			),
		),
		requests: api.CombineRequests(
			targetType.Requests(),
			value.Requests(),
		),
	}, nil
}
