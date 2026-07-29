package builtin

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	clearbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/clear"
	complexbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/complex"
	mapbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/map"
	newvalue "github.com/tsoniclang/gotots/internal/emit/expression/builtin/newvalue"
	orderedbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/ordered"
	panicbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/paniccall"
	recoverbuiltin "github.com/tsoniclang/gotots/internal/emit/expression/builtin/recovercall"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
	if target, handled, err := emitGenericMeasure(
		context,
		children,
		source,
		builtin,
		discarded,
	); handled {
		return target, err
	}
	if target, handled, err := panicbuiltin.Emit(
		context,
		children,
		source,
		builtin,
	); handled {
		return target, err
	}
	if target, handled, err := recoverbuiltin.Emit(
		context,
		source,
		builtin,
	); handled {
		return target, err
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
	if target, handled, err := clearbuiltin.Emit(
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
		if target, ok, err := emitConstantMeasure(context, source); ok {
			return target, err
		}
		if len(source.Args) == 1 &&
			supportsStringArgument(
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
		if array, ok := pointerArrayArgument(context, source); ok {
			return emitPointerArrayMeasure(
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
		if target, ok, err := emitConstantMeasure(context, source); ok {
			return target, err
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
		if array, ok := pointerArrayArgument(context, source); ok {
			return emitPointerArrayMeasure(
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
		if source.Ellipsis.IsValid() {
			return emitAppendSpread(context, children, source, discarded)
		}
		return emitAppend(context, children, source, discarded)
	case types.Universe.Lookup("copy"):
		return emitCopy(context, children, source, discarded)
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
}

func EmitDeferred(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, bool, error) {
	if source == nil || builtin == nil {
		return api.ExpressionEmission{}, false, nil
	}
	if target, handled, err := panicbuiltin.EmitDeferred(
		context,
		children,
		source,
		builtin,
	); handled {
		return target, true, err
	}
	if target, handled, err := mapbuiltin.EmitDeferred(
		context,
		children,
		source,
		builtin,
	); handled {
		return target, true, err
	}
	if target, handled, err := clearbuiltin.EmitDeferred(
		context,
		children,
		source,
		builtin,
	); handled {
		return target, true, err
	}
	if types.Object(builtin) == types.Universe.Lookup("copy") {
		target, err := emitDeferredCopy(context, children, source)
		return target, true, err
	}
	return api.ExpressionEmission{}, false, nil
}

func emitConstantMeasure(
	context api.Context,
	source *ast.CallExpr,
) (api.ExpressionEmission, bool, error) {
	if source == nil || len(source.Args) != 1 {
		return api.ExpressionEmission{}, false, nil
	}
	facts, ok := context.TypesInfo().Types[source]
	if !ok || facts.Type == nil || facts.Value == nil {
		return api.ExpressionEmission{}, false, nil
	}
	if facts.Value.Kind() != constant.Int {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(facts.Type, expected) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := constantvalue.EmitValue(
		context.WithRole(api.RoleBuiltinArgument),
		source,
		facts.Type,
		facts.Value,
	)
	return target, true, err
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
	_ api.Context,
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	return slicevalue.Source(sourceType)
}

func projectDefinedSlice(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return slicevalue.Project(context, sourceType, value)
}

func wrapDefinedSlice(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return slicevalue.Wrap(context, sourceType, value)
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

func pointerArrayArgument(
	context api.Context,
	source *ast.CallExpr,
) (arrayvalue.RuntimeArray, bool) {
	if source == nil || len(source.Args) != 1 {
		return arrayvalue.RuntimeArray{}, false
	}
	sourceType := context.TypesInfo().TypeOf(source.Args[0])
	_, elementType, ok := pointertype.Resolve(sourceType)
	if !ok {
		if defined, definedOK := definedtype.ResolvePointer(sourceType); definedOK {
			pointer, _ := defined.Pointer()
			elementType = pointer.Elem()
			ok = true
		}
	}
	if !ok {
		return arrayvalue.RuntimeArray{}, false
	}
	return arrayvalue.Resolve(context, elementType)
}

func emitArrayMeasure(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	array arrayvalue.RuntimeArray,
	discarded bool,
) (api.ExpressionEmission, error) {
	result := context.TypesInfo().TypeOf(source)
	if result == nil ||
		(!discarded &&
			(context.ExpectedType() == nil ||
				!types.AssignableTo(result, context.ExpectedType()))) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return array.EmitLength(context, children, source)
}

func emitPointerArrayMeasure(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	array arrayvalue.RuntimeArray,
	discarded bool,
) (api.ExpressionEmission, error) {
	resultType := context.TypesInfo().TypeOf(source)
	if resultType == nil ||
		(!discarded &&
			(context.ExpectedType() == nil ||
				!types.AssignableTo(resultType, context.ExpectedType()))) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	argumentType := context.TypesInfo().TypeOf(source.Args[0])
	argument, err := children.Expression(
		context.
			WithRole(api.RoleBuiltinArgument).
			WithExpectedType(argumentType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := context.Names().Temporary(api.TemporaryCallArgument)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	length, err := constantvalue.EmitValue(
		context.WithRole(api.RoleBuiltinArgument),
		source,
		resultType,
		constant.MakeInt64(array.Length()),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		argument.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						argument.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	before = append(before, length.Before()...)
	return api.NewExpressionEmission(
		before,
		length.Value(),
		api.CombineRequests(argument.Requests(), length.Requests()),
	)
}
