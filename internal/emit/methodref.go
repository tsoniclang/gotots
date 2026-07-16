// Method-value emission: the receiver is captured (dereferenced and
// copied per its flavor) at evaluation inside an immediately-applied
// arrow, and each call forwards it to the generated receiver-first
// method function — or through dynamic interface dispatch.
package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

func (p *printer) printMethodValue(n *ir.MethodValue) (string, error) {
	recv, err := p.printExpr(n.Recv)
	if err != nil {
		return "", err
	}
	capture, recvSpelled, err := p.methodValueCapture(n, recv)
	if err != nil {
		return "", err
	}
	var params, args []string
	for i, param := range n.T.Sig.Params {
		spelled, err := p.tsType(param)
		if err != nil {
			return "", err
		}
		name := fmt.Sprintf("$a%d", i)
		params = append(params, name+": "+spelled)
		args = append(args, name)
	}
	result, err := p.tsFuncResultType(n.T.Sig.Results)
	if err != nil {
		return "", err
	}
	var call string
	if n.Iface {
		// Reuse the closed token-switch dispatch: the bound receiver $r
		// is the interface box, and the method value's parameters are the
		// call arguments.
		argExprs := make([]ir.Expr, len(args))
		for i := range args {
			argExprs[i] = &ir.ParamRef{Name: args[i], T: n.T.Sig.Params[i]}
		}
		synthetic := &ir.IfaceCall{
			Recv:    &ir.ParamRef{Name: "$r", T: n.Recv.Type()},
			Display: n.Method,
			Args:    argExprs,
			Results: n.Results,
		}
		call, err = p.printIfaceCall(synthetic)
		if err != nil {
			return "", err
		}
	} else {
		callee, err := p.module.symbol(n.Pkg, n.TypeName+"$"+n.Method)
		if err != nil {
			return "", err
		}
		call = callee + "(" + joinComma(append([]string{"$r"}, args...)) + ")"
	}
	body := call
	if result != "void" {
		body = "(" + call + ")"
	}
	return fmt.Sprintf("(($r: %s) => (%s): %s => %s)(%s)",
		recvSpelled, strings.Join(params, ", "), result, body, capture), nil
}

// methodValueCapture spells the receiver capture expression and its TS
// type: nil interfaces and pointer callers of value-receiver methods
// panic here (at evaluation), and value receivers capture a copy.
func (p *printer) methodValueCapture(n *ir.MethodValue, recv string) (string, string, error) {
	recvType := n.Recv.Type()
	if n.Iface {
		spelled, err := p.tsType(recvType)
		if err != nil {
			return "", "", err
		}
		checked, err := p.nilCheckOf(recv, n.Recv.Type())
		if err != nil {
			return "", "", err
		}
		return checked, spelled, nil
	}
	captured := recvType
	if n.NilCheckRecv {
		checked, err := p.nilCheckOf(recv, n.Recv.Type())
		if err != nil {
			return "", "", err
		}
		recv = checked
		captured = *recvType.Elem
	}
	spelled, err := p.tsType(captured)
	if err != nil {
		return "", "", err
	}
	if !n.PointerRecv {
		switch captured.Kind {
		case ir.KindStruct:
			recv += ".goClone$()"
		case ir.KindArray:
			cloneElem, err := p.arrayElemClone(*captured.Elem)
			if err != nil {
				return "", "", err
			}
			recv = "gosl$.goArrayClone(" + recv + ", " + cloneElem + ")"
		}
	}
	return recv, spelled, nil
}

// tsFuncResultType spells a function type's result list.
func (p *printer) tsFuncResultType(results []ir.Type) (string, error) {
	switch len(results) {
	case 0:
		return "void", nil
	case 1:
		return p.tsType(results[0])
	default:
		var parts []string
		for _, result := range results {
			spelled, err := p.tsType(result)
			if err != nil {
				return "", err
			}
			parts = append(parts, spelled)
		}
		return "readonly [" + strings.Join(parts, ", ") + "]", nil
	}
}
