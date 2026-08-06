package callableabi

import (
	"fmt"
	"go/ast"
	"go/types"
)

func SourceCallableIdentity(function *types.Func) (string, error) {
	if function == nil || function.Pkg() == nil || function.Name() == "" {
		return "", &Error{Reason: "source callable identity is incomplete"}
	}
	signature := function.Signature()
	if signature.Recv() == nil {
		return PackageFunctionIdentity(function.Pkg().Path(), function.Name())
	}
	receiver := types.TypeString(signature.Recv().Type(), func(
		selected *types.Package,
	) string {
		if selected == nil {
			return ""
		}
		return selected.Path()
	})
	if receiver == "" {
		return "", &Error{
			Function: function.FullName(),
			Reason:   "method receiver identity is absent",
		}
	}
	return fmt.Sprintf(
		"method\x00%s\x00%s\x00%s",
		function.Pkg().Path(),
		receiver,
		function.Name(),
	), nil
}

func PointeeValueReadAtEntry(
	declaration *ast.FuncDecl,
	parameter *types.Var,
	info *types.Info,
) bool {
	if declaration == nil || declaration.Body == nil || parameter == nil ||
		info == nil || len(declaration.Body.List) != 1 {
		return false
	}
	result, ok := declaration.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return false
	}
	expression := unparenthesized(result.Results[0])
	dereference, ok := expression.(*ast.StarExpr)
	if !ok {
		return false
	}
	identifier, ok := unparenthesized(dereference.X).(*ast.Ident)
	return ok && info.Uses[identifier] == parameter
}

func unparenthesized(source ast.Expr) ast.Expr {
	for {
		parenthesized, ok := source.(*ast.ParenExpr)
		if !ok {
			return source
		}
		source = parenthesized.X
	}
}
