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
	RoleFileDeclaration      Role = "file-declaration"
	RoleFunctionBody         Role = "function-body"
	RoleParameterType        Role = "parameter-type"
	RoleResultType           Role = "result-type"
	RoleBlockStatement       Role = "block-statement"
	RoleReturnResult         Role = "return-result"
	RoleBinaryLeft           Role = "binary-left"
	RoleBinaryRight          Role = "binary-right"
	RoleIndexOperand         Role = "index-operand"
	RoleIndexValue           Role = "index-value"
	RoleSliceOperand         Role = "slice-operand"
	RoleSliceLow             Role = "slice-low"
	RoleSliceHigh            Role = "slice-high"
	RoleParenthesizedValue   Role = "parenthesized-value"
	RoleLocalType            Role = "local-type"
	RoleLocalValue           Role = "local-value"
	RoleLocalDeclaration     Role = "local-declaration"
	RoleLocalConstantType    Role = "local-constant-type"
	RoleLocalConstantValue   Role = "local-constant-value"
	RoleAssignmentValue      Role = "assignment-value"
	RoleIfInitializer        Role = "if-initializer"
	RoleIfCondition          Role = "if-condition"
	RoleIfThen               Role = "if-then"
	RoleIfElse               Role = "if-else"
	RoleUnaryOperand         Role = "unary-operand"
	RoleCallCallee           Role = "call-callee"
	RoleCallArgument         Role = "call-argument"
	RoleCallArgumentType     Role = "call-argument-type"
	RoleFunctionLiteralBody  Role = "function-literal-body"
	RoleCallableParameter    Role = "callable-parameter-type"
	RoleCallableResult       Role = "callable-result-type"
	RoleNamedResultZero      Role = "named-result-zero"
	RoleExpressionStatement  Role = "expression-statement"
	RoleIntegerConstantType  Role = "integer-constant-type"
	RoleBooleanConstantType  Role = "boolean-constant-type"
	RolePackageConstantType  Role = "package-constant-type"
	RolePackageConstantValue Role = "package-constant-value"
	RolePackageVariableType  Role = "package-variable-type"
	RolePackageVariableZero  Role = "package-variable-zero"
	RolePackageVariableValue Role = "package-variable-value"
	RolePackageInitBody      Role = "package-init-body"
	RoleForInitializer       Role = "for-initializer"
	RoleForCondition         Role = "for-condition"
	RoleForPost              Role = "for-post"
	RoleForBody              Role = "for-body"
	RoleSwitchInitializer    Role = "switch-initializer"
	RoleSwitchTag            Role = "switch-tag"
	RoleSwitchClause         Role = "switch-clause"
	RoleSwitchCaseExpression Role = "switch-case-expression"
	RoleSwitchCaseStatement  Role = "switch-case-statement"
	RoleStructField          Role = "struct-field"
	RoleStructFieldType      Role = "struct-field-type"
	RoleStructZeroField      Role = "struct-zero-field"
	RoleStructCopyField      Role = "struct-copy-field"
	RoleStructAssignField    Role = "struct-assign-field"
	RoleStructEqualField     Role = "struct-equal-field"
	RoleCompositeElement     Role = "composite-element"
	RoleFieldReceiver        Role = "field-receiver"
	RoleAssignmentTarget     Role = "assignment-target"
	RoleReceiverType         Role = "receiver-type"
	RoleReceiverValue        Role = "receiver-value"
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
