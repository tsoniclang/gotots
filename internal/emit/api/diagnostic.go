package api

import (
	"fmt"
	"go/ast"
	"go/token"
)

type Category string

const (
	CategoryDeclaration Category = "declaration"
	CategoryStatement   Category = "statement"
	CategoryExpression  Category = "expression"
	CategoryType        Category = "type"
)

type Role string

const (
	RoleFileDeclaration     Role = "file-declaration"
	RoleFunctionBody        Role = "function-body"
	RoleParameterType       Role = "parameter-type"
	RoleResultType          Role = "result-type"
	RoleBlockStatement      Role = "block-statement"
	RoleReturnResult        Role = "return-result"
	RoleBinaryLeft          Role = "binary-left"
	RoleBinaryRight         Role = "binary-right"
	RoleLocalType           Role = "local-type"
	RoleLocalValue          Role = "local-value"
	RoleAssignmentValue     Role = "assignment-value"
	RoleIfCondition         Role = "if-condition"
	RoleIfThen              Role = "if-then"
	RoleIfElse              Role = "if-else"
	RoleUnaryOperand        Role = "unary-operand"
	RoleCallArgument        Role = "call-argument"
	RoleIntegerConstantType Role = "integer-constant-type"
	RoleForInitializer      Role = "for-initializer"
	RoleForCondition        Role = "for-condition"
	RoleForPost             Role = "for-post"
	RoleForBody             Role = "for-body"
)

type UnsupportedError struct {
	Category  Category
	Construct string
	Role      Role
	Position  token.Position
}

func (e *UnsupportedError) Error() string {
	location := e.Position.String()
	if !e.Position.IsValid() {
		location = "<unknown>"
	}
	return fmt.Sprintf(
		"unsupported Go %s %s in role %s at %s",
		e.Category,
		e.Construct,
		e.Role,
		location,
	)
}

func Unsupported(context Context, category Category, source ast.Node) *UnsupportedError {
	return &UnsupportedError{
		Category:  category,
		Construct: fmt.Sprintf("%T", source),
		Role:      context.Role(),
		Position:  context.FileSet().Position(source.Pos()),
	}
}

type ContextError struct {
	Reason string
}

func (e *ContextError) Error() string {
	return "create emission context: " + e.Reason
}

type InvariantError struct {
	Role   Role
	Reason string
}

func (e *InvariantError) Error() string {
	return fmt.Sprintf("emission invariant in role %s: %s", e.Role, e.Reason)
}

type NameError struct {
	Name   string
	Reason string
}

type PlacementError struct {
	ModulePath   string
	ExportedName string
	Reason       string
}

func (e *PlacementError) Error() string {
	return fmt.Sprintf(
		"place import %q from %q: %s",
		e.ExportedName,
		e.ModulePath,
		e.Reason,
	)
}

func (e *NameError) Error() string {
	if e.Name == "" {
		return "resolve target name: " + e.Reason
	}
	return fmt.Sprintf("resolve target name %q: %s", e.Name, e.Reason)
}
