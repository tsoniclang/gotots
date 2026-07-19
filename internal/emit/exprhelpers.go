// Call-result casting, closures, and argument spelling.
package emit

import (
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

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
	subPrinter := &printer{out: &sub, module: p.module, indent: p.indent + 1,
		zeroFactories: p.zeroFactories, eqOps: p.eqOps, cloneOps: p.cloneOps, setOps: p.setOps, keyOps: p.keyOps,
		slicePlans: p.slicePlans}
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
		if spread, isSpread := arg.(*ir.TupleVariadicSpread); isSpread {
			iife, err := p.printTupleVariadicSpread(spread)
			if err != nil {
				return "", err
			}
			parts[i] = "...(" + iife + ")"
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

// generatedIdentifier accepts only bare generated binding names — ASCII
// letters, digits, "$", and "_" — never expression text.
func generatedIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		ok := r == '$' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !ok {
			return false
		}
	}
	return true
}

// callFamilyEnc reports whether a generic call binds any HARD map-keyed
// position to a struct (or, inside an encoded-family emission, forwards
// a bare parameter at a hard position) — selecting the callee's "$ek"
// variant.
func callFamilyEnc(typeArgs []ir.Type, hardKeyed []bool, familyEnc bool) bool {
	for i, arg := range typeArgs {
		if i >= len(hardKeyed) || !hardKeyed[i] {
			continue
		}
		if arg.Kind == ir.KindStruct && arg.TypeParamName == "" {
			return true
		}
		if familyEnc && arg.TypeParamName != "" {
			return true
		}
	}
	return false
}
