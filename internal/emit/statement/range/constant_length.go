package rangestatement

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func constantLengthRangeExpression(
	context api.Context,
	source ast.Expr,
) (bool, error) {
	switch source := source.(type) {
	case *ast.Ident, *ast.BasicLit, *ast.FuncLit:
		return true, nil
	case *ast.CompositeLit:
		return constantLengthRangeExpressions(context, source.Elts)
	case *ast.ParenExpr:
		return constantLengthRangeExpression(context, source.X)
	case *ast.SelectorExpr:
		return constantLengthRangeExpression(context, source.X)
	case *ast.IndexExpr:
		return constantLengthRangeExpressions(
			context,
			[]ast.Expr{source.X, source.Index},
		)
	case *ast.IndexListExpr:
		expressions := make([]ast.Expr, 0, len(source.Indices)+1)
		expressions = append(expressions, source.X)
		expressions = append(expressions, source.Indices...)
		return constantLengthRangeExpressions(context, expressions)
	case *ast.SliceExpr:
		return constantLengthRangeExpressions(
			context,
			[]ast.Expr{source.X, source.Low, source.High, source.Max},
		)
	case *ast.TypeAssertExpr:
		return constantLengthRangeExpression(context, source.X)
	case *ast.CallExpr:
		facts, ok := context.TypesInfo().Types[source.Fun]
		if !ok || !facts.IsType() {
			return false, nil
		}
		return constantLengthRangeExpressions(context, source.Args)
	case *ast.StarExpr:
		return constantLengthRangeExpression(context, source.X)
	case *ast.UnaryExpr:
		if source.Op == token.ARROW {
			return false, nil
		}
		return constantLengthRangeExpression(context, source.X)
	case *ast.BinaryExpr:
		return constantLengthRangeExpressions(
			context,
			[]ast.Expr{source.X, source.Y},
		)
	case *ast.KeyValueExpr:
		return constantLengthRangeExpressions(
			context,
			[]ast.Expr{source.Key, source.Value},
		)
	case *ast.Ellipsis, *ast.ArrayType, *ast.StructType, *ast.FuncType,
		*ast.InterfaceType, *ast.MapType, *ast.ChanType:
		return true, nil
	case *ast.BadExpr:
		return false, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	default:
		return false, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
}

func constantLengthRangeExpressions(
	context api.Context,
	expressions []ast.Expr,
) (bool, error) {
	for _, expression := range expressions {
		if expression == nil {
			continue
		}
		constant, err := constantLengthRangeExpression(context, expression)
		if err != nil || !constant {
			return constant, err
		}
	}
	return true, nil
}
