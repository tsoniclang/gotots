package frontend

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

func (index *objectIndex) indexCheckerSupport(
	view *source.TypeInfoView,
) error {
	if err := view.VisitDefinitions(func(
		identifier *ast.Ident,
		object types.Object,
	) error {
		index.work.CheckerDefinitionVisits++
		occurrence, present := index.input.index.OccurrenceID(
			identifier,
		)
		if !present {
			if index.input.index.CheckedViewOnly(identifier) {
				return nil
			}
			if index.checkerSupportObject(object, false) {
				return fmt.Errorf(
					"checker support object %s (%T) has no canonical occurrence",
					object.Name(),
					object,
				)
			}
			return nil
		}
		if index.intrinsicContractBinding(occurrence) {
			return nil
		}
		retained := index.input.occurrence(occurrence) != nil
		if !index.checkerSupportObject(object, retained) {
			return nil
		}
		if existing := index.checkerSourceByObject[object]; !existing.IsZero() &&
			existing != occurrence {
			return fmt.Errorf(
				"checker object %s has direct definition occurrences %s and %s",
				object.Name(), existing, occurrence,
			)
		}
		index.checkerSourceByObject[object] = occurrence
		return nil
	}); err != nil {
		return err
	}
	if err := view.VisitScopes(func(
		node ast.Node,
		scope *types.Scope,
	) error {
		index.work.CheckerScopeEvidenceVisits++
		occurrence, present := index.input.index.OccurrenceID(
			node,
		)
		if !present {
			return nil
		}
		if existing := index.scopeOwners[scope]; !existing.IsZero() &&
			existing != occurrence {
			return fmt.Errorf(
				"checker scope has direct owner occurrences %s and %s",
				existing, occurrence,
			)
		}
		index.scopeOwners[scope] = occurrence
		return nil
	}); err != nil {
		return err
	}
	return index.indexUnnamedSignatureBindings()
}

func (index *objectIndex) checkerSupportObject(
	object types.Object,
	retained bool,
) bool {
	if _, local := localDeclarationClass(object); local {
		return true
	}
	if !index.isBinding(object) {
		return false
	}
	switch object.(type) {
	case *types.Label, *types.PkgName:
		return retained
	default:
		return true
	}
}

func (index *objectIndex) supportRole(
	id identity.OccurrenceID,
) (catalog.Role, bool) {
	if record := index.input.occurrence(id); record != nil {
		return record.occurrence.Role(), true
	}
	return index.input.index.StructuralSupport(id)
}

func (index *objectIndex) supportDefinition(
	source identity.OccurrenceID,
	scope identity.OccurrenceID,
) identity.DefinitionID {
	if record := index.input.occurrence(source); record != nil &&
		!record.owner.IsZero() {
		return record.owner
	}
	if definition, present := index.input.index.OccurrenceDefinition(
		source,
	); present {
		return definition
	}
	if record := index.input.occurrence(scope); record != nil {
		return record.owner
	}
	definition, _ := index.input.index.OccurrenceDefinition(scope)
	return definition
}

type completeBindingOrderEntry struct {
	object types.Object
	anchor identity.OccurrenceID
	scope  identity.OccurrenceID
	role   identity.SemanticBindingRole
}

func (index *objectIndex) assignCompleteBindingOrdinals(
	view *source.TypeInfoView,
) error {
	entries := map[types.Object]completeBindingOrderEntry{}
	admit := func(entry completeBindingOrderEntry) error {
		if entry.object == nil ||
			entry.anchor.IsZero() ||
			entry.scope.IsZero() ||
			!entry.role.Valid() {
			return fmt.Errorf(
				"complete binding order requires object, anchor, scope, and role",
			)
		}
		if existing, present := entries[entry.object]; present &&
			existing != entry {
			return fmt.Errorf(
				"checker binding %s has conflicting complete-order evidence",
				entry.object.Name(),
			)
		}
		entries[entry.object] = entry
		return nil
	}
	for object, source := range index.checkerSourceByObject {
		if !index.isBinding(object) {
			continue
		}
		if index.intrinsicContractBinding(source) {
			continue
		}
		if _, label := object.(*types.Label); label &&
			index.input.occurrence(source) == nil {
			continue
		}
		role := index.bindingRole(object, source)
		scope, err := index.bindingScope(object, source)
		if err != nil {
			return err
		}
		if err := admit(completeBindingOrderEntry{
			object: object,
			anchor: source,
			scope:  scope,
			role:   role,
		}); err != nil {
			return err
		}
	}
	if err := view.VisitImplicits(func(
		node ast.Node,
		object types.Object,
	) error {
		index.work.CheckerImplicitEvidenceVisits++
		if !index.isBinding(object) {
			return nil
		}
		if variable, field := object.(*types.Var); field &&
			variable.IsField() {
			return nil
		}
		anchor, present := index.input.index.OccurrenceID(node)
		if !present {
			return nil
		}
		if index.intrinsicContractBinding(anchor) {
			return nil
		}
		context := index.contexts.context(anchor)
		role := index.implicitBindingRole(node, object, context)
		if !role.Valid() {
			return nil
		}
		scope, err := index.bindingScope(object, anchor)
		if err != nil {
			return err
		}
		return admit(completeBindingOrderEntry{
			object: object,
			anchor: anchor,
			scope:  scope,
			role:   role,
		})
	}); err != nil {
		return err
	}
	type groupKey struct {
		scope identity.OccurrenceID
		role  identity.SemanticBindingRole
	}
	groups := map[groupKey][]completeBindingOrderEntry{}
	for _, entry := range entries {
		key := groupKey{scope: entry.scope, role: entry.role}
		groups[key] = append(groups[key], entry)
	}
	for _, group := range groups {
		sort.Slice(group, func(left, right int) bool {
			return group[left].anchor.Compare(
				group[right].anchor,
			) < 0
		})
		for ordinal, entry := range group {
			if ordinal > 0 &&
				group[ordinal-1].anchor.Compare(entry.anchor) == 0 {
				return fmt.Errorf(
					"checker bindings %s and %s have indistinguishable complete-order evidence",
					group[ordinal-1].object.Name(),
					entry.object.Name(),
				)
			}
			index.bindingOrdinals[entry.object] = ordinal
		}
	}
	return nil
}

func (index *objectIndex) implicitBindingRole(
	node ast.Node,
	object types.Object,
	context occurrenceContext,
) identity.SemanticBindingRole {
	role := context.bindingRole
	switch object.(type) {
	case *types.PkgName:
		role = identity.SemanticBindingImport
	case *types.Var:
		if _, field := node.(*ast.Field); !field {
			role = identity.SemanticBindingTypeSwitch
		}
	}
	return role
}

func supportBindingRole(
	role catalog.Role,
) identity.SemanticBindingRole {
	switch role {
	case catalog.RoleTypeParameters:
		return identity.SemanticBindingTypeParameter
	case catalog.RoleParameters:
		return identity.SemanticBindingParameter
	case catalog.RoleResults:
		return identity.SemanticBindingResult
	case catalog.RoleReceiver:
		return identity.SemanticBindingReceiver
	case catalog.RoleImportAlias:
		return identity.SemanticBindingImport
	case catalog.RoleRangeKey, catalog.RoleRangeValue:
		return identity.SemanticBindingRange
	case catalog.RoleLabelDeclaration:
		return identity.SemanticBindingLabel
	default:
		return identity.SemanticBindingLocal
	}
}
