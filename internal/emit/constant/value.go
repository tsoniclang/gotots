package constant

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/stringvalue"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

// EmitValue emits a constant's canonical checker value as a target expression,
// selected by the value kind and target type. It reads the go/constant value
// only; it never evaluates the source spelling. This is the single owner of
// constant-value materialization: a typed constant's binding projects its own
// type here, and every use of an untyped constant projects its exact contextual
// type here. Float and complex constants remain a typed unsupported boundary
// until their value families exist.
func EmitValue(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	value constant.Value,
) (api.ExpressionEmission, error) {
	if value == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	switch value.Kind() {
	case constant.Int:
		return integervalue.EmitConstant(context, source, targetType, value)
	case constant.String:
		return stringvalue.EmitConstant(context, source, targetType, value)
	case constant.Bool:
		return emitBool(context, source, targetType, value)
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
}

// IsUntyped reports whether a constant's declared type is untyped — the signal
// that the constant is a compile-time value with no single runtime type, so it
// is projected at each use's contextual type rather than declared once.
func IsUntyped(sourceType types.Type) bool {
	basic, ok := sourceType.(*types.Basic)
	return ok && basic.Info()&types.IsUntyped != 0
}

func emitBool(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	value constant.Value,
) (api.ExpressionEmission, error) {
	basic, ok := types.Unalias(targetType).(*types.Basic)
	if !ok || basic.Kind() != types.Bool {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if constant.BoolVal(value) {
		return api.DirectExpression(context.Factory().TrueLiteral()), nil
	}
	return api.DirectExpression(context.Factory().FalseLiteral()), nil
}
