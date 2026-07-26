package constant

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type BindingEmission struct {
	declaration tsgo.VariableDeclaration
	requests    []api.PlacementRequest
}

func (e BindingEmission) Declaration() tsgo.VariableDeclaration {
	return e.declaration
}

func (e BindingEmission) Requests() []api.PlacementRequest {
	return slices.Clone(e.requests)
}

func EmitBinding(
	context api.Context,
	children api.ChildEmitter,
	sourceName *ast.Ident,
	sourceType ast.Expr,
	sourceValue ast.Expr,
	selected *types.Const,
	typeRole api.Role,
	valueRole api.Role,
) (BindingEmission, error) {
	if sourceName == nil || sourceType == nil || sourceValue == nil || selected == nil {
		return BindingEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "constant binding input is nil",
		}
	}
	if sourceName.Name == "_" ||
		context.TypesInfo().Defs[sourceName] != selected ||
		!types.Identical(context.TypesInfo().TypeOf(sourceType), selected.Type()) {
		return BindingEmission{},
			api.Unsupported(context, api.CategoryDeclaration, sourceName)
	}
	typeAndValue, ok := context.TypesInfo().Types[sourceValue]
	if !ok ||
		typeAndValue.Value == nil ||
		!constant.Compare(typeAndValue.Value, token.EQL, selected.Val()) ||
		!types.AssignableTo(typeAndValue.Type, selected.Type()) {
		return BindingEmission{},
			api.Unsupported(context.WithRole(valueRole), api.CategoryExpression, sourceValue)
	}
	value, err := children.Expression(
		context.
			WithRole(valueRole).
			WithExpectedType(selected.Type()),
		sourceValue,
	)
	if err != nil {
		return BindingEmission{}, err
	}
	if len(value.Before()) != 0 {
		return BindingEmission{},
			api.Unsupported(context.WithRole(valueRole), api.CategoryExpression, sourceValue)
	}
	targetType, err := children.RepresentedType(
		context.WithRole(typeRole),
		sourceType,
		selected.Type(),
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
			targetType.Value(),
			value.Value(),
		),
		requests: api.CombineRequests(
			targetType.Requests(),
			value.Requests(),
		),
	}, nil
}
