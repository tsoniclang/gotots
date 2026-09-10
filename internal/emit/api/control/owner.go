package control

import (
	"go/ast"
	"go/types"
)

type Owner interface {
	Valid() bool
	Source() (types.Object, bool)
	PackageInitializer() (*types.Package, *types.Initializer, bool)
}

func ValidIteratorRange(
	callable ast.Node,
	source *ast.RangeStmt,
) bool {
	return callable != nil &&
		source != nil &&
		source.X != nil &&
		source.Body != nil &&
		source.Pos() >= callable.Pos() &&
		source.End() <= callable.End()
}

func ValidDefer(
	callable ast.Node,
	source *ast.DeferStmt,
) bool {
	return callable != nil &&
		source != nil &&
		source.Call != nil &&
		source.Pos().IsValid() &&
		source.End() >= source.Pos() &&
		source.Pos() >= callable.Pos() &&
		source.End() <= callable.End()
}

func ValidAnchor(
	owner Owner,
	enclosing ast.Node,
	callable ast.Node,
) bool {
	if !owner.Valid() ||
		enclosing == nil ||
		callable == nil ||
		callable.Pos() < enclosing.Pos() ||
		callable.End() > enclosing.End() {
		return false
	}
	switch callable := callable.(type) {
	case *ast.FuncDecl:
		source, ok := owner.Source()
		function, functionOK := source.(*types.Func)
		return ok &&
			functionOK &&
			enclosing == callable &&
			callable.Type != nil &&
			callable.Body != nil &&
			function.Pos() >= callable.Pos() &&
			function.Pos() <= callable.End()
	case *ast.FuncLit:
		if callable.Type == nil || callable.Body == nil {
			return false
		}
		if source, ok := owner.Source(); ok {
			function, functionOK := source.(*types.Func)
			return functionOK &&
				function.Pos() >= enclosing.Pos() &&
				function.Pos() <= enclosing.End()
		}
		_, initializer, ok := owner.PackageInitializer()
		return ok &&
			initializer.Rhs != nil &&
			enclosing == initializer.Rhs
	default:
		return false
	}
}

func ValidOwner(
	owner Owner,
	enclosing ast.Node,
	callable ast.Node,
) bool {
	if enclosing != nil || callable != nil {
		return ValidAnchor(owner, enclosing, callable)
	}
	source, ok := owner.Source()
	function, functionOK := source.(*types.Func)
	return ok && functionOK && function.Origin() == function
}

func ValidIndirectVariable(owner Owner, enclosing ast.Node, variable *types.Var) bool {
	if enclosing == nil || variable == nil || variable.IsField() || variable.Parent() == nil ||
		variable.Pos() < enclosing.Pos() || variable.Pos() >= enclosing.End() || variable.Pkg() == nil ||
		variable.Parent() == variable.Pkg().Scope() {
		return false
	}
	if source, ok := owner.Source(); ok {
		return source.Pkg() == variable.Pkg()
	}
	sourcePackage, _, ok := owner.PackageInitializer()
	return ok && sourcePackage == variable.Pkg()
}
