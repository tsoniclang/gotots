package source

import (
	"go/ast"
	"go/types"
)

// TypeInfoView is the finalized read-only type-information API. Downstream
// consumers query it; the underlying maps are never exposed, so no consumer
// can mutate — or iterate beyond — the retained evidence. It covers every
// go/types.Info fact the semantic model needs — types, definitions, uses,
// selections, generic instances, implicits, scopes, initialization ordering,
// and per-file language versions — so the next phase never re-enters the
// checker or keeps an alternate evidence store.
type TypeInfoView struct {
	info *types.Info
}

// newTypeInfoView wraps retained (already depth-filtered) information.
func newTypeInfoView(info *types.Info) *TypeInfoView {
	if info == nil {
		return nil
	}
	return &TypeInfoView{info: info}
}

// TypeOf returns the type-and-value evidence of one expression.
func (v *TypeInfoView) TypeOf(expr ast.Expr) (types.TypeAndValue, bool) {
	tv, ok := v.info.Types[expr]
	return tv, ok
}

// UseOf returns the object one identifier uses.
func (v *TypeInfoView) UseOf(ident *ast.Ident) (types.Object, bool) {
	obj, ok := v.info.Uses[ident]
	return obj, ok
}

// DefOf returns the object one identifier defines.
func (v *TypeInfoView) DefOf(ident *ast.Ident) (types.Object, bool) {
	obj, ok := v.info.Defs[ident]
	return obj, ok
}

// SelectionOf returns the selection evidence of one selector.
func (v *TypeInfoView) SelectionOf(sel *ast.SelectorExpr) (*types.Selection, bool) {
	selection, ok := v.info.Selections[sel]
	return selection, ok && selection != nil
}

// InstanceOf reports whether one identifier is a generic instantiation.
func (v *TypeInfoView) InstanceOf(ident *ast.Ident) (types.Instance, bool) {
	instance, ok := v.info.Instances[ident]
	return instance, ok
}

// ImplicitOf returns the object one node implicitly declares (an unnamed
// import, a type-switch case binding, or a composite-literal element).
func (v *TypeInfoView) ImplicitOf(node ast.Node) (types.Object, bool) {
	obj, ok := v.info.Implicits[node]
	return obj, ok
}

// ScopeOf returns the lexical scope one node introduces.
func (v *TypeInfoView) ScopeOf(node ast.Node) (*types.Scope, bool) {
	scope, ok := v.info.Scopes[node]
	return scope, ok && scope != nil
}

// FileVersionOf returns one file's effective language version.
func (v *TypeInfoView) FileVersionOf(file *ast.File) (string, bool) {
	version, ok := v.info.FileVersions[file]
	return version, ok
}

// InitEntry is one package-level initialization step: the variables it assigns
// and the initializing expression, in dependency order.
type InitEntry struct {
	Vars []*types.Var
	Rhs  ast.Expr
}

// InitOrder returns the package's variable-initialization order (a fresh slice;
// the retained ordering is never exposed for mutation).
func (v *TypeInfoView) InitOrder() []InitEntry {
	out := make([]InitEntry, 0, len(v.info.InitOrder))
	for _, initializer := range v.info.InitOrder {
		out = append(out, InitEntry{
			Vars: append([]*types.Var(nil), initializer.Lhs...),
			Rhs:  initializer.Rhs,
		})
	}
	return out
}
