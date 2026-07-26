package index

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	_, elementType, ok := slicevalue.Scalar(context.TypesSizes(), sourceType)
	if !ok ||
		context.TypesInfo().TypeOf(source) == nil ||
		!types.Identical(context.TypesInfo().TypeOf(source), elementType) ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(elementType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	indexType := context.TypesInfo().TypeOf(source.Index)
	if !basictype.SupportsInteger(context.TypesSizes(), indexType) {
		return api.ExpressionEmission{},
			api.Unsupported(
				context.WithRole(api.RoleSliceIndex),
				api.CategoryExpression,
				source.Index,
			)
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
	index, err := children.Expression(
		context.
			WithRole(api.RoleSliceIndex).
			WithExpectedType(indexType),
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetReceiver := receiver.Value()
	before := receiver.Before()
	if len(index.Before()) != 0 {
		name, err := context.Names().Temporary(api.TemporarySliceReceiver)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						targetReceiver,
					),
				},
				tsgo.NodeFlagsConst,
			),
		))
		targetReceiver = context.Factory().Identifier(name)
	}
	before = append(before, index.Before()...)
	target := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			targetReceiver,
			nil,
			context.Factory().Identifier(
				runtimeslice.MemberName(runtimeslice.MemberGet),
			),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{index.Value()},
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		before,
		target,
		api.CombineRequests(receiver.Requests(), index.Requests()),
	)
}
