package array

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
			WithExpectedType(a.source),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index, err := emitIndex(context, children, source.Index)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := receiver.Before()
	receiverValue := receiver.Value()
	if len(index.Before()) != 0 {
		name, err := context.Names().Temporary(api.TemporaryArrayReceiver)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, variable(context, name, receiverValue))
		receiverValue = context.Factory().Identifier(name)
	}
	before = append(before, index.Before()...)
	return api.NewExpressionEmission(
		before,
		callMember(context, receiverValue, "get", index.Value()),
		api.CombineRequests(receiver.Requests(), index.Requests()),
	)
}

func (a RuntimeArray) EmitStore(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	receiver, err := children.StoreTarget(
		context.WithRole(api.RoleArrayReceiver),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !types.Identical(receiver.SourceType(), a.source) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	index, err := emitIndex(context, children, source.Index)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := index.Before()
	indexValue := index.Value()
	if len(value.Before()) != 0 {
		name, err := context.Names().Temporary(api.TemporaryArrayIndex)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, variable(context, name, indexValue))
		indexValue = context.Factory().Identifier(name)
	}
	before = append(before, value.Before()...)
	return api.NewExpressionEmission(
		before,
		callMember(
			context,
			receiver.Value(),
			"set",
			indexValue,
			value.Value(),
		),
		api.CombineRequests(
			receiver.Requests(),
			index.Requests(),
			value.Requests(),
		),
	)
}

func (a RuntimeArray) EmitLength(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	if len(source.Args) != 1 ||
		!types.Identical(context.TypesInfo().TypeOf(source.Args[0]), a.source) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleBuiltinArgument).
			WithExpectedType(a.source),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := tsgo.Expression(context.Factory().PropertyAccessExpression(
		value.Value(),
		nil,
		context.Factory().Identifier("length"),
		tsgo.NodeFlagsNone,
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

func emitIndex(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	alias, represented := api.PrimitiveAliasFor(
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
