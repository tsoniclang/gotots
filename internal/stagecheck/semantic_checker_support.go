package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (
	verifier *checkerSemanticVerifier,
) deriveIndependentCheckerSupport() error {
	return verifier.view.VisitDefinitions(func(
		identifier *ast.Ident,
		object types.Object,
	) error {
		occurrence, present := verifier.occurrenceID(
			identifier,
		)
		if !present {
			if verifier.index.CheckedViewOnly(identifier) {
				return nil
			}
			if independentCheckerSupportObject(object, false) {
				return fmt.Errorf(
					"checker support object %s (%T) has no canonical occurrence",
					object.Name(),
					object,
				)
			}
			return nil
		}
		if verifier.independentIntrinsicContractBinding(occurrence) {
			return nil
		}
		retained := verifier.expected.hasOccurrence(occurrence)
		if !independentCheckerSupportObject(object, retained) {
			return nil
		}
		if existing := verifier.checkerSourceByObject[object]; !existing.IsZero() &&
			existing != occurrence {
			return fmt.Errorf(
				"checker object %s has definition occurrences %s and %s",
				object.Name(), existing, occurrence,
			)
		}
		verifier.checkerSourceByObject[object] = occurrence
		return nil
	})
}

func independentCheckerSupportObject(
	object types.Object,
	retained bool,
) bool {
	if _, local := independentLocalDeclarationClass(object); local {
		return true
	}
	if !independentLexicalBinding(object) {
		return false
	}
	switch object.(type) {
	case *types.Label, *types.PkgName:
		return retained
	default:
		return true
	}
}

type independentBindingOrderEntry struct {
	object types.Object
	anchor identity.OccurrenceID
	scope  identity.OccurrenceID
	role   identity.SemanticBindingRole
}

func (
	verifier *checkerSemanticVerifier,
) independentCompleteBindingOrdinals(
	scopeOwners map[*types.Scope]identity.OccurrenceID,
) (map[types.Object]int, error) {
	entries := map[types.Object]independentBindingOrderEntry{}
	admit := func(entry independentBindingOrderEntry) error {
		if entry.object == nil ||
			entry.anchor.IsZero() ||
			entry.scope.IsZero() ||
			!entry.role.Valid() {
			return fmt.Errorf(
				"independent binding order requires object, anchor, scope, and role",
			)
		}
		if existing, present := entries[entry.object]; present &&
			existing != entry {
			return fmt.Errorf(
				"checker binding %s has conflicting independent-order evidence",
				entry.object.Name(),
			)
		}
		entries[entry.object] = entry
		return nil
	}
	for object, source := range verifier.checkerSourceByObject {
		if !independentLexicalBinding(object) {
			continue
		}
		if verifier.independentIntrinsicContractBinding(source) {
			continue
		}
		if _, label := object.(*types.Label); label &&
			verifier.expected.occurrence(source).ID().IsZero() {
			continue
		}
		role := verifier.independentBindingRole(object, source)
		scope, err := verifier.independentBindingScope(
			object, source, scopeOwners,
		)
		if err != nil {
			return nil, err
		}
		if err := admit(independentBindingOrderEntry{
			object: object,
			anchor: source,
			scope:  scope,
			role:   role,
		}); err != nil {
			return nil, err
		}
	}
	if err := verifier.view.VisitImplicits(func(
		node ast.Node,
		object types.Object,
	) error {
		if !independentLexicalBinding(object) {
			return nil
		}
		if variable, field := object.(*types.Var); field &&
			variable.IsField() {
			return nil
		}
		anchor, present := verifier.occurrenceID(node)
		if !present {
			return nil
		}
		if verifier.independentIntrinsicContractBinding(anchor) {
			return nil
		}
		role := independentImplicitBindingRole(
			node,
			object,
			verifier.independentBindingRole(object, anchor),
		)
		if !role.Valid() {
			return nil
		}
		scope, err := verifier.independentBindingScope(
			object, anchor, scopeOwners,
		)
		if err != nil {
			return err
		}
		return admit(independentBindingOrderEntry{
			object: object,
			anchor: anchor,
			scope:  scope,
			role:   role,
		})
	}); err != nil {
		return nil, err
	}
	groups := map[checkerBindingGroup][]independentBindingOrderEntry{}
	for _, entry := range entries {
		key := checkerBindingGroup{scope: entry.scope, role: entry.role}
		groups[key] = append(groups[key], entry)
	}
	out := map[types.Object]int{}
	for _, group := range groups {
		sort.Slice(group, func(left, right int) bool {
			return group[left].anchor.Compare(
				group[right].anchor,
			) < 0
		})
		for ordinal, entry := range group {
			if ordinal > 0 &&
				group[ordinal-1].anchor == entry.anchor {
				return nil, fmt.Errorf(
					"checker bindings %s and %s have indistinguishable independent-order evidence",
					group[ordinal-1].object.Name(),
					entry.object.Name(),
				)
			}
			out[entry.object] = ordinal
		}
	}
	return out, nil
}

func (expected semanticPackageExpectation) localBinding(
	record semantic.Binding,
) bool {
	if !record.Source().IsZero() &&
		expected.localFiles[record.Source().Span().File()] {
		return true
	}
	if !record.ID().Owner().IsZero() &&
		expected.localFiles[record.ID().Owner().Span().File()] {
		return true
	}
	_, local := expected.definitions[record.Definition()]
	return !record.Definition().IsZero() && local
}
