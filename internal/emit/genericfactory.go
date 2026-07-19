// Generic-instantiation factory emission: the zero and equality
// operations each generic call passes per type argument, and the Go
// zero-value spelling used throughout.
package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

// zeroFactoryArgs spells one zero factory then one equality operation
// per instantiated type argument, the trailing arguments of every
// generic function and method call.
func (p *printer) zeroFactoryArgs(typeArgs []ir.Type) (string, error) {
	parts := make([]string, 0, len(typeArgs)*2)
	for _, arg := range typeArgs {
		zero, err := p.zeroLiteral(arg)
		if err != nil {
			return "", err
		}
		parts = append(parts, "() => "+zero)
	}
	for _, arg := range typeArgs {
		eq, err := p.eqOperation(arg)
		if err != nil {
			return "", err
		}
		parts = append(parts, eq)
	}
	for _, arg := range typeArgs {
		clone, err := p.cloneOperation(arg)
		if err != nil {
			return "", err
		}
		parts = append(parts, clone)
	}
	for _, arg := range typeArgs {
		set, err := p.setOperation(arg)
		if err != nil {
			return "", err
		}
		parts = append(parts, set)
	}
	return joinComma(parts), nil
}

// cloneOperation spells the TOTAL per-binding copy of one instantiation
// type argument: the exact value copy for value-copy carriers (struct,
// fixed array, external value), the in-scope clone$U for a nested type
// parameter, and the identity arrow for every carrier whose assignment
// already copies (scalars, strings, pointers, slices, maps, functions,
// interfaces).
func (p *printer) cloneOperation(t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindIface && t.TypeParamName != "":
		op, has := p.cloneOps[t.TypeParamName]
		if !has {
			return "", fmt.Errorf("no clone operation in scope for type parameter %q", t.TypeParamName)
		}
		return op, nil
	case t.Kind == ir.KindStruct:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		return "(($v: " + spelled + ") => $v.goClone$())", nil
	case t.Kind == ir.KindArray:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		cloneElem, err := p.arrayElemClone(*t.Elem)
		if err != nil {
			return "", err
		}
		return "(($v: " + spelled + ") => gosl$.goArrayClone($v, " + cloneElem + "))", nil
	case t.Kind == ir.KindExternal:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(t.Pkg, externCloneSymbol(t.Named))
		if err != nil {
			return "", err
		}
		return "(($v: " + spelled + ") => " + callee + "($v))", nil
	}
	spelled, err := p.tsType(t)
	if err != nil {
		return "", err
	}
	return "(($v: " + spelled + ") => $v)", nil
}

// setOperation spells the per-binding in-place overwrite of one
// instantiation type argument, or "undefined" for every carrier whose
// store is a plain slot assignment.
func (p *printer) setOperation(t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindIface && t.TypeParamName != "":
		op, has := p.setOps[t.TypeParamName]
		if !has {
			return "", fmt.Errorf("no set operation in scope for type parameter %q", t.TypeParamName)
		}
		return op, nil
	case t.Kind == ir.KindStruct:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		return "(($d: " + spelled + ", $s: " + spelled + ") => $d.goSet$($s))", nil
	case t.Kind == ir.KindArray:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		setElem, err := p.arrayElemSet(*t.Elem)
		if err != nil {
			return "", err
		}
		return "(($d: " + spelled + ", $s: " + spelled + ") => gosl$.goArraySetAll($d, $s, " + setElem + "))", nil
	case t.Kind == ir.KindExternal:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		callee, err := p.externSetCallee(t)
		if err != nil {
			return "", err
		}
		if callee == "" {
			return "undefined", nil
		}
		return "(($d: " + spelled + ", $s: " + spelled + ") => " + callee + "($d, $s))", nil
	}
	return "undefined", nil
}

// eqCloneSetFactoryArgs spells the equality/clone/set factory triple per
// type argument — the trailing constructor arguments of every generic
// class (captured at construction so goEq$/goClone$/goSet$ stay
// source-shaped).
func (p *printer) eqCloneSetFactoryArgs(typeArgs []ir.Type) (string, error) {
	parts := make([]string, 0, len(typeArgs)*3)
	for _, arg := range typeArgs {
		eq, err := p.eqOperation(arg)
		if err != nil {
			return "", err
		}
		clone, err := p.cloneOperation(arg)
		if err != nil {
			return "", err
		}
		set, err := p.setOperation(arg)
		if err != nil {
			return "", err
		}
		parts = append(parts, eq, clone, set)
	}
	return joinComma(parts), nil
}

// zeroOnlyFactoryArgs spells the zero factories alone — generic class
// constructors and goZero$ take no equality operations.
func (p *printer) zeroOnlyFactoryArgs(typeArgs []ir.Type) (string, error) {
	parts := make([]string, 0, len(typeArgs))
	for _, arg := range typeArgs {
		zero, err := p.zeroLiteral(arg)
		if err != nil {
			return "", err
		}
		parts = append(parts, "() => "+zero)
	}
	return joinComma(parts), nil
}

// eqOperation spells the exact == operation for one instantiation type
// argument: direct === for comparable scalars and pointers, interface
// equality (with the uncomparable panic) for interfaces, the type's own
// goEq$ for comparable structs, and the in-scope eq$U for a nested type
// parameter.
func (p *printer) eqOperation(t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindIface && t.TypeParamName != "":
		op, has := p.eqOps[t.TypeParamName]
		if !has {
			return "", fmt.Errorf("no equality operation in scope for type parameter %q", t.TypeParamName)
		}
		return op, nil
	case t.Kind == ir.KindIface:
		// The interface's own exact union equality function (per-member
		// narrowing, no erased payload) is the instantiation's == operation.
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		return "(($a: " + spelled + ", $b: " + spelled + ") => " + spelled + "$eq($a, $b))", nil
	case t.Kind == ir.KindStruct:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		if t.Uncomparable {
			// Go's type system rejects == on this binding wherever it could
			// run, so the operation is provably unreachable: fail closed.
			return "(($a: " + spelled + ", $b: " + spelled + ") => gort$.goEqUnsupported())", nil
		}
		return "(($a: " + spelled + ", $b: " + spelled + ") => $a.goEq$($b))", nil
	}
	if t.Kind == ir.KindSlice || t.Kind == ir.KindMap || t.Kind == ir.KindFunc || t.Kind == ir.KindChan {
		// Go's type system rejects == on this binding wherever it could
		// run (slices, maps, functions, and channels never satisfy
		// comparable), so the operation is provably unreachable: fail
		// closed instead of an inexact reference ===.
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		return "(($a: " + spelled + ", $b: " + spelled + ") => gort$.goEqUnsupported())", nil
	}
	spelled, err := p.tsType(t)
	if err != nil {
		return "", err
	}
	return "(($a: " + spelled + ", $b: " + spelled + ") => $a === $b)", nil
}

// zeroLiteral spells the Go zero value of a reviewed type in TypeScript.
// A struct zero is a fresh instance from the class's zero factory.
func (p *printer) zeroLiteral(t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindBool:
		return "false", nil
	case t.Kind == ir.KindString:
		return `""`, nil
	case t.Kind == ir.KindIface && t.TypeParamName != "":
		factory, has := p.zeroFactories[t.TypeParamName]
		if !has {
			return "", fmt.Errorf("no zero factory in scope for type parameter %q", t.TypeParamName)
		}
		return factory + "()", nil
	case t.Kind.Nilable():
		return "undefined", nil
	case t.Kind == ir.KindStruct:
		class, err := p.module.symbol(t.Pkg, t.Named)
		if err != nil {
			return "", err
		}
		if len(t.TypeArgs) > 0 {
			args := make([]string, len(t.TypeArgs))
			factories := make([]string, len(t.TypeArgs))
			for i, arg := range t.TypeArgs {
				spelled, err := p.tsType(arg)
				if err != nil {
					return "", err
				}
				args[i] = spelled
				zero, err := p.zeroLiteral(arg)
				if err != nil {
					return "", err
				}
				factories[i] = "() => " + zero
			}
			cloneSet, err := p.eqCloneSetFactoryArgs(t.TypeArgs)
			if err != nil {
				return "", err
			}
			all := joinComma(factories)
			if cloneSet != "" {
				all += ", " + cloneSet
			}
			return class + ".goZero$<" + joinComma(args) + ">(" + all + ")", nil
		}
		return class + ".goZero$()", nil
	case t.Kind == ir.KindArray:
		return p.arrayZeroFactory(t)
	case t.Kind == ir.KindUnit:
		return "0", nil
	case t.Kind == ir.KindExternal:
		callee, err := p.module.symbol(t.Pkg, externZeroSymbol(t.Named))
		if err != nil {
			return "", err
		}
		return callee + "()", nil
	case t.Kind.Wide64():
		return "0n", nil
	case t.Kind.Integer(), t.Kind.Float():
		return "0", nil
	}
	return "", fmt.Errorf("no zero literal for type %q", t.Go)
}
