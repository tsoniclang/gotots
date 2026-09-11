package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitUnsafeViewRegion(context api.Context, children api.ChildEmitter, source ast.Expr, pointer *types.Pointer) (api.ExpressionEmission, error) {
	if address, ok := ast.Unparen(source).(*ast.UnaryExpr); ok && address.Op == token.AND {
		if indexed, ok := ast.Unparen(address.X).(*ast.IndexExpr); ok {
			sourceType := context.TypesInfo().TypeOf(indexed.X)
			if _, element, ok := slicevalue.Source(sourceType); ok && types.Identical(element, pointer.Elem()) {
				return emitSliceElementRegion(context, children, indexed, sourceType, element)
			}
		}
	}
	argument, err := children.Expression(context.WithRole(api.RoleCallArgument).WithExpectedType(pointer), source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return memorymarker.PointerRegion(context, children, source, pointer.Elem(), argument)
}

func emitSliceElementRegion(context api.Context, children api.ChildEmitter, source *ast.IndexExpr, sourceType, element types.Type) (api.ExpressionEmission, error) {
	receiver, err := children.Expression(context.WithRole(api.RoleCallArgument).WithExpectedType(sourceType), source.X)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, err = slicevalue.Project(context, sourceType, receiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index, err := integeroperand.Emit(context.WithRole(api.RoleCallArgument), children, source.Index)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	ordered, err := expressionoperands.Preserve(context, api.TemporaryAddressOperand,
		expressionoperands.Present(receiver), expressionoperands.Present(index))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storage, err := context.ContainerStorage().ContainerStorageType(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(api.RuntimeSliceElementRegion, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(ordered.Before(), context.Factory().CallExpression(runtime.Expression(context.Factory()), nil,
		[]tsgo.TypeNode{storage.Value()}, ordered.Values(), tsgo.NodeFlagsNone),
		api.CombineRequests(ordered.Requests(), storage.Requests(), runtime.Requests()))
}
