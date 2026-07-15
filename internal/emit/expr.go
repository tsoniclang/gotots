package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/ir"
)

// family maps an IR integer kind to its ABI carrier family.
func family(kind ir.Kind) (abi.Family, bool) {
	switch kind {
	case ir.KindInt8, ir.KindInt16, ir.KindInt32:
		return abi.FamilyNumberSigned, true
	case ir.KindUint8, ir.KindUint16, ir.KindUint32:
		return abi.FamilyNumberUnsigned, true
	case ir.KindInt64, ir.KindInt:
		return abi.FamilyBigSigned, true
	case ir.KindUint64, ir.KindUint, ir.KindUintptr:
		return abi.FamilyBigUnsigned, true
	}
	return "", false
}

func helper(name string) string { return "goabi$." + name }

// printExpr renders one IR expression, fully parenthesized.
func (p *printer) printExpr(e ir.Expr) (string, error) {
	if printed, isArray, err := p.printArrayExpr(e); isArray {
		return printed, err
	}
	if printed, isCollection, err := p.printCollectionExpr(e); isCollection {
		return printed, err
	}
	if printed, isIface, err := p.printIfaceExpr(e); isIface {
		return printed, err
	}
	switch n := e.(type) {
	case *ir.Const:
		return printConst(n)
	case *ir.VarRef:
		if n.Pkg == "" {
			return tsName(n.Name), nil
		}
		// A package-level variable: reads from other unit packages go
		// through the live ESM namespace binding.
		return p.module.symbol(n.Pkg, n.Name)
	case *ir.Binary:
		return p.printBinary(n)
	case *ir.Unary:
		return p.printUnary(n)
	case *ir.Convert:
		return p.printConvert(n)
	case *ir.Call:
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(n.Pkg, n.Callee)
		if err != nil {
			return "", err
		}
		// An instantiated generic call spells its type arguments
		// explicitly, so inference differences can never change types.
		typeArgs := ""
		if len(n.TypeArgs) > 0 {
			parts := make([]string, len(n.TypeArgs))
			for i, typeArg := range n.TypeArgs {
				spelled, err := p.tsType(typeArg)
				if err != nil {
					return "", err
				}
				parts[i] = spelled
			}
			typeArgs = "<" + strings.Join(parts, ", ") + ">"
		}
		return fmt.Sprintf("%s%s(%s)", callee, typeArgs, args), nil
	case *ir.MethodCall:
		recv, err := p.printExpr(n.Recv)
		if err != nil {
			return "", err
		}
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(n.Pkg, n.TypeName+"$"+n.Method)
		if err != nil {
			return "", err
		}
		// A value receiver dereferences a pointer caller at the call (Go
		// copies the pointee); a pointer receiver takes the pointer as
		// is — nil receivers run, as in Go.
		if !n.PointerRecv && n.Recv.Type().Kind == ir.KindPointer {
			recv = "gort$.goNilCheck(" + recv + ")"
		}
		if args == "" {
			return fmt.Sprintf("%s(%s)", callee, recv), nil
		}
		return fmt.Sprintf("%s(%s, %s)", callee, recv, args), nil
	case *ir.FieldLoad:
		base, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		if n.X.Type().Kind == ir.KindStruct {
			return fmt.Sprintf("%s.%s", base, n.Field), nil
		}
		return fmt.Sprintf("gort$.goNilCheck(%s).%s", base, n.Field), nil
	case *ir.StructNew:
		class, err := p.module.symbol(n.Pkg, n.TypeName)
		if err != nil {
			return "", err
		}
		if len(n.EvalOrder) == 0 {
			args, err := p.printArgs(n.Args)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("new %s(%s)", class, args), nil
		}
		// A keyed literal whose source order differs from field order:
		// the provided values evaluate in source order as arrow-function
		// arguments, then slot into constructor position. The staged
		// parameter names carry "$", which no Go identifier can spell.
		stagedParam := map[int]string{}
		var params, values []string
		for k, argIndex := range n.EvalOrder {
			spelled, err := p.tsType(n.Args[argIndex].Type())
			if err != nil {
				return "", err
			}
			name := fmt.Sprintf("$v%d", k)
			stagedParam[argIndex] = name
			params = append(params, name+": "+spelled)
			printed, err := p.printExpr(n.Args[argIndex])
			if err != nil {
				return "", err
			}
			values = append(values, printed)
		}
		ctorArgs := make([]string, len(n.Args))
		for i, arg := range n.Args {
			if name, staged := stagedParam[i]; staged {
				ctorArgs[i] = name
				continue
			}
			printed, err := p.printExpr(arg)
			if err != nil {
				return "", err
			}
			ctorArgs[i] = printed
		}
		return fmt.Sprintf("((%s) => new %s(%s))(%s)",
			strings.Join(params, ", "), class, strings.Join(ctorArgs, ", "), strings.Join(values, ", ")), nil
	case *ir.Closure:
		return p.printClosure(n)
	case *ir.MethodValue:
		return p.printMethodValue(n)
	case *ir.FuncRef:
		return p.module.symbol(n.Pkg, n.Name)
	case *ir.DynCall:
		fun, err := p.printExpr(n.Fun)
		if err != nil {
			return "", err
		}
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		// The helper preserves Go's order: callee operand, then the
		// arguments, then the nil panic at the invocation.
		call := fmt.Sprintf("gort$.goFuncInvoke(%s, [%s])", fun, args)
		return p.castResults(call, n.Results)
	case *ir.ExternalCall:
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(n.Pkg, n.Name)
		if err != nil {
			return "", err
		}
		typeArgs := ""
		if len(n.TypeArgs) > 0 {
			parts := make([]string, len(n.TypeArgs))
			for i, typeArg := range n.TypeArgs {
				spelled, err := p.tsType(typeArg)
				if err != nil {
					return "", err
				}
				parts[i] = spelled
			}
			typeArgs = "<" + strings.Join(parts, ", ") + ">"
		}
		return fmt.Sprintf("%s%s(%s)", callee, typeArgs, args), nil
	case *ir.StructCopy:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		if n.X.Type().Kind == ir.KindArray {
			cloneElem, err := p.arrayElemClone(*n.X.Type().Elem)
			if err != nil {
				return "", err
			}
			return "gosl$.goArrayClone(" + x + ", " + cloneElem + ")", nil
		}
		if t := n.X.Type(); t.Kind == ir.KindExternal {
			spelled, err := p.tsType(t)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("(goext$.goExternalCall(%q, [%s]) as %s)", t.Pkg+"."+t.Named+".goClone$", x, spelled), nil
		}
		return x + ".goClone$()", nil
	case *ir.StructZero:
		return p.zeroLiteral(n.T)
	case *ir.ExternZero:
		return p.zeroLiteral(n.T)
	case *ir.ExternVar:
		spelled, err := p.tsType(n.T)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(goext$.goExternalCall(%q, []) as (%s))", n.ID, spelled), nil
	case *ir.ExternalMethodCall:
		recv, err := p.printExpr(n.Recv)
		if err != nil {
			return "", err
		}
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		operands := recv
		if args != "" {
			operands += ", " + args
		}
		call := fmt.Sprintf("goext$.goExternalCall(%q, [%s])", n.TypeID+"."+n.Method, operands)
		return p.castResults(call, n.Results)
	case *ir.AddrOf:
		// The pointer to an addressable struct value is the instance.
		return p.printExpr(n.X)
	case *ir.Deref:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		// Identity carriers (structs, arrays, external handles) ARE their
		// pointer; cell carriers read through the cell.
		switch n.T.Kind {
		case ir.KindStruct, ir.KindArray, ir.KindExternal:
			return "gort$.goNilCheck(" + x + ")", nil
		}
		return "gort$.goNilCheck(" + x + ").v", nil
	case *ir.BoxedLoad:
		return n.Cell + ".v", nil
	case *ir.BoxedRef:
		return n.Cell, nil
	case *ir.CellNew:
		zero, err := p.printExpr(n.Zero)
		if err != nil {
			return "", err
		}
		spelled, err := p.tsType(*n.T.Elem)
		if err != nil {
			return "", err
		}
		return "({ v: " + zero + " } as gort$.GoCell<" + spelled + ">)", nil
	case *ir.NilConst:
		return "undefined", nil
	case *ir.IsNil:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		if n.Negate {
			return "(" + x + " !== undefined)", nil
		}
		return "(" + x + " === undefined)", nil
	case *ir.SliceLit:
		values, err := p.printArgs(n.Values)
		if err != nil {
			return "", err
		}
		return "gosl$.goSliceFrom([" + values + "])", nil
	case *ir.SliceMake:
		length, err := p.printExpr(n.Length)
		if err != nil {
			return "", err
		}
		// An omitted capacity defaults inside the helper, so the length
		// expression evaluates exactly once.
		capacity := "undefined"
		if n.Capacity != nil {
			capacity, err = p.printExpr(n.Capacity)
			if err != nil {
				return "", err
			}
		}
		zero, err := p.zeroLiteral(*n.T.Elem)
		if err != nil {
			return "", err
		}
		if n.T.Elem.Kind == ir.KindStruct || n.T.Elem.Kind == ir.KindArray {
			// Every struct or array element is a distinct fresh zero
			// instance.
			return "gosl$.goSliceMakeStruct(" + length + ", " + capacity + ", () => " + zero + ")", nil
		}
		return "gosl$.goSliceMake(" + length + ", " + capacity + ", " + zero + ")", nil
	case *ir.SliceGet:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		index, err := p.printExpr(n.Index)
		if err != nil {
			return "", err
		}
		return "gosl$.goSliceGet(" + x + ", " + index + ")", nil
	case *ir.SliceReslice:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		low := "0"
		if n.Low != nil {
			low, err = p.printExpr(n.Low)
			if err != nil {
				return "", err
			}
		}
		// An omitted high bound defaults to len(s) inside the helper,
		// computed from the single evaluation of the operand.
		high := "undefined"
		if n.High != nil {
			high, err = p.printExpr(n.High)
			if err != nil {
				return "", err
			}
		}
		return "gosl$.goSliceSlice(" + x + ", " + low + ", " + high + ")", nil
	case *ir.SliceAppend:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		values, err := p.printArgs(n.Values)
		if err != nil {
			return "", err
		}
		zero, err := p.zeroLiteral(*n.T.Elem)
		if err != nil {
			return "", err
		}
		if n.T.Elem.Kind == ir.KindStruct {
			return "gosl$.goSliceAppendStruct(" + x + ", [" + values + "], () => " + zero + ")", nil
		}
		return "gosl$.goSliceAppend(" + x + ", [" + values + "], " + zero + ")", nil
	case *ir.SliceAppendSlice:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		source, err := p.printExpr(n.Source)
		if err != nil {
			return "", err
		}
		zero, err := p.zeroLiteral(*n.T.Elem)
		if err != nil {
			return "", err
		}
		if n.T.Elem != nil && n.T.Elem.Kind == ir.KindStruct {
			return "gosl$.goSliceAppendSliceStruct(" + x + ", " + source + ", () => " + zero + ")", nil
		}
		return "gosl$.goSliceAppendSlice(" + x + ", " + source + ", " + zero + ")", nil
	case *ir.SliceCopy:
		dst, err := p.printExpr(n.Dst)
		if err != nil {
			return "", err
		}
		src, err := p.printExpr(n.Src)
		if err != nil {
			return "", err
		}
		if n.Dst.Type().Elem != nil && n.Dst.Type().Elem.Kind == ir.KindStruct {
			return "gosl$.goSliceCopyStruct(" + dst + ", " + src + ")", nil
		}
		return "gosl$.goSliceCopy(" + dst + ", " + src + ")", nil
	case *ir.SliceLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gosl$.goSliceLen(" + x + ")", nil
	case *ir.SliceCap:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gosl$.goSliceCap(" + x + ")", nil
	}
	return "", fmt.Errorf("no emission for IR expression %T", e)
}

// castResults types a dynamically dispatched call's unknown result with
// an erasable cast.
func (p *printer) castResults(call string, results []ir.Type) (string, error) {
	switch len(results) {
	case 0:
		return call, nil
	case 1:
		spelled, err := p.tsType(results[0])
		if err != nil {
			return "", err
		}
		return "(" + call + " as (" + spelled + "))", nil
	default:
		parts := make([]string, len(results))
		for i, result := range results {
			spelled, err := p.tsType(result)
			if err != nil {
				return "", err
			}
			parts[i] = spelled
		}
		return "(" + call + " as (readonly [" + strings.Join(parts, ", ") + "]))", nil
	}
}

// printClosure emits a function literal as an arrow function: JS arrows
// capture enclosing variables by reference with per-iteration loop
// bindings, exactly matching Go's capture semantics.
func (p *printer) printClosure(n *ir.Closure) (string, error) {
	var params []string
	for _, parameter := range n.Params {
		spelled, err := p.tsType(parameter.Type)
		if err != nil {
			return "", err
		}
		params = append(params, tsName(parameter.Name)+": "+spelled)
	}
	result, err := p.tsResultType(n.Results)
	if err != nil {
		return "", err
	}
	var sub strings.Builder
	subPrinter := &printer{out: &sub, module: p.module, indent: p.indent + 1}
	if err := subPrinter.printDeferWrappedBody(n.Body, n.UsesDeferStack); err != nil {
		return "", err
	}
	closing := strings.Repeat("  ", p.indent)
	return "((" + strings.Join(params, ", ") + "): " + result + " => {\n" + sub.String() + closing + "})", nil
}

func (p *printer) printArgs(args []ir.Expr) (string, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		if spread, isSpread := arg.(*ir.TupleSpread); isSpread {
			inner, err := p.printExpr(spread.X)
			if err != nil {
				return "", err
			}
			parts[i] = "...(" + inner + ")"
			continue
		}
		printed, err := p.printExpr(arg)
		if err != nil {
			return "", err
		}
		parts[i] = printed
	}
	return joinComma(parts), nil
}

// zeroLiteral spells the Go zero value of a reviewed type in TypeScript.
// A struct zero is a fresh instance from the class's zero factory.
func (p *printer) zeroLiteral(t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindBool:
		return "false", nil
	case t.Kind == ir.KindString:
		return `""`, nil
	case t.Kind.Nilable():
		return "undefined", nil
	case t.Kind == ir.KindStruct:
		class, err := p.module.symbol(t.Pkg, t.Named)
		if err != nil {
			return "", err
		}
		return class + ".goZero$()", nil
	case t.Kind == ir.KindArray:
		return p.arrayZeroFactory(t)
	case t.Kind == ir.KindUnit:
		return "0", nil
	case t.Kind == ir.KindExternal:
		spelled, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(goext$.goExternalCall(%q, []) as %s)", t.Pkg+"."+t.Named+".goZero$", spelled), nil
	case t.Kind.Wide64():
		return "0n", nil
	case t.Kind.Integer(), t.Kind.Float():
		return "0", nil
	}
	return "", fmt.Errorf("no zero literal for type %q", t.Go)
}
