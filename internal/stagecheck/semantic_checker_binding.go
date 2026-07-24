package stagecheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type checkerBindingCandidate struct {
	id         identity.SemanticBindingID
	object     types.Object
	anchor     identity.OccurrenceID
	source     identity.OccurrenceID
	scope      identity.OccurrenceID
	role       identity.SemanticBindingRole
	definition identity.DefinitionID
	typ        types.Type
	name       string
	ordinal    int
}

type checkerBindingGroup struct {
	scope identity.OccurrenceID
	role  identity.SemanticBindingRole
}

func (verifier *checkerSemanticVerifier) verifyBindings() error {
	if err := verifier.deriveIndependentUnnamedSignatureBindings(); err != nil {
		return err
	}
	candidates, err := verifier.independentBindingCandidates()
	if err != nil {
		return err
	}
	groups := map[checkerBindingGroup][]*checkerBindingCandidate{}
	for _, candidate := range candidates {
		key := checkerBindingGroup{
			scope: candidate.scope,
			role:  candidate.role,
		}
		groups[key] = append(groups[key], candidate)
	}
	ordinals, err := verifier.independentCompleteBindingOrdinals(
		verifier.scopeOwners,
	)
	if err != nil {
		return err
	}
	expected := map[identity.SemanticBindingID]*checkerBindingCandidate{}
	for _, group := range groups {
		sort.Slice(group, func(left, right int) bool {
			return group[left].anchor.Compare(
				group[right].anchor,
			) < 0
		})
		for _, candidate := range group {
			ordinal, present := ordinals[candidate.object]
			if !present {
				return fmt.Errorf(
					"checker binding %s has no independent complete-set ordinal",
					candidate.name,
				)
			}
			candidate.ordinal = ordinal
			id, err := identity.NewSemanticBindingID(
				candidate.scope,
				candidate.source,
				candidate.role,
				ordinal,
			)
			if err != nil {
				return err
			}
			if existing := expected[id]; existing != nil {
				return fmt.Errorf(
					"checker-derived binding identity %s is duplicated",
					id,
				)
			}
			candidate.id = id
			expected[id] = candidate
			if candidate.object != nil {
				if existing := verifier.bindingByObject[candidate.object]; !existing.IsZero() && existing != id {
					return fmt.Errorf(
						"checker object %s maps to bindings %s and %s",
						candidate.name, existing, id,
					)
				}
				verifier.bindingByObject[candidate.object] = id
			}
		}
	}
	verifier.bindings = expected
	for id, candidate := range expected {
		verifier.bindingsByDefinition[candidate.definition] = append(
			verifier.bindingsByDefinition[candidate.definition], id,
		)
	}
	for definition := range verifier.bindingsByDefinition {
		sort.Slice(
			verifier.bindingsByDefinition[definition],
			func(left, right int) bool {
				return verifier.bindingsByDefinition[definition][left].
					Compare(
						verifier.bindingsByDefinition[definition][right],
					) < 0
			},
		)
	}
	var missing, extra []string
	seen := map[identity.SemanticBindingID]bool{}
	actualCount := 0
	if err := verifier.visitBindings(func(
		record semantic.Binding,
	) error {
		candidate := expected[record.ID()]
		if candidate == nil {
			extra = append(extra, record.ID().String())
			return nil
		}
		actualCount++
		if seen[record.ID()] {
			extra = append(extra, record.ID().String())
			return nil
		}
		seen[record.ID()] = true
		if err := verifier.verifyBindingRecord(
			record, candidate,
		); err != nil {
			return fmt.Errorf("binding %s: %w", record.ID(), err)
		}
		verifier.bindingCaptures[record.ID()] = record.CapturedBy()
		return nil
	}); err != nil {
		return err
	}
	for id := range expected {
		if !seen[id] {
			missing = append(missing, id.String())
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) != 0 || len(extra) != 0 {
		return fmt.Errorf(
			"bindings checker-derived=%d semantic=%d; missing=%v extra=%v; missing-details=%v",
			len(expected),
			actualCount,
			missing,
			extra,
			verifier.missingBindingDetails(expected, seen),
		)
	}
	return nil
}

func (verifier *checkerSemanticVerifier) independentBindingCandidates() (
	[]*checkerBindingCandidate,
	error,
) {
	scopeOwners := verifier.scopeOwners
	if scopeOwners == nil {
		var err error
		scopeOwners, err = verifier.independentScopeOwners()
		if err != nil {
			return nil, err
		}
		verifier.scopeOwners = scopeOwners
	}
	objects := map[types.Object]identity.OccurrenceID{}
	for object, source := range verifier.sourceByObject {
		objects[object] = source
	}
	implicit := map[types.Object]*checkerBindingCandidate{}
	for _, occurrenceReference := range verifier.expected.order {
		occurrenceID := verifier.expected.
			occurrenceRecord(occurrenceReference).ID()
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		if !present {
			continue
		}
		if identifier, ok := node.(*ast.Ident); ok {
			object, defined := verifier.view.DefOf(identifier)
			if defined &&
				identifier.Name != "_" &&
				independentLexicalBinding(object) {
				if verifier.independentIntrinsicContractBinding(
					occurrenceID,
				) {
					continue
				}
				if existing := objects[object]; !existing.IsZero() && existing != occurrenceID {
					return nil, fmt.Errorf(
						"checker binding %s has sources %s and %s",
						object.Name(), existing, occurrenceID,
					)
				}
				objects[object] = occurrenceID
			}
			object, used := verifier.view.UseOf(identifier)
			if used &&
				independentLexicalBinding(object) &&
				!verifier.checkerSourceByObject[object].IsZero() {
				source := verifier.checkerSourceByObject[object]
				objects[object] = source
			}
		}
		object, present := verifier.view.ImplicitOf(node)
		if !present || !independentLexicalBinding(object) {
			continue
		}
		if verifier.independentIntrinsicContractBinding(occurrenceID) {
			continue
		}
		if variable, field := object.(*types.Var); field && variable.IsField() {
			continue
		}
		candidate, err := verifier.independentImplicitBinding(
			object, occurrenceID, scopeOwners,
		)
		if err != nil {
			return nil, err
		}
		if existing := implicit[object]; existing != nil {
			return nil, fmt.Errorf(
				"implicit checker binding %s has anchors %s and %s",
				object.Name(), existing.anchor, occurrenceID,
			)
		}
		implicit[object] = candidate
	}
	out := make(
		[]*checkerBindingCandidate,
		0,
		len(objects)+len(implicit),
	)
	for object, source := range objects {
		if !independentLexicalBinding(object) {
			continue
		}
		if verifier.independentIntrinsicContractBinding(source) {
			continue
		}
		role := verifier.independentBindingRole(object, source)
		if !role.Valid() {
			return nil, fmt.Errorf(
				"checker binding %s has no closed role",
				object.Name(),
			)
		}
		scope, err := verifier.independentBindingScope(
			object, source, scopeOwners,
		)
		if err != nil {
			return nil, err
		}
		typ := object.Type()
		if independentTypelessBinding(role) {
			typ = nil
		}
		out = append(out, &checkerBindingCandidate{
			object: object,
			anchor: source,
			source: source,
			scope:  scope,
			role:   role,
			definition: verifier.supportBindingDefinition(
				source, scope,
			),
			typ:  typ,
			name: object.Name(),
		})
	}
	for _, candidate := range implicit {
		out = append(out, candidate)
	}
	return out, nil
}

func (verifier *checkerSemanticVerifier) independentImplicitBinding(
	object types.Object,
	anchor identity.OccurrenceID,
	scopeOwners map[*types.Scope]identity.OccurrenceID,
) (*checkerBindingCandidate, error) {
	role := verifier.independentBindingRole(object, anchor)
	node, _ := verifier.index.OccurrenceNode(anchor)
	role = independentImplicitBindingRole(node, object, role)
	if !role.Valid() {
		return nil, fmt.Errorf(
			"implicit checker binding %s has no closed role",
			object.Name(),
		)
	}
	scope, err := verifier.independentBindingScope(
		object, anchor, scopeOwners,
	)
	if err != nil {
		return nil, err
	}
	typ := object.Type()
	if independentTypelessBinding(role) {
		typ = nil
	}
	if typ == nil && !independentTypelessBinding(role) {
		return nil, fmt.Errorf(
			"implicit checker binding %s has no type", object.Name(),
		)
	}
	return &checkerBindingCandidate{
		object: object,
		anchor: anchor,
		scope:  scope,
		role:   role,
		definition: verifier.bindingDefinition(
			anchor, scope,
		),
		typ:  typ,
		name: object.Name(),
	}, nil
}

func (verifier *checkerSemanticVerifier) independentScopeOwners() (
	map[*types.Scope]identity.OccurrenceID,
	error,
) {
	out := map[*types.Scope]identity.OccurrenceID{}
	err := verifier.view.VisitScopes(func(
		node ast.Node,
		scope *types.Scope,
	) error {
		occurrenceID, present := verifier.index.OccurrenceID(
			node,
		)
		if !present {
			return nil
		}
		if existing := out[scope]; !existing.IsZero() && existing != occurrenceID {
			return fmt.Errorf(
				"checker scope has owners %s and %s",
				existing, occurrenceID,
			)
		}
		out[scope] = occurrenceID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (verifier *checkerSemanticVerifier) independentBindingScope(
	object types.Object,
	source identity.OccurrenceID,
	scopeOwners map[*types.Scope]identity.OccurrenceID,
) (identity.OccurrenceID, error) {
	if owner := verifier.independentCheckerScopeOwner(
		object.Parent(), scopeOwners,
	); !owner.IsZero() {
		return owner, nil
	}
	return verifier.independentScopeForOccurrence(
		source, scopeOwners,
	)
}

func (verifier *checkerSemanticVerifier) independentScopeForOccurrence(
	occurrenceID identity.OccurrenceID,
	scopeOwners map[*types.Scope]identity.OccurrenceID,
) (identity.OccurrenceID, error) {
	if verifier.scopeOccurrenceResolved[occurrenceID] {
		scope := verifier.scopeByOccurrence[occurrenceID]
		if scope.IsZero() {
			return identity.OccurrenceID{}, fmt.Errorf(
				"binding anchor %s has no canonical scope",
				occurrenceID,
			)
		}
		return scope, nil
	}
	var path []identity.OccurrenceID
	current := occurrenceID
	scopeOwner := identity.OccurrenceID{}
	for !current.IsZero() {
		if verifier.scopeOccurrenceResolved[current] {
			scopeOwner = verifier.scopeByOccurrence[current]
			break
		}
		path = append(path, current)
		occurrence, present := verifier.expected.occurrences.get(current)
		if !present {
			break
		}
		node, nodePresent := verifier.index.OccurrenceNode(current)
		if nodePresent {
			if scope, scopePresent := verifier.view.ScopeOf(node); scopePresent && scopeOwners[scope] == current {
				scopeOwner = current
				break
			}
		}
		if catalog.LexicalScope(occurrence.Kind()) ==
			catalog.LexicalScopeAlways {
			scopeOwner = current
			break
		}
		current = occurrence.Parent()
	}
	for _, member := range path {
		verifier.scopeOccurrenceResolved[member] = true
		verifier.scopeByOccurrence[member] = scopeOwner
	}
	if scopeOwner.IsZero() {
		return identity.OccurrenceID{}, fmt.Errorf(
			"binding anchor %s has no canonical scope", occurrenceID,
		)
	}
	return scopeOwner, nil
}

func (verifier *checkerSemanticVerifier) independentCheckerScopeOwner(
	scope *types.Scope,
	scopeOwners map[*types.Scope]identity.OccurrenceID,
) identity.OccurrenceID {
	if scope == nil {
		return identity.OccurrenceID{}
	}
	if verifier.checkerScopeResolved[scope] {
		return verifier.checkerScopeOwner[scope]
	}
	var path []*types.Scope
	current := scope
	owner := identity.OccurrenceID{}
	for current != nil {
		if verifier.checkerScopeResolved[current] {
			owner = verifier.checkerScopeOwner[current]
			break
		}
		path = append(path, current)
		if direct := scopeOwners[current]; !direct.IsZero() {
			owner = direct
			break
		}
		current = current.Parent()
	}
	for _, member := range path {
		verifier.checkerScopeResolved[member] = true
		verifier.checkerScopeOwner[member] = owner
	}
	return owner
}

func (verifier *checkerSemanticVerifier) independentBindingRole(
	object types.Object,
	source identity.OccurrenceID,
) identity.SemanticBindingRole {
	switch object.(type) {
	case *types.PkgName:
		return identity.SemanticBindingImport
	case *types.Label:
		return identity.SemanticBindingLabel
	case *types.TypeName:
		return identity.SemanticBindingTypeParameter
	}
	for current := source; !current.IsZero(); {
		occurrence, present := verifier.expected.occurrences.get(current)
		if !present {
			if role, supportPresent :=
				verifier.index.StructuralSupport(current); supportPresent {
				return independentSupportBindingRole(role)
			}
			break
		}
		switch occurrence.Role() {
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
		}
		current = occurrence.Parent()
	}
	return identity.SemanticBindingLocal
}

func (verifier *checkerSemanticVerifier) bindingDefinition(
	anchor identity.OccurrenceID,
	scope identity.OccurrenceID,
) identity.DefinitionID {
	if owner := verifier.expected.occurrenceOwner(anchor); !owner.IsZero() {
		return owner
	}
	if owner := verifier.expected.structuralOccurrenceOwner(anchor); !owner.IsZero() {
		return owner
	}
	if owner := verifier.expected.occurrenceOwner(scope); !owner.IsZero() {
		return owner
	}
	return verifier.expected.structuralOccurrenceOwner(scope)
}

func (verifier *checkerSemanticVerifier) supportBindingDefinition(
	source identity.OccurrenceID,
	scope identity.OccurrenceID,
) identity.DefinitionID {
	if definition := verifier.bindingDefinition(
		source, scope,
	); !definition.IsZero() {
		return definition
	}
	if definition, present := verifier.index.OccurrenceDefinition(
		source,
	); present {
		return definition
	}
	definition, _ := verifier.index.OccurrenceDefinition(scope)
	return definition
}
