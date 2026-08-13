package api

import "go/ast"

type DeferControl struct {
	stack  string
	static map[*ast.DeferStmt]string
}

func NewDeferControl(stack string) (DeferControl, error) {
	if stack == "" {
		return DeferControl{}, &InvariantError{
			Role:   RoleBlockStatement,
			Reason: "defer stack identity is empty",
		}
	}
	return DeferControl{stack: stack}, nil
}

func NewStaticDeferControl(
	bindings map[*ast.DeferStmt]string,
) (DeferControl, error) {
	if len(bindings) == 0 {
		return DeferControl{}, &InvariantError{
			Role:   RoleBlockStatement,
			Reason: "static defer bindings are empty",
		}
	}
	selected := make(map[*ast.DeferStmt]string, len(bindings))
	names := make(map[string]struct{}, len(bindings))
	for statement, name := range bindings {
		if statement == nil || statement.Call == nil || name == "" {
			return DeferControl{}, &InvariantError{
				Role:   RoleBlockStatement,
				Reason: "static defer binding is invalid",
			}
		}
		if _, duplicate := names[name]; duplicate {
			return DeferControl{}, &InvariantError{
				Role:   RoleBlockStatement,
				Reason: "static defer binding name is duplicated",
			}
		}
		selected[statement] = name
		names[name] = struct{}{}
	}
	return DeferControl{static: selected}, nil
}

func (c DeferControl) Valid() bool {
	return (c.stack != "") != (len(c.static) != 0)
}

func (c DeferControl) Stack() string {
	return c.stack
}

func (c DeferControl) StaticBinding(
	statement *ast.DeferStmt,
) (string, bool) {
	name, ok := c.static[statement]
	return name, ok
}
