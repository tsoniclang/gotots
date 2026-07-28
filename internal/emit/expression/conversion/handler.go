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
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
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
	operandValue, err := children.Expression(
		context.
			WithRole(api.RoleConversionOperand).
			WithExpectedType(operandFacts.Type),
		operand,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	sourceType := operandFacts.Type
	if defined, ok := definedtype.Resolve(sourceType); ok {
		operandValue, err = api.NewExpressionEmission(
			operandValue.Before(),
			defined.Unwrap(context.Factory(), operandValue.Value()),
			operandValue.Requests(),
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		sourceType = defined.Underlying()
	}
	representedTargetType := targetType
	targetDefined, wrapsTarget := definedtype.Resolve(targetType)
	if wrapsTarget {
		representedTargetType = targetDefined.Underlying()
	}
	var target api.ExpressionEmission
	switch {
	case directArrayConversion(sourceType, representedTargetType):
		target, err = context.Values().Copy(
			context.WithRole(api.RoleConversionOperand),
			operand,
			sourceType,
			operandValue,
		)
	case isInteger(context, representedTargetType):
		target, err = integerconversion.Convert(
			context,
			source,
			sourceType,
			representedTargetType,
			operandValue,
		)
	case isFloat(representedTargetType):
		target, err = floatconversion.Convert(
			context,
			source,
			sourceType,
			representedTargetType,
			operandValue,
		)
	case isComplex(representedTargetType):
		target, err = complexconversion.Convert(
			context,
			source,
			sourceType,
			representedTargetType,
			operandValue,
		)
	case directBasicConversion(sourceType, representedTargetType):
		target = operandValue
	default:
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if wrapsTarget {
		target, err = targetDefined.Wrap(context, target)
	}
	return target, true, err
}

func directArrayConversion(sourceType, targetType types.Type) bool {
	source, sourceOK := types.Unalias(sourceType).(*types.Array)
	target, targetOK := types.Unalias(targetType).(*types.Array)
	return sourceOK && targetOK && types.Identical(source, target)
}

func isInteger(context api.Context, sourceType types.Type) bool {
	_, ok := integervalue.Describe(context.TypesSizes(), sourceType)
	return ok
}

func isFloat(sourceType types.Type) bool {
	_, ok := floatvalue.Describe(sourceType)
	return ok
}

func isComplex(sourceType types.Type) bool {
	_, ok := complexvalue.Describe(sourceType)
	return ok
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
