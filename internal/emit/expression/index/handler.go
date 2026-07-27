package index

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapindexexpression "github.com/tsoniclang/gotots/internal/emit/expression/mapindex"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	operandType := context.TypesInfo().TypeOf(source.X)
	if _, ok := types.Unalias(operandType).(*types.Map); ok {
		return mapindexexpression.Emit(context, children, source)
	}
	if array, ok := arrayvalue.Resolve(context, operandType); ok {
		return array.EmitIndex(context, children, source)
	}
	if _, _, ok := slicevalue.Scalar(
		context.TypesSizes(),
		operandType,
	); ok {
		return emitSliceIndex(context, children, source)
	}
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
	target := tsgo.Expression(context.Factory().CallExpression(
		context.Factory().Identifier(reference.Name()),
		nil,
		nil,
		[]tsgo.Expression{operand.Value(), index.Value()},
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
	return api.DirectExpression(
		target,
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

func emitSliceIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	_, elementType, ok := slicevalue.Scalar(
		context.TypesSizes(),
		sourceType,
	)
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
