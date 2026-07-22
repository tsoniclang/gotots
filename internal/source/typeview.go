package source

import (
	"go/ast"
	"go/types"
)

// TypeInfoView is the finalized read-only type-information API. Downstream
// consumers query it; the underlying maps are never exposed, so no consumer
// can mutate — or iterate beyond — the retained evidence.
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
