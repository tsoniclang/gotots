// Runtime-identity requirement scan: a generic declaration whose body
// converts a bare type-parameter VALUE into an interface needs the
// binding's runtime type identity (discriminant, rtti, vtable) — the
// rt$P slot of the factory protocol, requirement-scoped exactly like
// key$P. The scan walks every implicit-conversion context the builder
// admits (assignments, declarations, call arguments, returns, composite
// literals) with checker evidence; a context it does not model leaves
// the requirement unset, and the box site fails closed at build — never
// silently.
package translate

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/ir"

	"golang.org/x/tools/go/packages"
)

// collectParamRttiRequirements records, for every generic declaration,
// which of its type parameters are boxed into interfaces by its body.
func collectParamRttiRequirements(unit ir.Scope, pkgs []*packages.Package) {
	for _, p := range pkgs {
		ranges := genericDeclRanges(p)
		info := p.TypesInfo
		require := func(pos token.Pos, source, target types.Type) {
			if source == nil || target == nil {
				return
			}
			param, isParam := types.Unalias(source).(*types.TypeParam)
			if !isParam {
				return
			}
			if _, targetParam := types.Unalias(target).(*types.TypeParam); targetParam {
				// P into a parameter-typed slot is a value bind, not a box.
				return
			}
			if _, isIface := types.Unalias(target).Underlying().(*types.Interface); !isIface {
				return
			}
			owner := ranges.at(pos)
			if owner == nil {
				return
			}
			// A method on a generic type resolves its receiver parameters
			// through the TYPE's object — the same identity every mask
			// lookup (declRttiRequires, call-site threading) reads.
			if fn, isFunc := owner.(*types.Func); isFunc {
				signature := fn.Type().(*types.Signature)
				if recvParams := signature.RecvTypeParams(); recvParams != nil && recvParams.Len() > 0 {
					recv := signature.Recv().Type()
					if pointer, isPointer := types.Unalias(recv).(*types.Pointer); isPointer {
						recv = pointer.Elem()
					}
					if named, isNamed := types.Unalias(recv).(*types.Named); isNamed {
						owner = named.Obj()
					}
				}
			}
			if index, mine := ownerParamIndex(owner, param); mine {
				unit.RequireParamRtti(owner, index)
			}
		}
		exprType := func(e ast.Expr) types.Type {
			if tv, ok := info.Types[e]; ok {
				return tv.Type
			}
			return nil
		}
		for _, file := range p.Syntax {
			var funcLits []*ast.FuncLit
			ast.Inspect(file, func(n ast.Node) bool {
				if lit, isLit := n.(*ast.FuncLit); isLit {
					funcLits = append(funcLits, lit)
				}
				return true
			})
			ast.Inspect(file, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.AssignStmt:
					if len(n.Lhs) != len(n.Rhs) {
						return true
					}
					for i, rhs := range n.Rhs {
						require(n.Pos(), exprType(rhs), exprType(n.Lhs[i]))
					}
				case *ast.ValueSpec:
					if n.Type == nil {
						return true
					}
					declared := exprType(n.Type)
					for _, value := range n.Values {
						require(n.Pos(), exprType(value), declared)
					}
				case *ast.ReturnStmt:
					// The INNERMOST enclosing function's results type this
					// return — a closure inside a generic declaration returns
					// to its own signature (computeFn's func literal returns T
					// into any).
					var results *types.Tuple
					if lit := innermostFuncLit(funcLits, n.Pos()); lit != nil {
						if sig, isSig := exprType(lit).(*types.Signature); isSig {
							results = sig.Results()
						}
					} else if fn, isFunc := ranges.at(n.Pos()).(*types.Func); isFunc {
						results = fn.Type().(*types.Signature).Results()
					}
					if results == nil || results.Len() != len(n.Results) {
						return true
					}
					for i, result := range n.Results {
						require(n.Pos(), exprType(result), results.At(i).Type())
					}
				case *ast.CallExpr:
					tv, ok := info.Types[n.Fun]
					if !ok || tv.IsType() {
						return true
					}
					signature, isSig := tv.Type.(*types.Signature)
					if !isSig {
						return true
					}
					for i, arg := range n.Args {
						var target types.Type
						switch {
						case i < signature.Params().Len()-1 || (i < signature.Params().Len() && !signature.Variadic()):
							target = signature.Params().At(i).Type()
						case signature.Variadic() && !n.Ellipsis.IsValid():
							last := signature.Params().At(signature.Params().Len() - 1).Type()
							if slice, isSlice := last.Underlying().(*types.Slice); isSlice {
								target = slice.Elem()
							}
						}
						require(n.Pos(), exprType(arg), target)
					}
				case *ast.CompositeLit:
					litType := exprType(n)
					if litType == nil {
						return true
					}
					switch u := litType.Underlying().(type) {
					case *types.Struct:
						for i, element := range n.Elts {
							if kv, isKV := element.(*ast.KeyValueExpr); isKV {
								name, isIdent := kv.Key.(*ast.Ident)
								if !isIdent {
									continue
								}
								for f := range u.NumFields() {
									if u.Field(f).Name() == name.Name {
										require(n.Pos(), exprType(kv.Value), u.Field(f).Type())
									}
								}
								continue
							}
							if i < u.NumFields() {
								require(n.Pos(), exprType(element), u.Field(i).Type())
							}
						}
					case *types.Slice:
						for _, element := range n.Elts {
							if kv, isKV := element.(*ast.KeyValueExpr); isKV {
								require(n.Pos(), exprType(kv.Value), u.Elem())
								continue
							}
							require(n.Pos(), exprType(element), u.Elem())
						}
					case *types.Array:
						for _, element := range n.Elts {
							if kv, isKV := element.(*ast.KeyValueExpr); isKV {
								require(n.Pos(), exprType(kv.Value), u.Elem())
								continue
							}
							require(n.Pos(), exprType(element), u.Elem())
						}
					case *types.Map:
						for _, element := range n.Elts {
							if kv, isKV := element.(*ast.KeyValueExpr); isKV {
								require(n.Pos(), exprType(kv.Key), u.Key())
								require(n.Pos(), exprType(kv.Value), u.Elem())
							}
						}
					}
				}
				return true
			})
		}
	}
}

// innermostFuncLit finds the innermost function literal enclosing a
// position (nil when the position is directly in a declared function).
func innermostFuncLit(lits []*ast.FuncLit, pos token.Pos) *ast.FuncLit {
	var innermost *ast.FuncLit
	for _, lit := range lits {
		if pos < lit.Pos() || pos >= lit.End() {
			continue
		}
		if innermost == nil || lit.Pos() > innermost.Pos() {
			innermost = lit
		}
	}
	return innermost
}

// ownerParamIndex resolves a type parameter's position among its owning
// generic declaration's parameters — receiver parameters are DISTINCT
// objects from the type's own, matched by name.
func ownerParamIndex(owner types.Object, param *types.TypeParam) (int, bool) {
	var params *types.TypeParamList
	switch o := owner.(type) {
	case *types.TypeName:
		if named, ok := o.Type().(*types.Named); ok {
			params = named.TypeParams()
		}
	case *types.Func:
		signature := o.Type().(*types.Signature)
		params = signature.TypeParams()
		if recv := signature.RecvTypeParams(); recv != nil && recv.Len() > 0 {
			params = recv
		}
	}
	if params == nil {
		return 0, false
	}
	for i := range params.Len() {
		if params.At(i) == param || params.At(i).Obj().Name() == param.Obj().Name() {
			return i, true
		}
	}
	return 0, false
}
