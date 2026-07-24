package catalog

// IdentifierDefinitionCandidate reports the complete grammatical role class
// in which go/types may place an identifier in Info.Defs. Context-sensitive
// roles remain candidates here; the checker decides whether a definition
// actually exists.
func IdentifierDefinitionCandidate(role Role) bool {
	switch role {
	case RolePackageName,
		RoleDeclarationName,
		RoleAssignmentTarget,
		RoleRangeKey,
		RoleRangeValue,
		RoleImportAlias,
		RoleLabelDeclaration:
		return true
	default:
		return false
	}
}
