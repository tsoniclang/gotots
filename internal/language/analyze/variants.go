// Semantic-variant resolution: each variant-bearing occurrence is resolved
// from typed go/types evidence plus its parent edge into exactly one closed
// catalog variant. Unresolvable variant-bearing occurrences fail closed.
package analyze

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// resolveVariants assigns the semantic variant of every occurrence.
func resolveVariants(b *builder, info *types.Info) error {
	for i, n := range b.nodes {
		variant, err := b.variantOf(i, n, info)
		if err != nil {
			return err
		}
		b.occurrences[i].variant = variant
	}
	return nil
}

// variantOf resolves one occurrence. Kinds without a variant axis resolve to
// VariantNone.
func (b *builder) variantOf(i int, n ast.Node, info *types.Info) (catalog.Variant, error) {
	unresolved := func(reason string) (catalog.Variant, error) {
		return catalog.VariantInvalid, newResolutionError(b.occurrences[i].kind, b.file, b.occurrences[i].span, reason)
	}
	switch n := n.(type) {
	case *ast.CallExpr:
		if tv, ok := info.Types[n.Fun]; ok && tv.IsType() {
			return catalog.VariantConversion, nil
		}
		if callee := calleeIdent(n.Fun); callee != nil {
			if _, isBuiltin := info.Uses[callee].(*types.Builtin); isBuiltin {
				return catalog.VariantCallBuiltin, nil
			}
		}
		if sel, ok := ast.Unparen(n.Fun).(*ast.SelectorExpr); ok {
			if selection, ok := info.Selections[sel]; ok && selection.Kind() == types.MethodVal {
				return catalog.VariantCallMethod, nil
			}
		}
		return catalog.VariantCallFunction, nil
	case *ast.IndexExpr:
		return b.indexVariant(i, n.X, n, info)
	case *ast.IndexListExpr:
		return catalog.VariantGenericInstantiation, nil
	case *ast.TypeAssertExpr:
		if n.Type == nil {
			return catalog.VariantTypeSwitchGuard, nil
		}
		if b.commaOkContext(i) {
			return catalog.VariantAssertCommaOk, nil
		}
		return catalog.VariantAssertValue, nil
	case *ast.SelectorExpr:
		if selection, ok := info.Selections[n]; ok {
			switch selection.Kind() {
			case types.MethodVal:
				return catalog.VariantSelectMethodValue, nil
			case types.MethodExpr:
				return catalog.VariantSelectMethodExpression, nil
			case types.FieldVal:
				if len(selection.Index()) > 1 {
					return catalog.VariantSelectPromotedField, nil
				}
				return catalog.VariantSelectField, nil
			}
		}
		if base, ok := ast.Unparen(n.X).(*ast.Ident); ok {
			if _, isPkg := info.Uses[base].(*types.PkgName); isPkg {
				return catalog.VariantSelectPackageMember, nil
			}
		}
		// Qualified type references and method expressions on named types
		// resolve through Uses of the selected identifier.
		if info.Uses[n.Sel] != nil || info.Defs[n.Sel] != nil {
			return catalog.VariantSelectPackageMember, nil
		}
		return unresolved("selector has no selection, package, or use evidence")
	case *ast.AssignStmt:
		return b.assignVariant(n), nil
	case *ast.CompositeLit:
		tv, ok := info.Types[n]
		if !ok {
			return unresolved("composite literal has no type evidence")
		}
		switch aggregateShape(tv.Type).(type) {
		case *types.Struct:
			return catalog.VariantLitStruct, nil
		case *types.Array:
			return catalog.VariantLitArray, nil
		case *types.Slice:
			return catalog.VariantLitSlice, nil
		case *types.Map:
			return catalog.VariantLitMap, nil
		}
		return unresolved("composite literal over unsupported underlying type")
	case *ast.KeyValueExpr:
		lit, ok := b.parentNode(i).(*ast.CompositeLit)
		if !ok {
			return unresolved("key-value element outside a composite literal")
		}
		tv, ok := info.Types[lit]
		if !ok {
			return unresolved("enclosing literal has no type evidence")
		}
		switch aggregateShape(tv.Type).(type) {
		case *types.Struct:
			return catalog.VariantKeyFieldName, nil
		case *types.Map:
			return catalog.VariantKeyMapKey, nil
		case *types.Array, *types.Slice:
			return catalog.VariantKeyArrayIndex, nil
		}
		return unresolved("keyed element in unsupported literal")
	case *ast.UnaryExpr:
		if n.Op != token.ARROW {
			return catalog.VariantNone, nil
		}
		if b.commaOkContext(i) {
			return catalog.VariantReceiveCommaOk, nil
		}
		return catalog.VariantReceiveValue, nil
	case *ast.StarExpr:
		if tv, ok := info.Types[n]; ok && tv.IsType() {
			return catalog.VariantStarPointerType, nil
		}
		return catalog.VariantStarDereference, nil
	case *ast.ReturnStmt:
		if len(n.Results) > 0 {
			return catalog.VariantReturnValues, nil
		}
		if fn := b.enclosingFunc(i); fn != nil && fn.Results != nil && fn.Results.NumFields() > 0 {
			return catalog.VariantReturnBare, nil
		}
		return catalog.VariantReturnVoid, nil
	case *ast.RangeStmt:
		return b.rangeVariant(i, n, info)
	case *ast.SwitchStmt:
		if n.Tag != nil {
			return catalog.VariantSwitchExpression, nil
		}
		return catalog.VariantSwitchTrue, nil
	case *ast.TypeSpec:
		if n.Assign.IsValid() {
			return catalog.VariantTypeAlias, nil
		}
		return catalog.VariantTypeDefinition, nil
	case *ast.CommClause:
		switch comm := n.Comm.(type) {
		case nil:
			return catalog.VariantCommDefault, nil
		case *ast.SendStmt:
			return catalog.VariantCommSend, nil
		default:
			_ = comm
			return catalog.VariantCommReceive, nil
		}
	}
	return catalog.VariantNone, nil
}

// indexVariant distinguishes generic instantiation, map lookups, and element
// indexing for one IndexExpr.
func (b *builder) indexVariant(i int, x ast.Expr, n *ast.IndexExpr, info *types.Info) (catalog.Variant, error) {
	if tv, ok := info.Types[n]; ok && tv.IsType() {
		return catalog.VariantGenericInstantiation, nil
	}
	if callee := calleeIdent(x); callee != nil {
		if _, instantiated := info.Instances[callee]; instantiated {
			return catalog.VariantGenericInstantiation, nil
		}
	}
	tv, ok := info.Types[x]
	if !ok {
		return catalog.VariantInvalid, newResolutionError(b.occurrences[i].kind, b.file, b.occurrences[i].span,
			"index operand has no type evidence")
	}
	if _, isMap := coreOf(tv.Type).(*types.Map); isMap {
		if b.commaOkContext(i) {
			return catalog.VariantMapLookupCommaOk, nil
		}
		return catalog.VariantMapLookupValue, nil
	}
	return catalog.VariantIndexElement, nil
}

// assignVariant resolves the assignment statement form.
func (b *builder) assignVariant(n *ast.AssignStmt) catalog.Variant {
	define := n.Tok == token.DEFINE
	if !define && n.Tok != token.ASSIGN {
		return catalog.VariantAssignCompound
	}
	if len(n.Lhs) != len(n.Rhs) && len(n.Rhs) == 1 {
		if _, isCall := ast.Unparen(n.Rhs[0]).(*ast.CallExpr); isCall {
			if define {
				return catalog.VariantDefineFromCall
			}
			return catalog.VariantAssignFromCall
		}
		if define {
			return catalog.VariantDefineCommaOk
		}
		return catalog.VariantAssignCommaOk
	}
	if define {
		return catalog.VariantDefineBalanced
	}
	return catalog.VariantAssignBalanced
}

// rangeVariant resolves the range operand class.
func (b *builder) rangeVariant(i int, n *ast.RangeStmt, info *types.Info) (catalog.Variant, error) {
	tv, ok := info.Types[n.X]
	if !ok {
		return catalog.VariantInvalid, newResolutionError(b.occurrences[i].kind, b.file, b.occurrences[i].span,
			"range operand has no type evidence")
	}
	switch t := coreOf(types.Default(tv.Type)).(type) {
	case *types.Array:
		return catalog.VariantRangeArray, nil
	case *types.Pointer:
		if _, toArray := coreOf(t.Elem()).(*types.Array); toArray {
			return catalog.VariantRangePointerToArray, nil
		}
	case *types.Slice:
		return catalog.VariantRangeSlice, nil
	case *types.Map:
		return catalog.VariantRangeMap, nil
	case *types.Chan:
		return catalog.VariantRangeChannel, nil
	case *types.Signature:
		return catalog.VariantRangeFunc, nil
	case *types.Basic:
		if t.Info()&types.IsString != 0 {
			return catalog.VariantRangeString, nil
		}
		if t.Info()&types.IsInteger != 0 {
			return catalog.VariantRangeInteger, nil
		}
	}
	return catalog.VariantInvalid, newResolutionError(b.occurrences[i].kind, b.file, b.occurrences[i].span,
		"range operand over unsupported type")
}

// commaOkContext reports whether the occurrence at index i is the single RHS
// of a two-target assignment or the single initializer of a two-name var
// binding — the comma-ok forms.
func (b *builder) commaOkContext(i int) bool {
	self := b.nodes[i]
	switch parent := b.parentNode(i).(type) {
	case *ast.AssignStmt:
		return len(parent.Lhs) == 2 && len(parent.Rhs) == 1 && parent.Rhs[0] == self
	case *ast.ValueSpec:
		return len(parent.Names) == 2 && len(parent.Values) == 1 && parent.Values[0] == self
	}
	return false
}

// aggregateShape resolves the effective aggregate shape of a composite
// literal type: alias-free underlying, resolving type-parameter core types and
// unwrapping one pointer level (a []*T literal's elements are implicit &T{}).
func aggregateShape(t types.Type) types.Type {
	u := coreOf(t)
	if pointer, ok := u.(*types.Pointer); ok {
		u = coreOf(pointer.Elem())
	}
	return u
}

// coreOf is the alias-free underlying of t, resolving a type parameter to the
// single core type of its constraint's type set when one exists.
func coreOf(t types.Type) types.Type {
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		if core := constraintCore(tp); core != nil {
			return core
		}
	}
	return types.Unalias(t).Underlying()
}

// constraintCore computes the single common underlying type of a constraint's
// type set, or nil when the set is empty or mixed.
func constraintCore(tp *types.TypeParam) types.Type {
	iface, ok := types.Unalias(tp.Constraint()).Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	var core types.Type
	ok = true
	var visit func(t types.Type)
	visit = func(t types.Type) {
		if !ok {
			return
		}
		switch t := types.Unalias(t).(type) {
		case *types.Union:
			for i := 0; i < t.Len(); i++ {
				visit(t.Term(i).Type())
			}
		case *types.Interface:
			for i := 0; i < t.NumEmbeddeds(); i++ {
				visit(t.EmbeddedType(i))
			}
		default:
			u := types.Unalias(t).Underlying()
			if core == nil {
				core = u
			} else if !types.Identical(core, u) {
				ok = false
			}
		}
	}
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		visit(iface.EmbeddedType(i))
	}
	if !ok {
		return nil
	}
	return core
}

// calleeIdent unwraps the identifier a callee/index base resolves through:
// a bare identifier or the selected name of a selector.
func calleeIdent(x ast.Expr) *ast.Ident {
	switch x := ast.Unparen(x).(type) {
	case *ast.Ident:
		return x
	case *ast.SelectorExpr:
		return x.Sel
	}
	return nil
}
