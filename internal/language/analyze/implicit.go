// Implicit-operation detection: the inventory-owned members of the implicit
// operation catalog are detected from syntax plus go/types evidence and
// attached to the occurrences that carry them. Semantic-model-owned members
// are produced by the later phase; the catalog records that ownership.
package analyze

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// detectImplicit attaches inventory-owned implicit operations from typed
// evidence plus the parent-assigned context recorded during traversal.
func detectImplicit(b *builder) {
	info := b.info
	for i, n := range b.nodes {
		ctx := b.contexts[i]
		switch n := n.(type) {
		case *ast.ValueSpec:
			// Zeroing: a var binding without initializer takes the type's
			// zero value implicitly (the parent GenDecl supplied var context).
			if ctx.varDecl && len(n.Values) == 0 {
				b.attach(i, catalog.ImplicitZeroing)
			}
			// Typed var initializers cross the declared-type boundary.
			if n.Type != nil && len(n.Values) > 0 {
				target := typeOf(info, n.Type)
				for _, value := range n.Values {
					b.markBoundary(target, value, info)
				}
			}
		case *ast.SelectorExpr:
			if selection, ok := info.SelectionOf(n); ok {
				if len(selection.Index()) > 1 {
					b.attach(i, catalog.ImplicitMethodPromotion)
				}
				// Receiver adjustment: the declared receiver's pointerness
				// differs from the selected operand's — Go inserts the
				// address-of or dereference implicitly.
				if selection.Kind() == types.MethodVal {
					if fn, ok := selection.Obj().(*types.Func); ok {
						if recv := fn.Type().(*types.Signature).Recv(); recv != nil {
							_, declaredPtr := types.Unalias(recv.Type()).Underlying().(*types.Pointer)
							_, actualPtr := types.Unalias(selection.Recv()).Underlying().(*types.Pointer)
							if declaredPtr != actualPtr || selection.Indirect() {
								b.attach(i, catalog.ImplicitReceiverAdjustment)
							}
						}
					}
				}
			}
		case *ast.AssignStmt:
			if len(n.Lhs) == len(n.Rhs) {
				for k := range n.Rhs {
					b.markBoundary(typeOf(info, n.Lhs[k]), n.Rhs[k], info)
				}
			}
		case *ast.CallExpr:
			b.markCallBoundaries(n, info)
		}
	}
}

// markCallBoundaries attaches conversion/copy evidence to call arguments
// crossing into parameter types. Conversions and builtin calls are excluded —
// their operand semantics belong to their own variants.
func (b *builder) markCallBoundaries(n *ast.CallExpr, info *source.TypeInfoView) {
	tv, ok := info.TypeOf(n.Fun)
	if !ok || tv.IsType() {
		return
	}
	signature, ok := types.Unalias(tv.Type).Underlying().(*types.Signature)
	if !ok {
		return
	}
	params := signature.Params()
	for i, arg := range n.Args {
		var target types.Type
		switch {
		case signature.Variadic() && i >= params.Len()-1:
			if n.Ellipsis.IsValid() {
				if i == params.Len()-1 {
					target = params.At(params.Len() - 1).Type()
				}
			} else if params.Len() > 0 {
				if slice, ok := types.Unalias(params.At(params.Len() - 1).Type()).Underlying().(*types.Slice); ok {
					target = slice.Elem()
				}
			}
		case i < params.Len():
			target = params.At(i).Type()
		}
		if target != nil {
			b.markBoundary(target, arg, info)
		}
	}
}

// markBoundary attaches interface-conversion or value-copy evidence to the
// operand occurrence when the typed boundary requires it.
func (b *builder) markBoundary(target types.Type, operand ast.Expr, info *source.TypeInfoView) {
	if target == nil {
		return
	}
	operandTV, ok := info.TypeOf(operand)
	if !ok || operandTV.Type == nil {
		return
	}
	idx := b.indexOf(operand)
	if idx < 0 {
		return
	}
	if types.IsInterface(target) {
		if !types.IsInterface(operandTV.Type) {
			b.attach(idx, catalog.ImplicitInterfaceConversion)
		}
		return
	}
	switch types.Unalias(operandTV.Type).Underlying().(type) {
	case *types.Struct, *types.Array:
		b.attach(idx, catalog.ImplicitValueCopy)
	}
}

// attach appends one implicit operation to the occurrence at index i,
// deduplicating repeats.
func (b *builder) attach(i int, op catalog.ImplicitOp) {
	for _, existing := range b.occurrences[i].implicit {
		if existing == op {
			return
		}
	}
	b.occurrences[i].implicit = append(b.occurrences[i].implicit, op)
}

// indexOf finds the occurrence index of one syntax node; -1 when the node is
// not inventoried (never expected for expression operands).
func (b *builder) indexOf(n ast.Node) int {
	if i, ok := b.index[n]; ok {
		return i
	}
	return -1
}

// typeOf is the type evidence of one expression, nil for the blank identifier
// and untyped targets.
func typeOf(info *source.TypeInfoView, x ast.Expr) types.Type {
	if tv, ok := info.TypeOf(x); ok {
		return tv.Type
	}
	return nil
}
