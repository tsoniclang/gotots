package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	unsafeoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin/unsafeoperation"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	runtimestring "github.com/tsoniclang/gotots/internal/emit/runtime/stringvalue"
	"github.com/tsoniclang/gotots/internal/emit/stringvalue"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitUnsafeView(context api.Context, children api.ChildEmitter, source *ast.CallExpr, kind unsafeoperation.Kind, discarded bool) (api.ExpressionEmission, error) {
	signature, ok := context.TypesInfo().TypeOf(source.Fun).(*types.Signature)
	if !ok || signature == nil || signature.Params() == nil || signature.Params().Len() != 2 || len(source.Args) != 2 || source.Ellipsis != token.NoPos {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	argumentType := signature.Params().At(0).Type()
	pointer, ok := types.Unalias(argumentType).(*types.Pointer)
	if !ok || kind == unsafeoperation.String && !types.Identical(pointer.Elem(), types.Typ[types.Uint8]) {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	region, length, err := emitUnsafeViewRegion(context, children, source, pointer)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	ordered, err := expressionoperands.Preserve(context, api.TemporaryCallArgument, expressionoperands.Present(region), expressionoperands.Present(length))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	region = api.DirectExpression(ordered.Values()[0])
	var result api.ExpressionEmission
	if kind == unsafeoperation.String {
		result, err = stringvalue.Construct(context, runtimestring.FromRegionMember, region, api.DirectExpression(ordered.Values()[1]))
	} else {
		storage, storageErr := context.ContainerStorage().ContainerStorageType(context, source, pointer.Elem())
		if storageErr != nil {
			return api.ExpressionEmission{}, storageErr
		}
		runtime, runtimeErr := context.Names().Runtime(api.RuntimeSliceRegion, api.ImportPhaseValue)
		if runtimeErr != nil {
			return api.ExpressionEmission{}, runtimeErr
		}
		result, err = api.NewExpressionEmission(region.Before(), context.Factory().CallExpression(runtime.Expression(context.Factory()), nil,
			[]tsgo.TypeNode{storage.Value()}, []tsgo.Expression{region.Value(), ordered.Values()[1]}, tsgo.NodeFlagsNone),
			api.CombineRequests(storage.Requests(), region.Requests(), runtime.Requests()))
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(append(ordered.Before(), result.Before()...), result.Value(), api.CombineRequests(ordered.Requests(), result.Requests()))
}

func emitUnsafeStringData(context api.Context, children api.ChildEmitter, source *ast.CallExpr) (api.ExpressionEmission, error) {
	if len(source.Args) != 1 || source.Ellipsis != token.NoPos {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOf(source.Args[0])
	value, err := children.Expression(context.WithRole(api.RoleCallArgument).WithExpectedType(sourceType), source.Args[0])
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if model, ok := definedtype.ResolveBasic(sourceType); ok {
		value, err = model.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	return stringvalue.Member(context, runtimestring.DataMember, value)
}

func emitUnsafeViewRegion(context api.Context, children api.ChildEmitter, source *ast.CallExpr, pointer *types.Pointer) (api.ExpressionEmission, api.ExpressionEmission, error) {
	if address, ok := ast.Unparen(source.Args[0]).(*ast.UnaryExpr); ok && address.Op == token.AND {
		if indexed, ok := ast.Unparen(address.X).(*ast.IndexExpr); ok {
			sourceType := context.TypesInfo().TypeOf(indexed.X)
			if _, element, ok := slicevalue.Source(sourceType); ok && types.Identical(element, pointer.Elem()) {
				return emitSliceElementRegion(context, children, indexed, source.Args[1], sourceType, element)
			}
		}
	}
	argument, err := children.Expression(context.WithRole(api.RoleCallArgument).WithExpectedType(pointer), source.Args[0])
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	length, err := integeroperand.Emit(context.WithRole(api.RoleCallArgument), children, source.Args[1])
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	region, err := memorymarker.PointerRegion(context, children, source, pointer.Elem(), argument)
	return region, length, err
}

func emitSliceElementRegion(context api.Context, children api.ChildEmitter, source *ast.IndexExpr, sourceLength ast.Expr, sourceType, element types.Type) (api.ExpressionEmission, api.ExpressionEmission, error) {
	receiver, err := children.Expression(context.WithRole(api.RoleCallArgument).WithExpectedType(sourceType), source.X)
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	receiver, err = slicevalue.Project(context, sourceType, receiver)
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	index, err := integeroperand.Emit(context.WithRole(api.RoleCallArgument), children, source.Index)
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	length, err := integeroperand.Emit(context.WithRole(api.RoleCallArgument), children, sourceLength)
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	if facts, constant := context.TypesInfo().TypeAndValue(sourceLength); !constant || facts.Value == nil {
		length, err = expressionoperands.Snapshot(context, length, api.DirectType(memoryview.CountType(context.Factory())))
		if err != nil {
			return api.ExpressionEmission{}, api.ExpressionEmission{}, err
		}
	}
	ordered, err := expressionoperands.Preserve(context, api.TemporaryAddressOperand,
		expressionoperands.Present(receiver), expressionoperands.Present(index), expressionoperands.Present(length))
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	storage, err := context.ContainerStorage().ContainerStorageType(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(api.RuntimeSliceElementRegion, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	region, err := api.NewExpressionEmission(ordered.Before(), context.Factory().CallExpression(runtime.Expression(context.Factory()), nil,
		[]tsgo.TypeNode{storage.Value()}, values[:2], tsgo.NodeFlagsNone),
		api.CombineRequests(ordered.Requests(), storage.Requests(), runtime.Requests()))
	return region, api.DirectExpression(values[2]), err
}
