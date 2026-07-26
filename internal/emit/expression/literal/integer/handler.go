package integer

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	exactNumberMaximum = uint64(1<<53 - 1)
	wideBase           = uint64(1 << 32)
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	if !isIntegerLiteralSyntax(source) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	typeAndValue, ok := context.TypesInfo().Types[source]
	if !ok || typeAndValue.Value == nil || typeAndValue.Value.Kind() != constant.Int {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, exact := constant.Int64Val(typeAndValue.Value)
	if !exact {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := context.ExpectedType()
	if targetType == nil {
		targetType = typeAndValue.Type
	}
	width, ok := signedWidth(context, targetType)
	if !ok || !fitsWidth(value, width) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil || !types.AssignableTo(sourceType, targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return emitValue(context, children, source, targetType, width, value)
}

func isIntegerLiteralSyntax(source ast.Expr) bool {
	switch source := source.(type) {
	case *ast.BasicLit:
		return source.Kind == token.INT
	case *ast.UnaryExpr:
		literal, ok := source.X.(*ast.BasicLit)
		return source.Op == token.SUB && ok && literal.Kind == token.INT
	default:
		return false
	}
}

func signedWidth(context api.Context, sourceType types.Type) (int, bool) {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return 0, false
	}
	switch basic.Kind() {
	case types.Int:
		switch context.TypesSizes().Sizeof(types.Typ[types.Int]) {
		case 4:
			return 32, true
		case 8:
			return 64, true
		default:
			return 0, false
		}
	case types.Int64:
		return 64, true
	default:
		return 0, false
	}
}

func fitsWidth(value int64, width int) bool {
	if width == 64 {
		return true
	}
	return value >= math.MinInt32 && value <= math.MaxInt32
}

func emitValue(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	targetType types.Type,
	width int,
	value int64,
) (api.ExpressionEmission, error) {
	typedLiteral := func(value uint64) (api.ExpressionEmission, error) {
		target, err := children.RepresentedType(
			context.WithRole(api.RoleIntegerConstantType),
			source,
			targetType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().AsExpression(
				context.Factory().NumericLiteral(
					strconv.FormatUint(value, 10),
					tsgo.TokenFlagsNone,
				),
				target.Value(),
			),
			target.Requests()...,
		), nil
	}
	if value >= 0 {
		return emitPositive(context, typedLiteral, uint64(value))
	}
	magnitude := uint64(-(value + 1)) + 1
	return emitNegative(context, typedLiteral, width, magnitude)
}

type typedLiteralEmitter func(uint64) (api.ExpressionEmission, error)

func emitPositive(
	context api.Context,
	typedLiteral typedLiteralEmitter,
	value uint64,
) (api.ExpressionEmission, error) {
	if value <= exactNumberMaximum {
		return typedLiteral(value)
	}
	high, low := value>>32, value&(wideBase-1)
	left, err := typedLiteral(high)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	base, err := typedLiteral(wideBase)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result := binary(context, left, tsgo.BinaryOperatorAsteriskToken, base)
	if low == 0 {
		return result, nil
	}
	right, err := typedLiteral(low)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return binary(context, result, tsgo.BinaryOperatorPlusToken, right), nil
}

func emitNegative(
	context api.Context,
	typedLiteral typedLiteralEmitter,
	width int,
	magnitude uint64,
) (api.ExpressionEmission, error) {
	if magnitude <= math.MaxInt32 {
		return subtraction(context, typedLiteral, magnitude)
	}
	if width == 32 {
		minimum, err := subtraction(context, typedLiteral, math.MaxInt32)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		one, err := typedLiteral(1)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return binary(context, minimum, tsgo.BinaryOperatorMinusToken, one), nil
	}

	high := (magnitude + wideBase - 1) / wideBase
	remainder := high*wideBase - magnitude
	negativeHigh, err := subtraction(context, typedLiteral, high)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	base, err := typedLiteral(wideBase)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result := binary(context, negativeHigh, tsgo.BinaryOperatorAsteriskToken, base)
	if remainder == 0 {
		return result, nil
	}
	right, err := typedLiteral(remainder)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return binary(context, result, tsgo.BinaryOperatorPlusToken, right), nil
}

func subtraction(
	context api.Context,
	typedLiteral typedLiteralEmitter,
	magnitude uint64,
) (api.ExpressionEmission, error) {
	zero, err := typedLiteral(0)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if magnitude <= math.MaxInt32 {
		right, err := typedLiteral(magnitude)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return binary(context, zero, tsgo.BinaryOperatorMinusToken, right), nil
	}
	maximum, err := typedLiteral(math.MaxInt32)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	one, err := typedLiteral(1)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	partial := binary(context, zero, tsgo.BinaryOperatorMinusToken, maximum)
	return binary(context, partial, tsgo.BinaryOperatorMinusToken, one), nil
}

func binary(
	context api.Context,
	left api.ExpressionEmission,
	operator tsgo.BinaryOperator,
	right api.ExpressionEmission,
) api.ExpressionEmission {
	return api.DirectExpression(
		context.Factory().BinaryExpression(
			nil,
			left.Value(),
			nil,
			context.Factory().BinaryOperatorToken(operator),
			right.Value(),
		),
		api.CombineRequests(left.Requests(), right.Requests())...,
	)
}
