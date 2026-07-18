package ir

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// BuildPackageVarInit builds one package-level variable initializer (or
// the zero value when e is nil). Every variable declares with its zero
// value first; initializers then run in the type checker's exact
// dependency order (types.Info.InitOrder) as module-level assignments,
// so calls and variable reads observe exactly Go's initialization
// sequence.
func BuildPackageVarInit(p *packages.Package, sourceDir string, unit Scope, e ast.Expr, expected Type, pos token.Pos) (Expr, error) {
	b := &builder{
		fset:       p.Fset,
		info:       p.TypesInfo,
		pkgPath:    p.PkgPath,
		sourceDir:  sourceDir,
		unit:       unit,
		operations: map[string]bool{},
		sites:      &[]UnsupportedSite{},
	}
	span := b.span(pos)
	if e == nil {
		return zeroValue(expected, span)
	}
	// A package-level initializer may itself contain function literals with
	// their own locals; assign canonical binding identities over the whole
	// initializer so those bindings draw allocated names too.
	b.assignBindings(e, nil)
	return b.buildExprAs(e, expected)
}
