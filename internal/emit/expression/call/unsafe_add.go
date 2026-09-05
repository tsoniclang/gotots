package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/expression/operands"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	rawpointermarker "github.com/tsoniclang/gotots/internal/emit/marker/rawpointer"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
)

func emitUnsafeAdd(context api.Context, children api.ChildEmitter, source *ast.CallExpr) (api.ExpressionEmission, error) {
	if len(source.Args) != 2 || source.Ellipsis != token.NoPos ||
		!types.Identical(context.TypesInfo().TypeOf(source), types.Typ[types.UnsafePointer]) {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	pointer, err := children.Expression(context.WithRole(api.RoleBuiltinArgument).
		WithExpectedType(types.Typ[types.UnsafePointer]), source.Args[0])
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	offset, err := integeroperand.Emit(context.WithRole(api.RoleBuiltinArgument), children, source.Args[1])
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	ordered, err := operands.Preserve(context, api.TemporaryCallArgument, operands.Present(pointer), operands.Present(offset))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	abi, err := memorymarker.DataLayout(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	operation, err := rawpointermarker.Operation(context, tsoniccore.SymbolOffsetRawPointer,
		api.DirectExpression(values[0]), api.DirectExpression(values[1]), abi)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(ordered.Before(), operation.Value(),
		api.CombineRequests(ordered.Requests(), operation.Requests()))
}
