package emit

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	binaryexpression "github.com/tsoniclang/gotots/internal/emit/expression/binary"
	identifierexpression "github.com/tsoniclang/gotots/internal/emit/expression/identifier"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e *Emitter) Expression(context api.Context, source ast.Expr) (tsgo.Expression, error) {
	switch source := source.(type) {
	case *ast.BinaryExpr:
		return binaryexpression.Emit(context, e, source)
	case *ast.Ident:
		return identifierexpression.Emit(context, source)
	default:
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
}
