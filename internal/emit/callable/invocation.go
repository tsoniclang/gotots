package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func StaticallyNonNil(info api.TypeInfoView, source ast.Expr) bool {
	switch source := source.(type) {
	case *ast.FuncLit:
		return true
	case *ast.ParenExpr:
		return StaticallyNonNil(info, source.X)
	case *ast.Ident:
		if !info.Valid() {
			return false
		}
		_, ok := info.UseOf(source).(*types.Func)
		return ok
	case *ast.SelectorExpr:
		if !info.Valid() || info.SelectionOf(source) != nil {
			return false
		}
		qualifier, ok := source.X.(*ast.Ident)
		if !ok {
			return false
		}
		packageName, ok := info.UseOf(qualifier).(*types.PkgName)
		if !ok {
			return false
		}
		function, ok := info.UseOf(source.Sel).(*types.Func)
		return ok && function.Pkg() == packageName.Imported()
	default:
		return false
	}
}

func NilGuard(
	context api.Context,
	callee tsgo.Expression,
) (tsgo.Statement, []api.RootRequest, error) {
	reference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	condition := context.Factory().BinaryExpression(
		nil,
		callee,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		context.Factory().VoidExpression(
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		),
	)
	raise := panicruntime.Call(
		context.Factory(),
		reference.Name(),
		context.Factory().StringLiteral(
			"call of nil function",
			tsgo.TokenFlagsNone,
		),
	)
	return context.Factory().IfStatement(
		condition,
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ExpressionStatement(raise),
			},
			true,
		),
		nil,
	), reference.Requests(), nil
}

func DetachedNilGuard(
	context api.Context,
	callee tsgo.Expression,
	nonNil tsgo.Expression,
) (tsgo.Expression, []api.RootRequest, error) {
	reference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	condition := context.Factory().BinaryExpression(
		nil,
		callee,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		context.Factory().VoidExpression(
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		),
	)
	raise := panicruntime.Call(
		context.Factory(),
		reference.Name(),
		context.Factory().StringLiteral(
			"call of nil function",
			tsgo.TokenFlagsNone,
		),
	)
	return context.Factory().ParenthesizedExpression(
		context.Factory().ConditionalExpression(
			condition,
			context.Factory().QuestionToken(),
			raise,
			context.Factory().ColonToken(),
			nonNil,
		),
	), reference.Requests(), nil
}
