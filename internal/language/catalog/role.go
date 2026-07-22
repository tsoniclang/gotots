package catalog

import "fmt"

// Role is the closed grammatical role a parent assigns to one child edge.
// Values are explicit and permanent; TestRoleIDsArePinned freezes the mapping.
// A child never infers its role by inspecting its parent or source text — the
// role arrives with the edge.
type Role uint16

// Explicit, permanent role identities. Do not renumber; append only.
const (
	RoleInvalid Role = 0

	RoleDocumentation         Role = 1
	RoleTrailingDocumentation Role = 2
	RoleCommentText           Role = 3
	RolePackageName           Role = 4
	RoleDeclaration           Role = 5
	RoleDeclarationName       Role = 6
	RoleTypeExpression        Role = 7
	RoleFieldTag              Role = 8
	RoleFieldGroup            Role = 9
	RoleConstructedType       Role = 10
	RoleCompositeElement      Role = 11
	RoleOperand               Role = 12
	RoleSelectorBase          Role = 13
	RoleSelectedName          Role = 14
	RoleIndexedOperand        Role = 15
	RoleIndex                 Role = 16
	RoleSlicedOperand         Role = 17
	RoleSliceBound            Role = 18
	RoleAssertedValue         Role = 19
	RoleAssertedType          Role = 20
	RoleCallee                Role = 21
	RoleCallArgument          Role = 22
	RoleLeftOperand           Role = 23
	RoleRightOperand          Role = 24
	RoleElementKey            Role = 25
	RoleElementValue          Role = 26
	RoleArrayLength           Role = 27
	RoleElementType           Role = 28
	RoleStructFields          Role = 29
	RoleTypeParameters        Role = 30
	RoleParameters            Role = 31
	RoleResults               Role = 32
	RoleInterfaceMethods      Role = 33
	RoleKeyType               Role = 34
	RoleValueType             Role = 35
	RoleLabelDeclaration      Role = 36
	RoleLabeledStatement      Role = 37
	RoleStatementExpression   Role = 38
	RoleChannelOperand        Role = 39
	RoleSentValue             Role = 40
	RoleAssignablePlace       Role = 41
	RoleAssignmentTarget      Role = 42
	RoleAssignedValue         Role = 43
	RoleSpawnedCall           Role = 44
	RoleDeferredCall          Role = 45
	RoleReturnValue           Role = 46
	RoleLabelReference        Role = 47
	RoleStatement             Role = 48
	RoleInitStatement         Role = 49
	RoleCondition             Role = 50
	RoleBody                  Role = 51
	RoleElseBranch            Role = 52
	RoleCaseValue             Role = 53
	RoleSwitchTag             Role = 54
	RoleTypeSwitchGuard       Role = 55
	RoleCommStatement         Role = 56
	RolePostStatement         Role = 57
	RoleRangeKey              Role = 58
	RoleRangeValue            Role = 59
	RoleRangeOperand          Role = 60
	RoleImportAlias           Role = 61
	RoleImportPath            Role = 62
	RoleInitializerValue      Role = 63
	RoleReceiver              Role = 64
	RoleFunctionSignature     Role = 65
	RoleFunctionBody          Role = 66
	RoleSpecification         Role = 67

	// roleCount is the highest assigned identity; append-only.
	roleCount = 67
)

var roleNames = [roleCount + 1]string{
	RoleDocumentation: "documentation", RoleTrailingDocumentation: "trailing-documentation",
	RoleCommentText: "comment-text", RolePackageName: "package-name",
	RoleDeclaration: "declaration", RoleDeclarationName: "declaration-name",
	RoleTypeExpression: "type-expression", RoleFieldTag: "field-tag",
	RoleFieldGroup: "field-group", RoleConstructedType: "constructed-type",
	RoleCompositeElement: "composite-element", RoleOperand: "operand",
	RoleSelectorBase: "selector-base", RoleSelectedName: "selected-name",
	RoleIndexedOperand: "indexed-operand", RoleIndex: "index",
	RoleSlicedOperand: "sliced-operand", RoleSliceBound: "slice-bound",
	RoleAssertedValue: "asserted-value", RoleAssertedType: "asserted-type",
	RoleCallee: "callee", RoleCallArgument: "call-argument",
	RoleLeftOperand: "left-operand", RoleRightOperand: "right-operand",
	RoleElementKey: "element-key", RoleElementValue: "element-value",
	RoleArrayLength: "array-length", RoleElementType: "element-type",
	RoleStructFields: "struct-fields", RoleTypeParameters: "type-parameters",
	RoleParameters: "parameters", RoleResults: "results",
	RoleInterfaceMethods: "interface-methods", RoleKeyType: "key-type",
	RoleValueType: "value-type", RoleLabelDeclaration: "label-declaration",
	RoleLabeledStatement: "labeled-statement", RoleStatementExpression: "statement-expression",
	RoleChannelOperand: "channel-operand", RoleSentValue: "sent-value",
	RoleAssignablePlace: "assignable-place", RoleAssignmentTarget: "assignment-target",
	RoleAssignedValue: "assigned-value", RoleSpawnedCall: "spawned-call",
	RoleDeferredCall: "deferred-call", RoleReturnValue: "return-value",
	RoleLabelReference: "label-reference", RoleStatement: "statement",
	RoleInitStatement: "init-statement", RoleCondition: "condition",
	RoleBody: "body", RoleElseBranch: "else-branch",
	RoleCaseValue: "case-value", RoleSwitchTag: "switch-tag",
	RoleTypeSwitchGuard: "type-switch-guard", RoleCommStatement: "comm-statement",
	RolePostStatement: "post-statement", RoleRangeKey: "range-key",
	RoleRangeValue: "range-value", RoleRangeOperand: "range-operand",
	RoleImportAlias: "import-alias", RoleImportPath: "import-path",
	RoleInitializerValue: "initializer-value", RoleReceiver: "receiver",
	RoleFunctionSignature: "function-signature", RoleFunctionBody: "function-body",
	RoleSpecification: "specification",
}

// Valid reports whether r names a role in the catalog.
func (r Role) Valid() bool { return r >= 1 && r <= roleCount }

// String renders r for diagnostics and reports.
func (r Role) String() string {
	if r.Valid() && roleNames[r] != "" {
		return roleNames[r]
	}
	return fmt.Sprintf("catalog.Role(%d)", uint16(r))
}

// AllRoles returns every valid Role in ascending identity order.
func AllRoles() []Role {
	roles := make([]Role, 0, roleCount)
	for id := 1; id <= roleCount; id++ {
		roles = append(roles, Role(id))
	}
	return roles
}
