package control

import (
	"go/ast"
	"go/token"
	"go/types"
)

func ExposedVariable(expression ast.Expr, info *types.Info) *types.Var {
	if expression == nil || info == nil {
		return nil
	}
	if unary, ok := ast.Unparen(expression).(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expression = unary.X
	}
	variable := rootVariable(expression, info)
	if variable == nil || variable.IsField() || variable.Parent() == nil ||
		variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope() {
		return nil
	}
	basic, scalar := variable.Type().Underlying().(*types.Basic)
	if !scalar || basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsString) == 0 {
		return nil
	}
	return variable
}

func IndirectlyMutable(expression ast.Expr, info *types.Info, writes map[*types.Var]struct{}) bool {
	if info == nil || expression == nil {
		return false
	}
	sourceType := info.TypeOf(expression)
	if sourceType == nil {
		return false
	}
	basic, scalar := sourceType.Underlying().(*types.Basic)
	if !scalar || basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsString) == 0 {
		return false
	}
	switch expression := expression.(type) {
	case *ast.ParenExpr:
		return IndirectlyMutable(expression.X, info, writes)
	case *ast.UnaryExpr:
		return expression.Op == token.NOT && IndirectlyMutable(expression.X, info, writes)
	case *ast.StarExpr, *ast.IndexExpr:
		return true
	case *ast.SelectorExpr:
		selected := info.Selections[expression]
		if selected != nil && selected.Kind() == types.FieldVal {
			return true
		}
	}
	variable := rootVariable(expression, info)
	if variable == nil {
		return false
	}
	if variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope() {
		return true
	}
	_, exposed := writes[variable]
	return exposed
}

func rootVariable(expression ast.Expr, info *types.Info) *types.Var {
	switch expression := expression.(type) {
	case *ast.ParenExpr:
		return rootVariable(expression.X, info)
	case *ast.Ident:
		variable, _ := info.ObjectOf(expression).(*types.Var)
		return variable
	case *ast.SelectorExpr:
		if info.Selections[expression] != nil {
			return rootVariable(expression.X, info)
		}
		variable, _ := info.ObjectOf(expression.Sel).(*types.Var)
		return variable
	case *ast.IndexExpr:
		return rootVariable(expression.X, info)
	default:
		return nil
	}
}
