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
)

func Emit(
	context api.Context,
	_ api.ChildEmitter,
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
	if context.IntegerRepresentation() == api.IntegerRepresentationNumber &&
		(value > int64(exactNumberMaximum) ||
			value < -int64(exactNumberMaximum)) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil || !types.AssignableTo(sourceType, targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return emitValue(context, value)
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
	case types.Int32:
		return 32, true
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
	value int64,
) (api.ExpressionEmission, error) {
	literal := func(value uint64) (tsgo.Expression, error) {
		return api.IntegerLiteral(
			context.Factory(),
			context.IntegerRepresentation(),
			strconv.FormatUint(value, 10),
		)
	}
	if value >= 0 {
		target, err := literal(uint64(value))
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(target), nil
	}
	magnitude := uint64(-(value + 1)) + 1
	target, err := literal(magnitude)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			target,
		),
	), nil
}
