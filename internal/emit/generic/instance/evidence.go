package instance

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func FunctionEvidence(
	info api.TypeInfoView,
	source ast.Expr,
) (*types.Func, api.TypeInstance, bool) {
	if !info.Valid() || source == nil {
		return nil, api.TypeInstance{}, false
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
		if info.SelectionOf(selected) != nil {
			return nil, api.TypeInstance{}, false
		}
		identifier = selected.Sel
	default:
		return nil, api.TypeInstance{}, false
	}
	instance, instantiated := info.InstanceOf(identifier)
	owner, function := info.UseOf(identifier).(*types.Func)
	if !instantiated || !function || owner.Origin() == nil {
		return nil, api.TypeInstance{}, false
	}
	return owner.Origin(), instance, true
}
