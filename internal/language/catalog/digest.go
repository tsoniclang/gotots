package catalog

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
)

var structureDigest = sync.OnceValue(computeStructureDigest)

// StructureDigest is the canonical digest of the complete catalog structure:
// every pinned identity and name across every closed domain. Audit artifacts
// bind to it so a catalog change invalidates stored coverage evidence.
func StructureDigest() string {
	return structureDigest()
}

func computeStructureDigest() string {
	var parts []string
	for _, kind := range All() {
		parts = append(parts, fmt.Sprintf("kind:%d=%s:%s:%s", uint16(kind), kind.Name(), kind.Category(), kind.Disposition()))
	}
	for _, edge := range AllEdges() {
		parts = append(parts, fmt.Sprintf(
			"edge:%d=%s:%s:%v:%v",
			uint16(edge),
			edge.Name(),
			edge.Role(),
			edge.IsList(),
			edge.DefinitionEntry(),
		))
	}
	for _, kind := range All() {
		for _, scope := range []DefinitionScope{
			DefinitionScopePackage,
			DefinitionScopeExecutable,
		} {
			for _, declaration := range []TokenKind{
				TokenInvalid, TokenCONST, TokenIMPORT, TokenTYPE, TokenVAR,
			} {
				context, err := NewDefinitionContext(scope, declaration)
				if err != nil {
					panic(err)
				}
				for _, hasEntry := range []bool{false, true} {
					definition, present, err := DefinitionKind(
						kind, context, hasEntry,
					)
					if err != nil {
						panic(err)
					}
					parts = append(parts, fmt.Sprintf(
						"definition:%d:%d:%d:%t=%d:%t",
						uint16(kind),
						uint8(scope),
						uint16(declaration),
						hasEntry,
						uint8(definition),
						present,
					))
				}
			}
		}
	}
	for _, role := range AllRoles() {
		parts = append(parts, fmt.Sprintf("role:%d=%s", uint16(role), role.String()))
	}
	for _, variant := range AllVariants() {
		parts = append(parts, fmt.Sprintf("variant:%d=%s", uint16(variant), variant.String()))
	}
	for _, token := range AllTokens() {
		parts = append(parts, fmt.Sprintf("token:%d=%s:%s", uint16(token), token.ConstName(), token.Class()))
	}
	for _, pre := range AllPredeclared() {
		parts = append(parts, fmt.Sprintf("predeclared:%d=%s:%s", uint16(pre), pre.Name(), pre.Class()))
	}
	for _, directive := range AllDirectives() {
		parts = append(parts, fmt.Sprintf("directive:%d=%s:%s", uint16(directive), directive.Name(), directive.Disposition()))
	}
	for _, op := range AllImplicitOps() {
		parts = append(parts, fmt.Sprintf("implicit:%d=%s:%s", uint16(op), op.Name(), op.Owner()))
	}
	canonical := SelectedGoVersion + "|" + strings.Join(parts, "|")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(canonical)))
}
