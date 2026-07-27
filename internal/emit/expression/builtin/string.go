package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitStringLength(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	if len(source.Args) != 1 ||
		!basictype.SupportsString(
			context.TypesInfo().TypeOf(source.Args[0]),
		) ||
		discarded ||
		context.ExpectedResults() != nil ||
		!basictype.SupportsInteger(
			context.TypesSizes(),
			context.TypesInfo().TypeOf(source),
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(context.TypesInfo().TypeOf(source), expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(types.Typ[types.String]),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	length := tsgo.Expression(context.Factory().PropertyAccessExpression(
		value.Value(),
		nil,
		context.Factory().Identifier("length"),
		tsgo.NodeFlagsNone,
	))
	if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
		length = context.Factory().CallExpression(
			context.Factory().Identifier("BigInt"),
			nil,
			nil,
			[]tsgo.Expression{length},
			tsgo.NodeFlagsNone,
		)
	}
	return api.NewExpressionEmission(
		value.Before(),
		length,
		value.Requests(),
	)
}
