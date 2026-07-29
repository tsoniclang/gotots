package array

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) EmitIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	if !types.Identical(context.TypesInfo().TypeOf(source), a.ElementType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleArrayReceiver).
			WithExpectedType(a.sourceType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index, err := emitIndex(context, children, source.Index)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporaryArrayReceiver,
		expressionoperands.Present(receiver),
		expressionoperands.Present(index),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	return api.NewExpressionEmission(
		ordered.Before(),
		callMember(
			context,
			a.storage(context, values[0]),
			arraymember.Get,
			values[1],
		),
		ordered.Requests(),
	)
}

func (a RuntimeArray) EmitStoreTarget(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.StoreTargetEmission, error) {
	if !types.Identical(
		context.TypesInfo().TypeOf(source.X),
		a.sourceType,
	) {
		return api.StoreTargetEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleArrayReceiver).
			WithExpectedType(a.sourceType),
		source.X,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	index, err := emitIndex(context, children, source.Index)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	targetReceiver, err := api.NewExpressionEmission(
		receiver.Before(),
		a.storage(context, receiver.Value()),
		receiver.Requests(),
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewAccessorStoreTargetEmission(
		targetReceiver,
		arraymember.Get.Name(),
		arraymember.Set.Name(),
		[]api.ExpressionEmission{index},
		a.ElementType(),
	)
}

func (a RuntimeArray) EmitLength(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	if len(source.Args) != 1 ||
		!types.Identical(
			context.TypesInfo().TypeOf(source.Args[0]),
			a.sourceType,
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleBuiltinArgument).
			WithExpectedType(a.sourceType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := tsgo.Expression(memberProperty(
		context,
		a.storage(context, value.Value()),
		arraymember.Length,
	))
	if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
		target = context.Factory().CallExpression(
			context.Factory().Identifier("BigInt"),
			nil,
			nil,
			[]tsgo.Expression{target},
			tsgo.NodeFlagsNone,
		)
	}
	return api.NewExpressionEmission(
		value.Before(),
		target,
		value.Requests(),
	)
}

func (a RuntimeArray) RangeElement(
	context api.Context,
	receiver tsgo.Expression,
	index tsgo.Expression,
) api.ExpressionEmission {
	return api.DirectExpression(callMember(
		context,
		a.storage(context, receiver),
		arraymember.Get,
		index,
	))
}

func emitIndex(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	alias, represented := basictype.PrimitiveAlias(
		context.TypesSizes(),
		sourceType,
	)
	if !represented || alias == api.PrimitiveBool {
		return api.ExpressionEmission{},
			api.Unsupported(
				context.WithRole(api.RoleArrayIndex),
				api.CategoryExpression,
				source,
			)
	}
	return children.Expression(
		context.
			WithRole(api.RoleArrayIndex).
			WithExpectedType(sourceType),
		source,
	)
}

func variable(
	context api.Context,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
