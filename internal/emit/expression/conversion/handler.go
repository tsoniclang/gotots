package conversion

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	complexconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/complex"
	floatconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/float"
	integerconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/integer"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, bool, error) {
	if source == nil {
		return api.ExpressionEmission{}, false, nil
	}
	calleeFacts, ok := context.TypesInfo().Types[source.Fun]
	if !ok || !calleeFacts.IsType() {
		return api.ExpressionEmission{}, false, nil
	}
	if source.Ellipsis != token.NoPos || len(source.Args) != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := context.TypesInfo().TypeOf(source)
	resultFacts, resultOK := context.TypesInfo().Types[source]
	operand := source.Args[0]
	operandFacts, operandOK := context.TypesInfo().Types[operand]
	if targetType == nil ||
		!resultOK ||
		resultFacts.Type == nil ||
		!types.Identical(resultFacts.Type, targetType) ||
		!operandOK ||
		operandFacts.Type == nil ||
		!types.ConvertibleTo(operandFacts.Type, targetType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(targetType, expected) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if resultFacts.Value != nil {
		if !supportedConstantKind(resultFacts.Value.Kind()) {
			return api.ExpressionEmission{}, true,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		target, err := constantvalue.EmitValue(
			context.WithRole(api.RoleConversionOperand),
			source,
			targetType,
			resultFacts.Value,
		)
		return target, true, err
	}
	if _, ok := integervalue.Describe(context.TypesSizes(), targetType); ok {
		target, err := integerconversion.Emit(
			context,
			children,
			source,
			operandFacts.Type,
			targetType,
		)
		return target, true, err
	}
	if _, ok := floatvalue.Describe(targetType); ok {
		target, err := floatconversion.Emit(
			context,
			children,
			source,
			operandFacts.Type,
			targetType,
		)
		return target, true, err
	}
	if _, ok := complexvalue.Describe(targetType); ok {
		target, err := complexconversion.Emit(
			context,
			children,
			source,
			operandFacts.Type,
			targetType,
		)
		return target, true, err
	}
	if directBasicConversion(operandFacts.Type, targetType) {
		target, err := children.Expression(
			context.
				WithRole(api.RoleConversionOperand).
				WithExpectedType(operandFacts.Type),
			operand,
		)
		return target, true, err
	}
	return api.ExpressionEmission{}, true,
		api.Unsupported(context, api.CategoryExpression, source)
}

func supportedConstantKind(kind constant.Kind) bool {
	switch kind {
	case constant.Bool,
		constant.String,
		constant.Int,
		constant.Float,
		constant.Complex:
		return true
	default:
		return false
	}
}

func directBasicConversion(sourceType, targetType types.Type) bool {
	source, sourceOK := types.Unalias(sourceType).(*types.Basic)
	target, targetOK := types.Unalias(targetType).(*types.Basic)
	if !sourceOK || !targetOK || source.Kind() != target.Kind() {
		return false
	}
	return source.Kind() == types.Bool || source.Kind() == types.String
}
