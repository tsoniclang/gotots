package conversion

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	complexconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/complex"
	floatconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/float"
	integerconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/integer"
	pointerconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/pointer"
	slicearrayconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/slicearray"
	stringconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/stringvalue"
	structconversion "github.com/tsoniclang/gotots/internal/emit/expression/conversion/structvalue"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
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
	operandExpected := operandFacts.Type
	if _, interfaceTarget := interfacetype.Resolve(targetType); interfaceTarget {
		operandExpected = interfacevalue.DynamicType(operandExpected)
	}
	operandValue, err := children.Expression(
		context.
			WithRole(api.RoleConversionOperand).
			WithExpectedType(operandExpected),
		operand,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	sourceType := operandFacts.Type
	if target, handled, interfaceErr := interfacevalue.Convert(
		context,
		source,
		sourceType,
		targetType,
		operandValue,
	); handled {
		return target, true, interfaceErr
	}
	if types.Identical(sourceType, targetType) {
		if _, ok := callable.Signature(targetType); ok {
			return operandValue, true, nil
		}
	}
	if target, handled, pointerErr := pointerconversion.Convert(
		context,
		children,
		source,
		sourceType,
		targetType,
		operandValue,
	); handled {
		return target, true, pointerErr
	}
	if target, handled, mapErr := maprepresentation.Convert(
		context,
		source,
		sourceType,
		targetType,
		operandValue,
	); handled {
		return target, true, mapErr
	}
	if defined, ok := definedtype.Resolve(sourceType); ok {
		operandValue, err = defined.Project(context, operandValue)
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
	if arrayTarget, handled, arrayErr := slicearrayconversion.Convert(
		context,
		children,
		source,
		sourceType,
		representedTargetType,
		operandValue,
	); handled {
		target, err = arrayTarget, arrayErr
	} else if structTarget, handled, structErr := structconversion.Convert(
		context,
		source,
		sourceType,
		representedTargetType,
		operandValue,
	); handled {
		target, err = structTarget, structErr
	} else if stringTarget, handled, stringErr := stringconversion.Convert(
		context,
		children,
		source,
		sourceType,
		representedTargetType,
		operandValue,
	); handled {
		target, err = stringTarget, stringErr
	} else {
		switch {
		case directReferenceConversion(sourceType, representedTargetType):
			target = operandValue
		case directArrayConversion(sourceType, representedTargetType):
			target, err = context.Values().Copy(
				context.WithRole(api.RoleConversionOperand),
				operand,
				sourceType,
				operandValue,
			)
		case directCallableConversion(sourceType, representedTargetType):
			target = operandValue
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
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if wrapsTarget {
		target, err = targetDefined.Wrap(context, target)
	}
	return target, true, err
}

func directReferenceConversion(sourceType, targetType types.Type) bool {
	if !types.Identical(sourceType, targetType) {
		return false
	}
	switch types.Unalias(sourceType).(type) {
	case *types.Slice, *types.Pointer:
		return true
	default:
		return false
	}
}

func directCallableConversion(sourceType, targetType types.Type) bool {
	source, sourceOK := callable.Signature(sourceType)
	target, targetOK := callable.Signature(targetType)
	return sourceOK && targetOK && types.Identical(source, target)
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
