package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	unsafeoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin/unsafeoperation"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	runtimestring "github.com/tsoniclang/gotots/internal/emit/runtime/stringvalue"
	"github.com/tsoniclang/gotots/internal/emit/stringvalue"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
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
