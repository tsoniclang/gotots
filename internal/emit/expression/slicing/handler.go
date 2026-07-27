package slicing

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type operand struct {
	emission api.ExpressionEmission
	present  bool
	omitted  tsgo.Expression
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	_, _, ok := slicevalue.Scalar(context.TypesSizes(), sourceType)
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
	max := operand{
		omitted: context.Factory().NullLiteral(),
	}
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
	targetReceiver, bounds, before, requests, err := arrange(
		context,
		receiver,
		[]operand{low, high, max},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				targetReceiver,
				nil,
				context.Factory().Identifier(
					runtimeslice.MemberName(runtimeslice.MemberSlice),
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			bounds,
			tsgo.NodeFlagsNone,
		),
		requests,
	)
}

func emitBound(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	omitted tsgo.Expression,
) (operand, error) {
	if source == nil {
		return operand{omitted: omitted}, nil
	}
	sourceType := context.TypesInfo().TypeOf(source)
	if !basictype.SupportsInteger(context.TypesSizes(), sourceType) {
		return operand{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	emission, err := children.Expression(
		context.WithExpectedType(sourceType),
		source,
	)
	if err != nil {
		return operand{}, err
	}
	return operand{emission: emission, present: true}, nil
}

func arrange(
	context api.Context,
	receiver api.ExpressionEmission,
	operands []operand,
) (
	tsgo.Expression,
	[]tsgo.Expression,
	[]tsgo.Statement,
	[]api.RootRequest,
	error,
) {
	capture := len(receiver.Before()) != 0
	for _, item := range operands {
		capture = capture || item.present && len(item.emission.Before()) != 0
	}
	if !capture {
		values := make([]tsgo.Expression, 0, len(operands))
		for _, item := range operands {
			if item.present {
				values = append(values, item.emission.Value())
			} else {
				values = append(values, item.omitted)
			}
		}
		requests := receiver.Requests()
		for _, item := range operands {
			if item.present {
				requests = append(requests, item.emission.Requests()...)
			}
		}
		return receiver.Value(), values, nil, requests, nil
	}
	targetReceiver, before, requests, err := captureOperand(context, receiver)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	values := make([]tsgo.Expression, 0, len(operands))
	for _, item := range operands {
		if !item.present {
			values = append(values, item.omitted)
			continue
		}
		value, statements, itemRequests, err := captureOperand(
			context,
			item.emission,
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		before = append(before, statements...)
		requests = append(requests, itemRequests...)
		values = append(values, value)
	}
	return targetReceiver, values, before, requests, nil
}

func captureOperand(
	context api.Context,
	emission api.ExpressionEmission,
) (tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	name, err := context.Names().Temporary(api.TemporarySliceOperand)
	if err != nil {
		return nil, nil, nil, err
	}
	statements := emission.Before()
	statements = append(statements, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					emission.Value(),
				),
			},
			tsgo.NodeFlagsConst,
		),
	))
	return context.Factory().Identifier(name), statements, emission.Requests(), nil
}
