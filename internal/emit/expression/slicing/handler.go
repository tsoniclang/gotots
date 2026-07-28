package slicing

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	_, _, ok := slicevalue.Resolve(sourceType)
	defined, definedOK := definedtype.ResolveSlice(sourceType)
	ok = ok || definedOK
	resultType := context.TypesInfo().TypeOf(source)
	if !ok ||
		resultType == nil ||
		!types.Identical(resultType, sourceType) ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(resultType, context.ExpectedType()) ||
		(source.Slice3 && source.Max == nil) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleSliceReceiver).
			WithExpectedType(sourceType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if definedOK {
		receiver, err = defined.Project(context, receiver)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	low, err := emitBound(
		context.WithRole(api.RoleSliceLow),
		children,
		source.Low,
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	high, err := emitBound(
		context.WithRole(api.RoleSliceHigh),
		children,
		source.High,
		context.Factory().NullLiteral(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	max := expressionoperands.Omitted(context.Factory().NullLiteral())
	if source.Slice3 {
		max, err = emitBound(
			context.WithRole(api.RoleSliceMax),
			children,
			source.Max,
			nil,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporarySliceOperand,
		expressionoperands.Present(receiver),
		low,
		high,
		max,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	result, err := api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				values[0],
				nil,
				context.Factory().Identifier(
					runtimeslice.MemberName(runtimeslice.MemberSlice),
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			values[1:],
			tsgo.NodeFlagsNone,
		),
		ordered.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if definedOK {
		return defined.Wrap(context, result)
	}
	return result, nil
}

func emitBound(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	omitted tsgo.Expression,
) (expressionoperands.Item, error) {
	if source == nil {
		return expressionoperands.Omitted(omitted), nil
	}
	sourceType := context.TypesInfo().TypeOf(source)
	if !basictype.SupportsInteger(context.TypesSizes(), sourceType) {
		return expressionoperands.Item{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	emission, err := children.Expression(
		context.WithExpectedType(sourceType),
		source,
	)
	if err != nil {
		return expressionoperands.Item{}, err
	}
	return expressionoperands.Present(emission), nil
}
