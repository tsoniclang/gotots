package structvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type model struct {
	named      *types.Named
	structType *types.Struct
}

func Convert(
	context api.Context,
	source *ast.CallExpr,
	operandSource ast.Node,
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
		operandSource,
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
	source ast.Node,
	operandSource ast.Node,
	sourceModel model,
	targetModel model,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	copied, err := context.Values().Copy(
		context.WithRole(api.RoleConversionOperand),
		operandSource,
		sourceType(sourceModel),
		operand,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := context.Values().ToStorage(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType(sourceModel),
		copied,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.Values().FromStorage(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType(targetModel),
		stored,
	)
}

func sourceType(source model) types.Type {
	if source.named != nil {
		return source.named
	}
	return source.structType
}
