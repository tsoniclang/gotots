package instance

import (
	"go/ast"
	"go/types"
)

func FunctionEvidence(
	info *types.Info,
	source ast.Expr,
) (*types.Func, types.Instance, bool) {
	if info == nil || source == nil {
		return nil, types.Instance{}, false
	}
	base := source
	switch selected := source.(type) {
	case *ast.IndexExpr:
		base = selected.X
	case *ast.IndexListExpr:
		base = selected.X
	}
	var identifier *ast.Ident
	switch selected := base.(type) {
	case *ast.Ident:
		identifier = selected
	case *ast.SelectorExpr:
		if info.Selections[selected] != nil {
			return nil, types.Instance{}, false
		}
		identifier = selected.Sel
	default:
		return nil, types.Instance{}, false
	}
	instance, instantiated := info.Instances[identifier]
	owner, function := info.Uses[identifier].(*types.Func)
	if !instantiated || !function || owner.Origin() == nil {
		return nil, types.Instance{}, false
	}
	return owner.Origin(), instance, true
}
