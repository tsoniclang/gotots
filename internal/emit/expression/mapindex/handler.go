package mapindex

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	mapType, ok := maprepresentation.Source(
		context,
		context.TypesInfo().TypeOf(source.X),
	)
	if !ok || !validResultContext(context, source, mapType.Map()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleMapReceiver).
			WithExpectedType(mapType.Type()),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, err = mapType.ReadReceiver(context, source.X, receiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	key, err := children.Expression(
		context.
			WithRole(api.RoleMapKey).
			WithExpectedType(mapType.Key()),
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	key, err = mapType.TransferKey(
		context.WithRole(api.RoleMapKey),
		source.Index,
		key,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values, before, requests, err := maprepresentation.ArrangeOperands(
		context,
		[]api.ExpressionEmission{receiver, key},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	member := mapruntime.MemberLookup
	if context.ExpectedResults() != nil {
		member = mapruntime.MemberLookupOK
	}
	method, err := mapruntime.Name(member)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				values[0],
				nil,
				context.Factory().Identifier(method),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{values[1]},
			tsgo.NodeFlagsNone,
		),
		requests,
	)
}

func validResultContext(
	context api.Context,
	source *ast.IndexExpr,
	mapType *types.Map,
) bool {
	if results := context.ExpectedResults(); results != nil {
		sourceResults, ok := context.TypesInfo().TypeOf(source).(*types.Tuple)
		return ok &&
			sourceResults.Len() == 2 &&
			types.Identical(sourceResults, results) &&
			types.Identical(results.At(0).Type(), mapType.Elem()) &&
			types.Identical(results.At(1).Type(), types.Typ[types.Bool])
	}
	sourceType := context.TypesInfo().TypeOf(source)
	expected := context.ExpectedType()
	return sourceType != nil &&
		expected != nil &&
		types.Identical(sourceType, mapType.Elem()) &&
		types.AssignableTo(sourceType, expected)
}
