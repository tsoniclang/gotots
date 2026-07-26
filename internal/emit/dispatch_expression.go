package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	binaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/binary"
	callexpression "github.com/tsoniclang/gotots/internal/emit/expression/call"
	compositeliteral "github.com/tsoniclang/gotots/internal/emit/expression/compositeliteral"
	identifierexpression "github.com/tsoniclang/gotots/internal/emit/expression/identifier"
	integerliteral "github.com/tsoniclang/gotots/internal/emit/expression/literal/integer"
	parenthesizedexpression "github.com/tsoniclang/gotots/internal/emit/expression/parenthesized"
	selectorexpression "github.com/tsoniclang/gotots/internal/emit/expression/selector"
	unaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/unary"
)

func (e *emitter) Expression(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	switch source := source.(type) {
	case *ast.BinaryExpr:
		return binaryexpression.Emit(context, e, source)
	case *ast.CallExpr:
		return callexpression.Emit(context, e, source)
	case *ast.CompositeLit:
		return compositeliteral.Emit(context, e, source)
	case *ast.Ident:
		return identifierexpression.Emit(context, e, source)
	case *ast.ParenExpr:
		return parenthesizedexpression.Emit(context, e, source)
	case *ast.SelectorExpr:
		return selectorexpression.Emit(context, e, source)
	case *ast.BasicLit:
		return e.IntegerConstant(context, source)
	case *ast.UnaryExpr:
		return unaryexpression.Emit(context, e, source)
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
}

func (e *emitter) IntegerConstant(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	return integerliteral.Emit(context, e, source)
}

func (e *emitter) DiscardedCall(
	context api.Context,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	return callexpression.EmitDiscarded(context, e, source)
}

func (e *emitter) Condition(
	context api.Context,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok || basic.Info()&types.IsBoolean == 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return e.Expression(context.WithExpectedType(types.Typ[types.Bool]), source)
}
