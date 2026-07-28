package binary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitSliceNilEquality(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	if source.Op != token.EQL && source.Op != token.NEQ {
		return api.ExpressionEmission{}, false, nil
	}
	valueSource, nilSource, ok := sliceAndNil(
		context.TypesInfo(),
		source.X,
		source.Y,
	)
	if !ok {
		valueSource, nilSource, ok = sliceAndNil(
			context.TypesInfo(),
			source.Y,
			source.X,
		)
	}
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	valueType := context.TypesInfo().TypeOf(valueSource)
	if _, _, represented := slicevalue.Resolve(valueType); !represented {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if context.ExpectedType() == nil ||
		!types.AssignableTo(types.Typ[types.Bool], context.ExpectedType()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if context.TypesInfo().Uses[nilSource] != types.Universe.Lookup("nil") {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleBinaryLeft).
			WithExpectedType(valueType),
		valueSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target := tsgo.Expression(context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			value.Value(),
			nil,
			context.Factory().Identifier(
				runtimeslice.MemberName(runtimeslice.MemberIsNil),
			),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	))
	if source.Op == token.NEQ {
		target = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			target,
		)
	}
	result, err := api.NewExpressionEmission(
		value.Before(),
		target,
		value.Requests(),
	)
	return result, true, err
}

func sliceAndNil(
	info *types.Info,
	value ast.Expr,
	nilCandidate ast.Expr,
) (ast.Expr, *ast.Ident, bool) {
	identifier, ok := nilCandidate.(*ast.Ident)
	if !ok || info.Uses[identifier] != types.Universe.Lookup("nil") {
		return nil, nil, false
	}
	if _, ok := types.Unalias(info.TypeOf(value)).(*types.Slice); !ok {
		return nil, nil, false
	}
	return value, identifier, true
}
