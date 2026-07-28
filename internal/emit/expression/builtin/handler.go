package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	complexbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/complex"
	mapbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/map"
	newvalue "github.com/tsoniclang/gotots/internal/emit/expression/builtin/newvalue"
	orderedbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/ordered"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
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
	if target, handled, err := complexbuiltin.Emit(
		context,
		children,
		source,
		builtin,
		discarded,
	); handled {
		return target, err
	}
	if target, handled, err := mapbuiltin.Emit(
		context,
		children,
		source,
		builtin,
		discarded,
	); handled {
		return target, err
	}
	if target, handled, err := orderedbuiltin.Emit(
		context,
		children,
		source,
		builtin,
	); handled {
		return target, err
	}
	switch types.Object(builtin) {
	case types.Universe.Lookup("new"):
		return newvalue.Emit(context, children, source, builtin)
	case types.Universe.Lookup("make"):
		return emitMake(context, children, source, discarded)
	case types.Universe.Lookup("len"):
		if len(source.Args) == 1 &&
			basictype.SupportsString(
				context.TypesInfo().TypeOf(source.Args[0]),
			) {
			return emitStringLength(
				context,
				children,
				source,
				discarded,
			)
		}
		if array, ok := arrayArgument(context, source); ok {
			return emitArrayMeasure(
				context,
				children,
				source,
				array,
				discarded,
			)
		}
		return emitMeasure(
			context,
			children,
			source,
			discarded,
			runtimeslice.MemberLength,
		)
	case types.Universe.Lookup("cap"):
		if array, ok := arrayArgument(context, source); ok {
			return emitArrayMeasure(
				context,
				children,
				source,
				array,
				discarded,
			)
		}
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

func Object(
	info *types.Info,
	source ast.Expr,
) (*types.Builtin, bool) {
	identifier, ok := source.(*ast.Ident)
	if !ok || info == nil {
		return nil, false
	}
	builtin, ok := info.Uses[identifier].(*types.Builtin)
	return builtin, ok
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

func arrayArgument(
	context api.Context,
	source *ast.CallExpr,
) (arrayvalue.RuntimeArray, bool) {
	if source == nil || len(source.Args) != 1 {
		return arrayvalue.RuntimeArray{}, false
	}
	return arrayvalue.Resolve(
		context,
		context.TypesInfo().TypeOf(source.Args[0]),
	)
}

func emitArrayMeasure(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	array arrayvalue.RuntimeArray,
	discarded bool,
) (api.ExpressionEmission, error) {
	result := context.TypesInfo().TypeOf(source)
	if discarded ||
		result == nil ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(result, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return array.EmitLength(context, children, source)
}
