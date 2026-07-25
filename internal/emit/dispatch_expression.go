package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	binaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/binary"
	callexpression "github.com/tsoniclang/gotots/internal/emit/expression/call"
	identifierexpression "github.com/tsoniclang/gotots/internal/emit/expression/identifier"
	integerliteral "github.com/tsoniclang/gotots/internal/emit/expression/literal/integer"
	unaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/unary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e *Emitter) Expression(context api.Context, source ast.Expr) (tsgo.Expression, error) {
	switch source := source.(type) {
	case *ast.BinaryExpr:
		return binaryexpression.Emit(context, e, source)
	case *ast.CallExpr:
		return callexpression.Emit(context, e, source)
	case *ast.Ident:
		return identifierexpression.Emit(context, source)
	case *ast.BasicLit:
		return e.IntegerConstant(context, source)
	case *ast.UnaryExpr:
		return unaryexpression.Emit(context, e, source)
	default:
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
}

func (e *Emitter) IntegerConstant(
	context api.Context,
	source ast.Expr,
) (tsgo.Expression, error) {
	return integerliteral.Emit(context, e, source)
}

func (e *Emitter) Condition(context api.Context, source ast.Expr) (tsgo.Expression, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok || basic.Info()&types.IsBoolean == 0 {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	return e.Expression(context.WithExpectedType(types.Typ[types.Bool]), source)
}
