package constant

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/stringvalue"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

// EmitValue emits a constant's canonical checker value as a target expression,
// selected by the value kind and target type. It reads the go/constant value
// only; it never evaluates the source spelling. This is the single owner of
// constant-value materialization: a typed constant's binding projects its own
// type here, and every use of an untyped constant projects its exact contextual
// type here.
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
	if defined, ok := definedtype.ResolveBasic(targetType); ok {
		operationContext, err := defined.OperationContext(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		underlying, err := EmitValue(
			operationContext,
			source,
			defined.Underlying(),
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return defined.Wrap(context, underlying)
	}
	// A float target owns its materialization regardless of the constant's value
	// kind: `const x float64 = 5` carries an Int value that must still render as a
	// float. Float-kind values only ever have float targets, so a non-float
	// target below never sees one.
	if _, ok := floatvalue.Describe(targetType); ok {
		return floatvalue.EmitConstant(context, source, targetType, value)
	}
	if _, ok := complexvalue.Describe(targetType); ok {
		return complexvalue.EmitConstant(context, source, targetType, value)
	}
	if _, ok := integervalue.Describe(context.TypesSizes(), targetType); ok {
		if value.Kind() == constant.Float {
			value = constant.ToInt(value)
		}
		return integervalue.EmitConstant(context, source, targetType, value)
	}
	switch value.Kind() {
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

// RequiresDeferredBinding reports whether materializing a package constant at
// ESM module evaluation would invoke the package-local runtime representation
// of a defined basic type. Those constants use a typed value thunk so legal Go
// same-package file cycles cannot observe an uninitialized target class.
func RequiresDeferredBinding(selected *types.Const) bool {
	if selected == nil || selected.Pkg() == nil ||
		selected.Parent() != selected.Pkg().Scope() ||
		IsUntyped(selected.Type()) {
		return false
	}
	_, ok := definedtype.ResolveBasic(selected.Type())
	return ok
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
