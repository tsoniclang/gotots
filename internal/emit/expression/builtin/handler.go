package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
	discarded bool,
) (api.ExpressionEmission, error) {
	if source == nil || builtin == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	switch types.Object(builtin) {
	case types.Universe.Lookup("make"):
		return emitMake(context, children, source, discarded)
	case types.Universe.Lookup("len"):
		return emitMeasure(
			context,
			children,
			source,
			discarded,
			runtimeslice.MemberLength,
		)
	case types.Universe.Lookup("cap"):
		return emitMeasure(
			context,
			children,
			source,
			discarded,
			runtimeslice.MemberCapacity,
		)
	case types.Universe.Lookup("append"):
		return emitAppend(context, children, source, discarded)
	case types.Universe.Lookup("copy"):
		return emitCopy(context, children, source, discarded)
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
}

func builtinObject(info *types.Info, source ast.Expr) (*types.Builtin, bool) {
	identifier, ok := source.(*ast.Ident)
	if !ok || info == nil {
		return nil, false
	}
	builtin, ok := info.Uses[identifier].(*types.Builtin)
	return builtin, ok
}

func SliceBuiltin(
	info *types.Info,
	source ast.Expr,
) (*types.Builtin, bool) {
	return builtinObject(info, source)
}

func resultType(
	context api.Context,
	source *ast.CallExpr,
	discarded bool,
) (types.Type, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	if !discarded &&
		(context.ExpectedType() == nil ||
			!types.AssignableTo(sourceType, context.ExpectedType())) {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	return sourceType, nil
}

func scalarSlice(
	context api.Context,
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	return slicevalue.Scalar(context.TypesSizes(), sourceType)
}
