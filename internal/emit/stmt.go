package emit

import (
	"fmt"
	"go/token"

	"github.com/tsoniclang/gotots/internal/ir"
)

func (p *printer) printBlockBody(block *ir.Block) error {
	for _, stmt := range block.Stmts {
		if err := p.printStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (p *printer) printStmt(stmt ir.Stmt) error {
	switch n := stmt.(type) {
	case *ir.Block:
		p.line("{")
		p.indent++
		if err := p.printBlockBody(n); err != nil {
			return err
		}
		p.indent--
		p.line("}")
		return nil

	case *ir.DeclStmt:
		return p.printDecl(n)

	case *ir.AssignStmt:
		return p.printAssign(n)

	case *ir.IfStmt:
		return p.printIf(n)

	case *ir.ForStmt:
		return p.printFor(n)

	case *ir.ReturnStmt:
		return p.printReturn(n)

	case *ir.ExprStmt:
		call, err := printExpr(n.Call)
		if err != nil {
			return err
		}
		p.line("%s;", call)
		return nil

	case *ir.BranchStmt:
		if n.Tok == token.BREAK {
			p.line("break;")
		} else {
			p.line("continue;")
		}
		return nil
	}
	return fmt.Errorf("no emission for IR statement %T", stmt)
}

func (p *printer) printDecl(n *ir.DeclStmt) error {
	if n.CallValues != nil {
		call, err := printExpr(n.CallValues)
		if err != nil {
			return err
		}
		tuple := p.temp()
		p.line("const %s = %s;", tuple, call)
		for i, name := range n.Names {
			spelled, err := tsType(n.Types[i])
			if err != nil {
				return err
			}
			p.line("let %s: %s = %s[%d];", name, spelled, tuple, i)
		}
		return nil
	}
	for i, name := range n.Names {
		spelled, err := tsType(n.Types[i])
		if err != nil {
			return err
		}
		value, err := printExpr(n.Values[i])
		if err != nil {
			return err
		}
		p.line("let %s: %s = %s;", name, spelled, value)
	}
	return nil
}

func (p *printer) printAssign(n *ir.AssignStmt) error {
	if n.CallValues != nil {
		call, err := printExpr(n.CallValues)
		if err != nil {
			return err
		}
		tuple := p.temp()
		p.line("const %s = %s;", tuple, call)
		for i, target := range n.Targets {
			p.line("%s = %s[%d];", target, tuple, i)
		}
		return nil
	}
	if len(n.Targets) == 1 {
		value, err := printExpr(n.Values[0])
		if err != nil {
			return err
		}
		p.line("%s = %s;", n.Targets[0], value)
		return nil
	}
	// Simultaneous assignment: every right-hand value is evaluated into a
	// temporary in source order before any store.
	temps := make([]string, len(n.Values))
	for i, value := range n.Values {
		printed, err := printExpr(value)
		if err != nil {
			return err
		}
		temps[i] = p.temp()
		p.line("const %s = %s;", temps[i], printed)
	}
	for i, target := range n.Targets {
		p.line("%s = %s;", target, temps[i])
	}
	return nil
}

func (p *printer) printIf(n *ir.IfStmt) error {
	// A Go if-init introduces its own scope covering both branches.
	if n.Init != nil {
		p.line("{")
		p.indent++
		if err := p.printStmt(n.Init); err != nil {
			return err
		}
	}
	cond, err := printExpr(n.Cond)
	if err != nil {
		return err
	}
	p.line("if (%s) {", cond)
	p.indent++
	if err := p.printBlockBody(n.Then); err != nil {
		return err
	}
	p.indent--
	switch elseStmt := n.Else.(type) {
	case nil:
		p.line("}")
	case *ir.Block:
		p.line("} else {")
		p.indent++
		if err := p.printBlockBody(elseStmt); err != nil {
			return err
		}
		p.indent--
		p.line("}")
	case *ir.IfStmt:
		// Emit as a nested else-block; init scoping stays correct.
		p.line("} else {")
		p.indent++
		if err := p.printStmt(elseStmt); err != nil {
			return err
		}
		p.indent--
		p.line("}")
	default:
		return fmt.Errorf("no emission for else branch %T", n.Else)
	}
	if n.Init != nil {
		p.indent--
		p.line("}")
	}
	return nil
}

// printFor emits the classic three-clause loop as a JS for header so that
// continue still executes the post statement (a body-tail post would be
// skipped by continue).
func (p *printer) printFor(n *ir.ForStmt) error {
	init, err := p.forClause(n.Init, true)
	if err != nil {
		return err
	}
	cond := ""
	if n.Cond != nil {
		cond, err = printExpr(n.Cond)
		if err != nil {
			return err
		}
	}
	post, err := p.forClause(n.Post, false)
	if err != nil {
		return err
	}
	p.line("for (%s; %s; %s) {", init, cond, post)
	p.indent++
	if err := p.printBlockBody(n.Body); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	return nil
}

// forClause renders an init/post statement as a for-header clause.
func (p *printer) forClause(stmt ir.Stmt, isInit bool) (string, error) {
	switch n := stmt.(type) {
	case nil:
		return "", nil
	case *ir.DeclStmt:
		if !isInit || n.CallValues != nil {
			return "", fmt.Errorf("declaration not expressible in this for-loop clause")
		}
		parts := make([]string, len(n.Names))
		for i, name := range n.Names {
			spelled, err := tsType(n.Types[i])
			if err != nil {
				return "", err
			}
			value, err := printExpr(n.Values[i])
			if err != nil {
				return "", err
			}
			parts[i] = fmt.Sprintf("%s: %s = %s", name, spelled, value)
		}
		return "let " + joinComma(parts), nil
	case *ir.AssignStmt:
		if n.CallValues != nil || len(n.Targets) != 1 {
			return "", fmt.Errorf("assignment not expressible in this for-loop clause")
		}
		value, err := printExpr(n.Values[0])
		if err != nil {
			return "", err
		}
		return n.Targets[0] + " = " + value, nil
	}
	return "", fmt.Errorf("statement %T not expressible in a for-loop clause", stmt)
}

func (p *printer) printReturn(n *ir.ReturnStmt) error {
	if n.CallValue != nil {
		call, err := printExpr(n.CallValue)
		if err != nil {
			return err
		}
		p.line("return %s;", call)
		return nil
	}
	switch len(n.Values) {
	case 0:
		p.line("return;")
	case 1:
		value, err := printExpr(n.Values[0])
		if err != nil {
			return err
		}
		p.line("return %s;", value)
	default:
		parts := make([]string, len(n.Values))
		for i, value := range n.Values {
			printed, err := printExpr(value)
			if err != nil {
				return err
			}
			parts[i] = printed
		}
		p.line("return [%s];", joinComma(parts))
	}
	return nil
}

func joinComma(parts []string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += ", "
		}
		out += part
	}
	return out
}
