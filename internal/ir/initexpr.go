package ir

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

// BuildPackageVarInit builds one package-level variable initializer (or
// the zero value when e is nil). Go initializes package variables in
// dependency order, which top-of-module evaluation cannot re-derive, so
// only order-independent initializers are admitted: no calls and no
// variable reads anywhere in the evaluated expression. Closure bodies
// are exempt — they evaluate at their eventual call, under the fully
// initialized module.
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
	built, err := b.buildExprAs(e, expected)
	if err != nil {
		return nil, err
	}
	if err := checkInitOrderFree(built, span); err != nil {
		return nil, err
	}
	return built, nil
}

// checkInitOrderFree walks an initializer's evaluated expression tree
// and rejects everything whose value could depend on initialization
// order: calls of any form and variable reads. Closure bodies are not
// walked (deferred evaluation).
func checkInitOrderFree(e Expr, span Span) error {
	reject := func(construct string) error {
		return &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
			Construct: "package-level variable initializer with " + construct + " (initialization order)", Span: span}
	}
	each := func(children ...Expr) error {
		for _, child := range children {
			if child == nil {
				continue
			}
			if err := checkInitOrderFree(child, span); err != nil {
				return err
			}
		}
		return nil
	}
	switch n := e.(type) {
	case *Call, *MethodCall, *DynCall:
		return reject("a call")
	case *VarRef:
		return reject("a variable read")
	case *Closure, *Const, *NilConst, *StructZero, *FuncRef, *MapMake:
		return nil
	case *Binary:
		return each(n.L, n.R)
	case *Unary:
		return each(n.X)
	case *Convert:
		return each(n.X)
	case *StructNew:
		return each(n.Args...)
	case *StructCopy:
		return each(n.X)
	case *AddrOf:
		return each(n.X)
	case *Deref:
		return each(n.X)
	case *IsNil:
		return each(n.X)
	case *FieldLoad:
		return each(n.X)
	case *MapFrom:
		if err := each(n.Keys...); err != nil {
			return err
		}
		return each(n.Values...)
	case *MapGet:
		return each(n.Map, n.Key)
	case *MapLookup:
		return each(n.Map, n.Key)
	case *MapLen:
		return each(n.X)
	case *StringLen:
		return each(n.X)
	case *SliceLit:
		return each(n.Values...)
	case *SliceMake:
		return each(n.Length, n.Capacity)
	case *SliceGet:
		return each(n.X, n.Index)
	case *SliceReslice:
		return each(n.X, n.Low, n.High)
	case *SliceAppend:
		if err := each(n.X); err != nil {
			return err
		}
		return each(n.Values...)
	case *SliceAppendSlice:
		return each(n.X, n.Source)
	case *SliceCopy:
		return each(n.Dst, n.Src)
	case *SliceLen:
		return each(n.X)
	case *SliceCap:
		return each(n.X)
	}
	return reject("an unreviewed expression form")
}
