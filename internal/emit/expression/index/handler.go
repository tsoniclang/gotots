package index

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	operandType := context.TypesInfo().TypeOf(source.X)
	indexType := context.TypesInfo().TypeOf(source.Index)
	resultType := context.TypesInfo().TypeOf(source)
	if !basictype.SupportsString(operandType) ||
		!basictype.SupportsStringIndex(context.TypesSizes(), indexType) ||
		!isByte(resultType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(resultType, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleIndexOperand).
			WithExpectedType(types.Typ[types.String]),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexExpected := indexType
	if isUntypedInteger(indexExpected) {
		indexExpected = types.Typ[types.Int]
	}
	index, err := children.Expression(
		context.
			WithRole(api.RoleIndexValue).
			WithExpectedType(indexExpected),
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(operand.Before()) != 0 || len(index.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Runtime(
		api.RuntimeStringIndex,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			[]tsgo.Expression{operand.Value(), index.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			operand.Requests(),
			index.Requests(),
			reference.Requests(),
		)...,
	), nil
}

func isByte(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return ok && basic.Kind() == types.Uint8
}

func isUntypedInteger(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return ok &&
		basic.Info()&types.IsUntyped != 0 &&
		basic.Info()&types.IsInteger != 0
}
