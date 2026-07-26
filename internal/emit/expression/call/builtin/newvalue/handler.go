package newvalue

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	if source == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	identifier, ok := source.Fun.(*ast.Ident)
	if !ok ||
		context.TypesInfo().Uses[identifier] != types.Universe.Lookup("new") ||
		source.Ellipsis != token.NoPos ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	resultType := context.TypesInfo().TypeOf(source)
	pointer, element, represented := pointertype.Scalar(
		context.TypesSizes(),
		resultType,
	)
	if !represented ||
		!types.Identical(context.TypesInfo().TypeOf(source.Args[0]), element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected == nil ||
		!types.AssignableTo(pointer, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleCallArgument),
		source.Args[0],
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleCallArgument),
		source.Args[0],
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(zero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().NewExpression(
			context.Factory().Identifier(reference.Name()),
			[]tsgo.TypeNode{targetElement.Value()},
			[]tsgo.Expression{zero.Value()},
		),
		api.CombineRequests(
			targetElement.Requests(),
			zero.Requests(),
			reference.Requests(),
		)...,
	), nil
}
