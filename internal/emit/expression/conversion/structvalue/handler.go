package structvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type model struct {
	named      *types.Named
	structType *types.Struct
}

func Convert(
	context api.Context,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	sourceModel, sourceOK := resolve(sourceType)
	targetModel, targetOK := resolve(targetType)
	if !sourceOK || !targetOK {
		return api.ExpressionEmission{}, false, nil
	}
	if !compatible(sourceModel.structType, targetModel.structType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := emitConversion(
		context,
		source,
		sourceModel,
		targetModel,
		operand,
	)
	return target, true, err
}

func resolve(sourceType types.Type) (model, bool) {
	if sourceType == nil {
		return model{}, false
	}
	actual := types.Unalias(sourceType)
	if named, ok := actual.(*types.Named); ok {
		if named.Obj() == nil || named.TypeParams().Len() != 0 {
			return model{}, false
		}
		structType, ok := named.Underlying().(*types.Struct)
		return model{named: named, structType: structType}, ok
	}
	structType, ok := actual.(*types.Struct)
	return model{structType: structType}, ok
}

func compatible(left, right *types.Struct) bool {
	if left == nil ||
		right == nil ||
		left.NumFields() != right.NumFields() {
		return false
	}
	for index := range left.NumFields() {
		leftField := left.Field(index)
		rightField := right.Field(index)
		if leftField.Name() != rightField.Name() ||
			leftField.Embedded() != rightField.Embedded() ||
			!types.Identical(leftField.Type(), rightField.Type()) {
			return false
		}
	}
	return true
}

func emitConversion(
	context api.Context,
	_ ast.Node,
	_ model,
	targetModel model,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	reference, err := targetConversion(context, targetModel)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	member, err := api.NamedStructOperationMemberName(
		api.NamedStructOperationConvert,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		operand.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				context.Factory().Identifier(member),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{operand.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(operand.Requests(), reference.Requests()),
	)
}

func targetConversion(
	context api.Context,
	target model,
) (api.NameReference, error) {
	if target.named != nil {
		return context.Names().NamedStructOperation(
			target.named.Obj(),
			api.NamedStructOperationConvert,
		)
	}
	return context.Names().AnonymousStruct(
		target.structType,
		api.AnonymousStructDemandConvert,
	)
}
