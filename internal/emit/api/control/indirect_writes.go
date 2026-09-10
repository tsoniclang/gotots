package control

import (
	"go/ast"
	"go/token"
	"go/types"
)

func IndirectWrites(source ast.Node, info *types.Info) map[*types.Var]struct{} {
	writes := make(map[*types.Var]struct{})
	if source != nil && info != nil {
		ast.Walk(indirectWriteVisitor{info: info, writes: writes}, source)
	}
	return writes
}

type indirectWriteVisitor struct {
	info    *types.Info
	writes  map[*types.Var]struct{}
	closure *ast.FuncLit
}

func (visitor indirectWriteVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case nil:
		return nil
	case *ast.FuncLit:
		visitor.closure = node
	case *ast.UnaryExpr:
		if node.Op == token.AND {
			visitor.record(node.X, false)
		}
	case *ast.AssignStmt:
		for _, target := range node.Lhs {
			visitor.record(target, true)
		}
	case *ast.IncDecStmt:
		visitor.record(node.X, true)
	case *ast.SelectorExpr:
		if selected := visitor.info.Selections[node]; selected != nil && selected.Kind() == types.MethodVal {
			if signature, ok := selected.Obj().Type().(*types.Signature); ok && signature.Recv() != nil {
				if _, pointer := signature.Recv().Type().Underlying().(*types.Pointer); pointer {
					visitor.record(node.X, false)
				}
			}
		}
	}
	return visitor
}

func (visitor indirectWriteVisitor) record(expression ast.Expr, captured bool) {
	if captured && visitor.closure == nil {
		return
	}
	variable := rootVariable(expression, visitor.info)
	if variable == nil || captured && variable.Pos() >= visitor.closure.Pos() && variable.Pos() < visitor.closure.End() {
		return
	}
	visitor.writes[variable] = struct{}{}
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
