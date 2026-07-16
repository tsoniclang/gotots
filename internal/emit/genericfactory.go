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
		return "goif$.goIfaceEqual", nil
	case t.Kind == ir.KindStruct:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		return "(($a: " + spelled + ", $b: " + spelled + ") => $a.goEq$($b))", nil
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
			return class + ".goZero$<" + joinComma(args) + ">(" + joinComma(factories) + ")", nil
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
