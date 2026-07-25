package catalog

import "testing"

func TestIdentifierDefinitionCandidateIsClosed(t *testing.T) {
	expected := map[Role]bool{
		RolePackageName:      true,
		RoleDeclarationName:  true,
		RoleAssignmentTarget: true,
		RoleRangeKey:         true,
		RoleRangeValue:       true,
		RoleImportAlias:      true,
		RoleLabelDeclaration: true,
	}
	for _, role := range AllRoles() {
		if IdentifierDefinitionCandidate(role) != expected[role] {
			t.Errorf(
				"IdentifierDefinitionCandidate(%s)=%t, want %t",
				role,
				IdentifierDefinitionCandidate(role),
				expected[role],
			)
		}
	}
}
