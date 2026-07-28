package newvalue

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, error) {
	if source == nil ||
		types.Object(builtin) != types.Universe.Lookup("new") {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if source.Ellipsis != token.NoPos ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	argumentFacts, ok := context.TypesInfo().Types[source.Args[0]]
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if !argumentFacts.IsType() {
		return emitExpression(context, children, source, builtin)
	}
	resultType := context.TypesInfo().TypeOf(source)
	pointer, element, represented := pointertype.Resolve(resultType)
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
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.CellName),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{targetElement.Value()},
			[]tsgo.Expression{zero.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			targetElement.Requests(),
			zero.Requests(),
			reference.Requests(),
		)...,
	), nil
}
