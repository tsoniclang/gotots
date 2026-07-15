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
	case *ir.IfaceBox:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		rtti, err := p.rttiRef(n.Rtti)
		if err != nil {
			return "", err
		}
		return "goif$.goIfaceBox(" + rtti + ", " + x + ")", nil
	case *ir.IfaceCall:
		recv, err := p.printExpr(n.Recv)
		if err != nil {
			return "", err
		}
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		call := fmt.Sprintf("goif$.goIfaceCall(%s, %q, [%s])", recv, n.Method, args)
		return p.castResults(call, n.Results)
	case *ir.TypeAssert:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		rtti, err := p.rttiRef(n.Rtti)
		if err != nil {
			return "", err
		}
		if n.CommaOk {
			zero, err := p.zeroLiteral(n.Target)
			if err != nil {
				return "", err
			}
			return "goif$.goIfaceLookup(" + x + ", " + rtti + ", " + zero + ")", nil
		}
		spelled, err := p.tsType(n.Target)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(goif$.goIfaceAssert(%s, %s, %q) as (%s))", x, rtti, n.SourceDisplay, spelled), nil
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
		return x + ".goClone$()", nil
	case *ir.StructZero:
		return p.zeroLiteral(n.T)
	case *ir.AddrOf:
		// The pointer to an addressable struct value is the instance.
		return p.printExpr(n.X)
	case *ir.Deref:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort$.goNilCheck(" + x + ")", nil
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
	case *ir.MapMake:
		return "gort$.goMapMake()", nil
	case *ir.MapFrom:
		var entries []string
		for i := range n.Keys {
			key, err := p.printExpr(n.Keys[i])
			if err != nil {
				return "", err
			}
			value, err := p.printExpr(n.Values[i])
			if err != nil {
				return "", err
			}
			entries = append(entries, "["+key+", "+value+"]")
		}
		return "gort$.goMapFrom([" + joinComma(entries) + "])", nil
	case *ir.MapGet:
		return p.printMapAccess("goMapGet", n.Map, n.Key, n.T)
	case *ir.MapLookup:
		return p.printMapAccess("goMapLookup", n.Map, n.Key, n.T)
	case *ir.MapLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort$.goMapLen(" + x + ")", nil
	case *ir.StringLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort$.goStringLen(" + x + ")", nil
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
	if err := subPrinter.printBlockBody(n.Body); err != nil {
		return "", err
	}
	closing := strings.Repeat("  ", p.indent)
	return "((" + strings.Join(params, ", ") + "): " + result + " => {\n" + sub.String() + closing + "})", nil
}

func (p *printer) printArgs(args []ir.Expr) (string, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		printed, err := p.printExpr(arg)
		if err != nil {
			return "", err
		}
		parts[i] = printed
	}
	return joinComma(parts), nil
}

// printMapAccess emits a map read with the exact zero value of the map's
// value type.
func (p *printer) printMapAccess(helper string, mapExpr, key ir.Expr, valueType ir.Type) (string, error) {
	m, err := p.printExpr(mapExpr)
	if err != nil {
		return "", err
	}
	k, err := p.printExpr(key)
	if err != nil {
		return "", err
	}
	zero, err := p.zeroLiteral(valueType)
	if err != nil {
		return "", err
	}
	return "gort$." + helper + "(" + m + ", " + k + ", " + zero + ")", nil
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
	case t.Kind.Wide64():
		return "0n", nil
	case t.Kind.Integer(), t.Kind.Float():
		return "0", nil
	}
	return "", fmt.Errorf("no zero literal for type %q", t.Go)
}
