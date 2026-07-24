package frontend

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func (index *objectIndex) bindIntrinsicDefinitionSources() error {
	if index.input.id.ImportPath() != "unsafe" {
		return nil
	}
	seen := map[catalog.UnsafeMemberKind]bool{}
	for _, occurrenceID := range index.input.order {
		index.work.IntrinsicOccurrenceVisits++
		record := index.input.occurrence(occurrenceID)
		identifier, identifierNode := record.node.(*ast.Ident)
		if !identifierNode ||
			record.occurrence.Role() !=
				catalog.RoleDeclarationName {
			continue
		}
		member := catalog.UnsafeMemberByName(identifier.Name)
		if !member.Valid() {
			continue
		}
		if seen[member] {
			return fmt.Errorf(
				"unsafe member %s has duplicate declaration occurrences",
				member,
			)
		}
		seen[member] = true
		object := intrinsicDefinitionObject(
			index.input, member.Name(),
		)
		if object == nil {
			return fmt.Errorf(
				"unsafe member %s has no cataloged checker object",
				member,
			)
		}
		if err := index.bindObjectSource(
			object, occurrenceID,
		); err != nil {
			return err
		}
		if member.Class() == catalog.UnsafeMemberClassBuiltin {
			if record.owner.IsZero() {
				return fmt.Errorf(
					"unsafe builtin %s has no Stage-1 definition",
					member,
				)
			}
			if err := index.bindObjectDefinition(
				object, record.owner,
			); err != nil {
				return err
			}
		}
	}
	for _, member := range catalog.AllUnsafeMembers() {
		if !seen[member] {
			return fmt.Errorf(
				"unsafe member %s has no source declaration occurrence",
				member,
			)
		}
	}
	return nil
}

func intrinsicDefinitionObject(
	input *packageInput,
	name string,
) types.Object {
	if input == nil ||
		input.loaded == nil ||
		input.loaded.Types() == nil ||
		input.id.ImportPath() != "unsafe" {
		return nil
	}
	member := catalog.UnsafeMemberByName(name)
	if !member.Valid() {
		return nil
	}
	object := input.loaded.Types().Scope().Lookup(member.Name())
	switch member.Class() {
	case catalog.UnsafeMemberClassBuiltin:
		if _, valid := object.(*types.Builtin); !valid {
			return nil
		}
	case catalog.UnsafeMemberClassType:
		if _, valid := object.(*types.TypeName); !valid {
			return nil
		}
	default:
		return nil
	}
	return object
}
