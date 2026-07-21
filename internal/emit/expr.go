package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/plan"
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
	case *ir.ParamRef:
		if !generatedIdentifier(n.Name) {
			return "", fmt.Errorf("ParamRef %q is not a bare generated identifier", n.Name)
		}
		return n.Name, nil
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
	case *ir.ParamReprView:
		return p.printParamReprView(n)
	case *ir.ParamReprCast:
		return p.printParamReprCast(n)
	case *ir.SliceSlotEq:
		return p.printSliceSlotEq(n)
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
		// explicitly, so inference differences can never change types,
		// and passes each argument's zero factory as trailing arguments.
		typeArgs := ""
		if len(n.TypeArgs) > 0 {
			parts := make([]string, 0, len(n.TypeArgs))
			surfaceArgs := make([]ir.Type, 0, len(n.TypeArgs))
			surfaceKeyed := make([]bool, 0, len(n.TypeArgs))
			for i, typeArg := range n.TypeArgs {
				if erasedAt(n.ErasedParams, i) {
					continue
				}
				spelled, err := p.tsType(typeArg)
				if err != nil {
					return "", err
				}
				parts = append(parts, spelled)
				surfaceArgs = append(surfaceArgs, typeArg)
				surfaceKeyed = append(surfaceKeyed, i < len(n.KeyedParams) && n.KeyedParams[i])
			}
			typeArgs = ""
			if len(parts) > 0 {
				typeArgs = "<" + strings.Join(parts, ", ") + ">"
			}
			factories, err := p.zeroFactoryArgs(surfaceArgs, surfaceKeyed)
			if err != nil {
				return "", err
			}
			rttiSlots, err := p.rttiFactoryArgs(n.RttiParams, n.RttiArgs, n.ErasedParams)
			if err != nil {
				return "", err
			}
			if rttiSlots != "" {
				factories += ", " + rttiSlots
			}
			if args == "" {
				args = factories
			} else {
				args += ", " + factories
			}
			variant := n.Callee
			if callFamilyEnc(n.TypeArgs, n.HardKeyed, p.familyEnc) {
				variant += "$ek"
			}
			if callFamilyPtr(n.TypeArgs, n.PtrParams, p.familyPtrCell) {
				variant += "$pc"
			}
			if variant != n.Callee {
				renamed, err := p.module.symbol(n.Pkg, variant)
				if err != nil {
					return "", err
				}
				callee = renamed
			}
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
		methodClass := n.TypeName
		recvT := n.Recv.Type()
		if recvT.Kind == ir.KindPointer && recvT.Elem != nil {
			recvT = *recvT.Elem
		}
		if recvT.MapFamilyEnc || (p.familyEnc && selfHardKeyedReference(recvT)) {
			methodClass += "$ek"
		}
		if recvT.MapFamilyPtrCell || (p.familyPtrCell && selfPtrReference(recvT)) {
			methodClass += "$pc"
		}
		// A family accessor renamed to avoid a field collision is called by
		// its renamed member name (the definition renames identically).
		member := tsName(n.Method)
		if _, _, ok := p.module.familyClassAndFamily(recvT); ok {
			member = p.module.familyMemberName(n.Method)
		}
		if n.Promoted {
			receiver := recv
			if n.Recv.Type().Kind == ir.KindPointer {
				checked, err := p.nilCheckOf(recv, n.Recv.Type())
				if err != nil {
					return "", err
				}
				receiver = checked
			}
			return receiver + "." + member + "(" + args + ")", nil
		}
		if len(n.TypeArgs) == 0 && methodClass == n.TypeName {
			emission := p.module.MethodEmissionFor(goid.Method(n.Pkg, n.TypeName, n.Method))
			receiverProven := n.Recv.Type().Kind != ir.KindPointer ||
				(p.memberReceiver != "" && recv == p.memberReceiver) || p.nilChecked(recv)
			switch {
			case emission == plan.MethodOrdinaryNilChecked:
				receiver := recv
				if n.Recv.Type().Kind == ir.KindPointer {
					checked, err := p.nilCheckOf(recv, n.Recv.Type())
					if err != nil {
						return "", err
					}
					receiver = checked
				}
				return receiver + "." + member + "(" + args + ")", nil
			case emission == plan.MethodFreeFunctionException && receiverProven:
				// The delegate member: source-shaped at proven sites,
				// forwarding to the single free-function body.
				suffix := ""
				if p.nilChecked(recv) && n.Recv.Type().Kind == ir.KindPointer {
					suffix = "!"
				}
				return recv + suffix + "." + member + "(" + args + ")", nil
			}
		}
		callee, err := p.module.symbol(n.Pkg, methodClass+"$"+n.Method)
		if err != nil {
			return "", err
		}
		if len(n.TypeArgs) > 0 {
			parts := make([]string, 0, len(n.TypeArgs))
			surfaceArgs := make([]ir.Type, 0, len(n.TypeArgs))
			surfaceKeyed := make([]bool, 0, len(n.TypeArgs))
			for i, typeArg := range n.TypeArgs {
				if erasedAt(n.ErasedParams, i) {
					continue
				}
				spelled, err := p.tsType(typeArg)
				if err != nil {
					return "", err
				}
				parts = append(parts, spelled)
				surfaceArgs = append(surfaceArgs, typeArg)
				surfaceKeyed = append(surfaceKeyed, i < len(n.KeyedParams) && n.KeyedParams[i])
			}
			if len(parts) > 0 {
				callee += "<" + joinComma(parts) + ">"
			}
			factories, err := p.zeroFactoryArgs(surfaceArgs, surfaceKeyed)
			if err != nil {
				return "", err
			}
			rttiSlots, err := p.rttiFactoryArgs(n.RttiParams, n.RttiArgs, n.ErasedParams)
			if err != nil {
				return "", err
			}
			if rttiSlots != "" {
				factories += ", " + rttiSlots
			}
			if args == "" {
				args = factories
			} else {
				args += ", " + factories
			}
		}
		// A value receiver dereferences a pointer caller at the call (Go
		// copies the pointee); a pointer receiver takes the pointer as
		// is — nil receivers run, as in Go. Cell pointees read through
		// the cell.
		if !n.PointerRecv && n.Recv.Type().Kind == ir.KindPointer {
			if recv, err = p.nilCheckOf(recv, n.Recv.Type()); err != nil {
				return "", err
			}
			if elem := n.Recv.Type().Elem; elem != nil {
				switch elem.Kind {
				case ir.KindStruct, ir.KindArray, ir.KindExternal:
				default:
					recv += ".v"
				}
			}
		}
		if args == "" {
			return fmt.Sprintf("%s(%s)", callee, recv), nil
		}
		return fmt.Sprintf("%s(%s, %s)", callee, recv, args), nil
	case *ir.FieldLoad:
		// Object-model spine collapse (ADR-0012): selecting the primary
		// embedded base of a family class is inherited access — the base IS
		// the object under native inheritance — so the hop disappears. A
		// nilable base keeps exactly one deref check; recursion collapses the
		// whole `x.ExprBase.NodeBase.Node` chain to `x`.
		if p.module.isFamilyPrimaryHop(n.X.Type(), n.Field) {
			base, err := p.printExpr(n.X)
			if err != nil {
				return "", err
			}
			if n.X.Type().Kind == ir.KindPointer {
				return "gort$.goNilCheck(" + base + ")", nil
			}
			return base, nil
		}
		base, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		// A cell field's value is read through the stable cell (.v); its
		// storage is one GoCell<T> so the field's address stays exact.
		suffix := "." + n.Field
		if n.Cell {
			suffix += ".v"
		}
		if n.X.Type().Kind == ir.KindStruct {
			return base + suffix, nil
		}
		checked, err := p.nilCheckOf(base, n.X.Type())
		if err != nil {
			return "", err
		}
		return checked + suffix, nil
	case *ir.StructNew:
		structT := n.T
		if structT.Kind == ir.KindPointer && structT.Elem != nil {
			structT = *structT.Elem
		}
		newClass := n.TypeName
		if structT.MapFamilyEnc || (p.familyEnc && selfHardKeyedReference(structT)) {
			newClass += "$ek"
		}
		if structT.MapFamilyPtrCell || (p.familyPtrCell && selfPtrReference(structT)) {
			newClass += "$pc"
		}
		class, err := p.module.symbol(n.Pkg, newClass)
		if err != nil {
			return "", err
		}
		// Object-model construction (ADR-0012): a family class has a
		// flattened total constructor over its transitive own fields, so a
		// composite literal's field-ordered args are reshaped into that
		// order (provided values by name, zeros for inherited/omitted).
		if fc, ok := p.module.familyClassOf(structT.Named); ok {
			return p.printFamilyStructNew(fc, n, class)
		}
		if len(structT.TypeArgs) > 0 {
			args := make([]string, len(structT.TypeArgs))
			for i, arg := range structT.TypeArgs {
				if args[i], err = p.tsType(arg); err != nil {
					return "", err
				}
			}
			class += "<" + joinComma(args) + ">"
		}
		factoryArgs, err := p.eqCloneSetFactoryArgs(structT.TypeArgs, structT.ClassKeyParams)
		if err != nil {
			return "", err
		}
		if len(n.EvalOrder) == 0 {
			args, err := p.printArgs(n.Args)
			if err != nil {
				return "", err
			}
			if factoryArgs != "" {
				if args != "" {
					args += ", "
				}
				args += factoryArgs
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
		if factoryArgs != "" {
			ctorArgs = append(ctorArgs, factoryArgs)
		}
		return fmt.Sprintf("((%s) => new %s(%s))(%s)",
			strings.Join(params, ", "), class, strings.Join(ctorArgs, ", "), strings.Join(values, ", ")), nil
	case *ir.Closure:
		return p.printClosure(n)
	case *ir.MethodValue:
		return p.printMethodValue(n)
	case *ir.FuncRef:
		// A method expression referencing an ordinary class member has
		// no free-function symbol: the reference is a receiver-first
		// adapter arrow built from the expression's own function type.
		if dollar := strings.Index(n.Name, "$"); dollar > 0 && n.T.Kind == ir.KindFunc && n.T.Sig != nil && len(n.T.Sig.Params) > 0 {
			typeName, member := n.Name[:dollar], n.Name[dollar+1:]
			if p.module.MethodEmissionFor(goid.Method(n.Pkg, typeName, member)) == plan.MethodOrdinaryNilChecked {
				recvType := n.T.Sig.Params[0]
				recvSpelled, err := p.tsType(recvType)
				if err != nil {
					return "", err
				}
				params := []string{"$r: " + recvSpelled}
				names := []string{}
				for i, param := range n.T.Sig.Params[1:] {
					spelled, err := p.tsType(param)
					if err != nil {
						return "", err
					}
					name := fmt.Sprintf("$a%d", i)
					params = append(params, name+": "+spelled)
					names = append(names, name)
				}
				result, err := p.tsFuncResultType(n.T.Sig.Results)
				if err != nil {
					return "", err
				}
				receiver := "$r"
				if recvType.Kind == ir.KindPointer {
					receiver = "gort$.goNilCheck($r)"
				}
				// A family accessor renamed to avoid a field collision is
				// called by its renamed member name.
				memberName := tsName(member)
				familyRecv := recvType
				if familyRecv.Kind == ir.KindPointer && familyRecv.Elem != nil {
					familyRecv = *familyRecv.Elem
				}
				if _, _, ok := p.module.familyClassAndFamily(familyRecv); ok {
					memberName = p.module.familyMemberName(member)
				}
				return fmt.Sprintf("((%s): %s => %s.%s(%s))",
					joinComma(params), result, receiver, memberName, joinComma(names)), nil
			}
		}
		return p.module.symbol(n.Pkg, n.Name)
	case *ir.MethodExprAdapter:
		callee, err := p.module.symbol(n.Pkg, n.Method)
		if err != nil {
			return "", err
		}
		recvSpelled, err := p.tsType(ir.Type{Kind: ir.KindPointer, Elem: &n.RecvValue})
		if err != nil {
			return "", err
		}
		params := []string{"$r$: " + recvSpelled}
		names := []string{}
		for i, param := range n.Params {
			spelled, err := p.tsType(param.Type)
			if err != nil {
				return "", err
			}
			name := fmt.Sprintf("$a%d", i)
			params = append(params, name+": "+spelled)
			names = append(names, name)
		}
		result, err := p.tsFuncResultType(n.Results)
		if err != nil {
			return "", err
		}
		// (*T).ValueMethod: deref the pointer (nil panics) and forward;
		// the value-receiver function copies on entry.
		checkedRecv, err := p.nilCheckOf("$r$", ir.Type{Kind: ir.KindPointer, Elem: &n.RecvValue})
		if err != nil {
			return "", err
		}
		if typeName := n.RecvValue.Named; typeName != "" {
			if member, isMethod := strings.CutPrefix(n.Method, typeName+"$"); isMethod &&
				p.module.MethodEmissionFor(goid.Method(n.Pkg, typeName, member)) == plan.MethodOrdinaryNilChecked {
				return fmt.Sprintf("(%s): %s => %s.%s(%s)",
					joinComma(params), result, checkedRecv, tsName(member), joinComma(names)), nil
			}
		}
		operands := append([]string{checkedRecv}, names...)
		return fmt.Sprintf("(%s): %s => %s(%s)",
			joinComma(params), result, callee, joinComma(operands)), nil
	case *ir.TupleAdapt:
		return p.printTupleAdapt(n)
	case *ir.DynCall:
		fun, err := p.printExpr(n.Fun)
		if err != nil {
			return "", err
		}
		// A fully typed staged invocation preserving Go's order: the
		// callee operand and every argument evaluate as the arrow's
		// arguments (left to right), then a nil callee panics, then the
		// call runs. goPanicNil returns never, so the callee narrows to
		// its exact signature — no erased Function, no result cast.
		funcType := n.Fun.Type()
		if funcType.Sig == nil {
			return "", fmt.Errorf("dynamic call without a function signature")
		}
		spelledFun, err := p.tsType(funcType)
		if err != nil {
			return "", err
		}
		result, err := p.tsFuncResultType(n.Results)
		if err != nil {
			return "", err
		}
		// f(g()): the sole argument is a multi-result spread. The tuple
		// evaluates once (inside the arrow, after the callee is bound, so
		// operand-then-args order holds) and spreads into the call — the
		// const infers its exact tuple type, and the nil-callee panic still
		// follows argument evaluation, exactly Go.
		if inner, isSpread, err := p.spreadInner(n.Args); err != nil {
			return "", err
		} else if isSpread {
			ret := "return "
			if result == "void" {
				ret = ""
			}
			return fmt.Sprintf("((f$: %s): %s => { const t$ = %s; if (f$ === undefined) { gort$.goPanicNil(); } %sf$(...t$); })(%s)",
				spelledFun, result, inner, ret, fun), nil
		}
		arrowParams := []string{}
		callArgs := []string{}
		passed := []string{fun}
		arrowParams = append(arrowParams, "f$: "+spelledFun)
		for i, argument := range n.Args {
			printed, err := p.printExpr(argument)
			if err != nil {
				return "", err
			}
			spelled, err := p.tsType(argument.Type())
			if err != nil {
				return "", err
			}
			name := fmt.Sprintf("a%d$", i)
			arrowParams = append(arrowParams, name+": "+spelled)
			callArgs = append(callArgs, name)
			passed = append(passed, printed)
		}
		body := "if (f$ === undefined) { gort$.goPanicNil(); } return f$(" + joinComma(callArgs) + ");"
		if result == "void" {
			body = "if (f$ === undefined) { gort$.goPanicNil(); } f$(" + joinComma(callArgs) + ");"
		}
		return fmt.Sprintf("((%s): %s => { %s })(%s)",
			joinComma(arrowParams), result, body, joinComma(passed)), nil
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
			factories, err := p.zeroFactoryArgs(n.TypeArgs, nil)
			if err != nil {
				return "", err
			}
			if args == "" {
				args = factories
			} else {
				args += ", " + factories
			}
		}
		return fmt.Sprintf("%s%s(%s)", callee, typeArgs, args), nil
	case *ir.ParamCopy:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		op, has := p.cloneOps[n.Param]
		if !has {
			return "", fmt.Errorf("no clone operation in scope for type parameter %q", n.Param)
		}
		return op + "(" + x + ")", nil
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
			callee, err := p.module.symbol(t.Pkg, externCloneSymbol(t.Named))
			if err != nil {
				return "", err
			}
			return callee + "(" + x + ")", nil
		}
		return x + ".goClone$()", nil
	case *ir.StructZero:
		return p.zeroLiteral(n.T)
	case *ir.ExternZero:
		return p.zeroLiteral(n.T)
	case *ir.GenericFuncValue, *ir.ExternEqual, *ir.ExternFieldRead, *ir.ExternToOwned, *ir.ExternLit, *ir.PtrElemRef:
		return p.printExternBridgeExpr(e)
	case *ir.ParamEqual:
		left, err := p.printExpr(n.L)
		if err != nil {
			return "", err
		}
		right, err := p.printExpr(n.R)
		if err != nil {
			return "", err
		}
		op, has := p.eqOps[n.Param]
		if !has {
			return "", fmt.Errorf("no equality operation in scope for type parameter %q", n.Param)
		}
		call := op + "(" + left + ", " + right + ")"
		if n.Negate {
			return "(!" + call + ")", nil
		}
		return call, nil
	case *ir.ParamZero:
		return p.zeroLiteral(n.T)
	case *ir.ExternVar:
		dot := strings.LastIndex(n.ID, ".")
		callee, err := p.module.symbol(n.ID[:dot], externVarSymbol(n.ID[dot+1:]))
		if err != nil {
			return "", err
		}
		return callee + "()", nil
	case *ir.ExternalMethodCall:
		recv, err := p.printExpr(n.Recv)
		if err != nil {
			return "", err
		}
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(n.Pkg, externMethodSymbol(n.TypeName, n.Method))
		if err != nil {
			return "", err
		}
		operands := recv
		if args != "" {
			operands += ", " + args
		}
		return fmt.Sprintf("%s(%s)", callee, operands), nil
	case *ir.AddrOf:
		// The pointer to an addressable struct value is the instance.
		return p.printExpr(n.X)
	case *ir.FieldCellRef:
		return p.printFieldCellRef(n)
	case *ir.Deref:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		// Identity carriers (structs, arrays, external handles) ARE their
		// pointer; cell carriers read through the cell.
		checked, err := p.nilCheckOf(x, n.X.Type())
		if err != nil {
			return "", err
		}
		if n.X.Type().Elem != nil && n.X.Type().Elem.Kind == ir.KindIface && n.X.Type().Elem.TypeParamName != "" {
			if p.ptrSplit {
				if p.familyPtrCell {
					return checked + ".v", nil
				}
				// The object family: the pointer IS the instance.
				return checked, nil
			}
			// Unreachable by construction: the IR rejects dereference of a
			// pointer-to-type-parameter (its representation is opaque
			// inside the generic body).
			return "", fmt.Errorf("dereference of pointer-to-type-parameter reached emission")
		}
		switch n.T.Kind {
		case ir.KindStruct, ir.KindArray, ir.KindExternal:
			return checked, nil
		}
		return checked + ".v", nil
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
	}
	return "", fmt.Errorf("no emission for IR expression %T", e)
}

// castResults types a dynamically dispatched call's unknown result with
// an erasable cast.
