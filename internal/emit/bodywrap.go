// Function-body wrappers: the named-exit label (deferred mutations of
// named results reach the trailing return) and the defer-stack
// try/finally.
package emit

import (
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printNamedExitBody prints a function's body with the named-exit
// wrapper when it has one: the body runs inside a fn$ label (returns
// lowered to breaks), and ONE trailing return reads the named locals
// AFTER every deferred mutation.
func (p *printer) printNamedExitBody(function *ir.Func) error {
	if !function.NamedExit {
		return p.printDeferWrappedBody(function.Body, function.UsesDeferStack)
	}
	// The body's leading statement is the named-results declaration
	// (prepended at build): it hoists ABOVE the label so the trailing
	// return still sees the locals.
	body := function.Body
	if len(body.Stmts) > 0 {
		if decl, isDecl := body.Stmts[0].(*ir.DeclStmt); isDecl {
			if err := p.printStmt(decl); err != nil {
				return err
			}
			body = &ir.Block{Stmts: body.Stmts[1:]}
		}
	}
	p.line("fn$: {")
	p.indent++
	if err := p.printDeferWrappedBody(body, function.UsesDeferStack); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	names := make([]string, 0, len(function.Results))
	for _, result := range function.Results {
		names = append(names, tsName(result.Name))
	}
	if len(names) == 1 {
		p.line("return %s;", names[0])
	} else {
		p.line("return [%s];", strings.Join(names, ", "))
	}
	return nil
}

// printDeferWrappedBody prints a function body, wrapping it — when the
// body uses the defer stack — in one try/finally that drains deferred
// calls in LIFO order at every exit.
func (p *printer) printDeferWrappedBody(body *ir.Block, usesDeferStack bool) error {
	if !usesDeferStack {
		return p.printBlockBody(body)
	}
	// Go's defer/panic drain: every deferred call runs at function exit,
	// LIFO; a panic from one defer replaces the in-flight panic but the
	// remaining (older) defers still run, and the last surviving panic
	// propagates. Normal returns run the same drain via finally.
	p.line("const _ds$: (() => void)[] = [];")
	p.line("let _dp$: { readonly v: unknown } | undefined = undefined;")
	p.line("try {")
	p.indent++
	if err := p.printBlockBody(body); err != nil {
		return err
	}
	p.indent--
	p.line("} catch (_de$) {")
	p.indent++
	p.line("_dp$ = { v: _de$ };")
	p.indent--
	p.line("} finally {")
	p.indent++
	p.line("for (let _di$ = _ds$.length - 1; _di$ >= 0; _di$--) {")
	p.indent++
	p.line("try {")
	p.indent++
	p.line("(_ds$[_di$] as () => void)();")
	p.indent--
	p.line("} catch (_de$) {")
	p.indent++
	p.line("_dp$ = { v: _de$ };")
	p.indent--
	p.line("}")
	p.indent--
	p.line("}")
	p.line("if (_dp$ !== undefined) { throw _dp$.v; }")
	p.indent--
	p.line("}")
	return nil
}
