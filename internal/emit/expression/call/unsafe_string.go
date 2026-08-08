package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitUnsafeString(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	signature, ok := context.TypesInfo().TypeOf(source.Fun).(*types.Signature)
	if !ok ||
		signature == nil ||
		signature.Params() == nil ||
		signature.Params().Len() != 2 ||
		len(source.Args) != 2 ||
		source.Ellipsis != token.NoPos {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	address, ok := ast.Unparen(source.Args[0]).(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	indexed, ok := ast.Unparen(address.X).(*ast.IndexExpr)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	pointer, ok := types.Unalias(context.TypesInfo().TypeOf(source.Args[0])).(*types.Pointer)
	sourceType := context.TypesInfo().TypeOf(indexed.X)
	_, element, sliceOK := slicevalue.Source(sourceType)
	if !ok ||
		!sliceOK ||
		!types.Identical(pointer.Elem(), types.Typ[types.Uint8]) ||
		!types.Identical(element, types.Typ[types.Uint8]) ||
		!types.Identical(context.TypesInfo().TypeOf(indexed), element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(sourceType),
		indexed.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, err = slicevalue.Project(context, sourceType, receiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index, err := integeroperand.Emit(
		context.WithRole(api.RoleCallArgument),
		children,
		indexed.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	length, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(signature.Params().At(1).Type()),
		source.Args[1],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporaryCallArgument,
		expressionoperands.Present(receiver),
		expressionoperands.Present(index),
		expressionoperands.Present(length),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storage, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeUnsafeString,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			[]tsgo.TypeNode{storage.Value()},
			ordered.Values(),
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			ordered.Requests(),
			storage.Requests(),
			runtime.Requests(),
		),
	)
}
