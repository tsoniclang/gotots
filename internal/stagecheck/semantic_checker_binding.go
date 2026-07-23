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
	object     types.Object
	anchor     identity.OccurrenceID
	source     identity.OccurrenceID
	scope      identity.OccurrenceID
	role       identity.SemanticBindingRole
	definition identity.DefinitionID
	typ        types.Type
	name       string
}

type checkerBindingGroup struct {
	scope identity.OccurrenceID
	role  identity.SemanticBindingRole
}

func (verifier *checkerSemanticVerifier) verifyBindings() error {
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
	expected := map[identity.SemanticBindingID]*checkerBindingCandidate{}
	for _, group := range groups {
		sort.Slice(group, func(left, right int) bool {
			return independentBindingOrder(group[left]) <
				independentBindingOrder(group[right])
		})
		for ordinal, candidate := range group {
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
	if len(expected) != len(verifier.bindings) {
		return fmt.Errorf(
			"bindings checker-derived=%d semantic=%d; missing=%v extra=%v",
			len(expected),
			len(verifier.bindings),
			bindingIdentityDifference(expected, verifier.bindings),
			semanticBindingIdentityDifference(verifier.bindings, expected),
		)
	}
	for id, candidate := range expected {
		record, present := verifier.bindings[id]
		if !present {
			return fmt.Errorf("checker-derived binding %s is absent", id)
		}
		if err := verifier.verifyBindingRecord(
			record, candidate,
		); err != nil {
			return fmt.Errorf("binding %s: %w", id, err)
		}
	}
	return nil
}

func (verifier *checkerSemanticVerifier) independentBindingCandidates() (
	[]*checkerBindingCandidate,
	error,
) {
	scopeOwners, err := verifier.independentScopeOwners()
	if err != nil {
		return nil, err
	}
	objects := map[types.Object]identity.OccurrenceID{}
	implicit := map[types.Object]*checkerBindingCandidate{}
	for _, occurrenceID := range verifier.expected.order {
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		if !present {
			continue
		}
		if identifier, ok := node.(*ast.Ident); ok {
			object, defined := verifier.view.DefOf(identifier)
			if defined &&
				identifier.Name != "_" &&
				independentLexicalBinding(object) {
				if existing := objects[object]; !existing.IsZero() && existing != occurrenceID {
					return nil, fmt.Errorf(
						"checker binding %s has sources %s and %s",
						object.Name(), existing, occurrenceID,
					)
				}
				objects[object] = occurrenceID
			}
		}
		object, present := verifier.view.ImplicitOf(node)
		if !present || !independentLexicalBinding(object) {
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
	for _, occurrenceID := range verifier.expected.order {
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		identifier, identifierNode := node.(*ast.Ident)
		if !present || !identifierNode {
			continue
		}
		object, used := verifier.view.UseOf(identifier)
		if !used ||
			!independentLexicalBinding(object) ||
			independentPackageName(object) ||
			!object.Pos().IsValid() ||
			!objects[object].IsZero() {
			continue
		}
		source, err := verifier.index.IdentifierOccurrence(
			object.Pos(), object.Name(),
		)
		if err != nil {
			return nil, err
		}
		objects[object] = source
	}
	out := make(
		[]*checkerBindingCandidate,
		0,
		len(objects)+len(implicit),
	)
	for object, source := range objects {
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
			definition: verifier.bindingDefinition(
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
	if variable, ok := object.(*types.Var); ok &&
		!variable.IsField() {
		node, _ := verifier.index.OccurrenceNode(anchor)
		if _, field := node.(*ast.Field); !field {
			role = identity.SemanticBindingTypeSwitch
		}
	}
	if !role.Valid() {
		return nil, fmt.Errorf(
			"implicit checker binding %s has no closed role",
			object.Name(),
		)
	}
	scope, err := verifier.independentScopeForOccurrence(
		anchor, scopeOwners,
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
	for _, occurrenceID := range verifier.expected.order {
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		if !present {
			continue
		}
		scope, present := verifier.view.ScopeOf(node)
		if !present {
			continue
		}
		if existing := out[scope]; !existing.IsZero() && existing != occurrenceID {
			return nil, fmt.Errorf(
				"checker scope has owners %s and %s",
				existing, occurrenceID,
			)
		}
		out[scope] = occurrenceID
	}
	return out, nil
}

func (verifier *checkerSemanticVerifier) independentBindingScope(
	object types.Object,
	source identity.OccurrenceID,
	scopeOwners map[*types.Scope]identity.OccurrenceID,
) (identity.OccurrenceID, error) {
	for current := object.Parent(); current != nil; current = current.Parent() {
		if owner := scopeOwners[current]; !owner.IsZero() {
			return owner, nil
		}
	}
	return verifier.independentScopeForOccurrence(
		source, scopeOwners,
	)
}

func (verifier *checkerSemanticVerifier) independentScopeForOccurrence(
	occurrenceID identity.OccurrenceID,
	scopeOwners map[*types.Scope]identity.OccurrenceID,
) (identity.OccurrenceID, error) {
	current := occurrenceID
	for !current.IsZero() {
		occurrence, present := verifier.expected.occurrences[current]
		if !present {
			break
		}
		node, nodePresent := verifier.index.OccurrenceNode(current)
		if nodePresent {
			if scope, scopePresent := verifier.view.ScopeOf(node); scopePresent && scopeOwners[scope] == current {
				return current, nil
			}
		}
		if catalog.LexicalScope(occurrence.Kind()) ==
			catalog.LexicalScopeAlways {
			return current, nil
		}
		current = occurrence.Parent()
	}
	node, present := verifier.index.OccurrenceNode(occurrenceID)
	if !present {
		return identity.OccurrenceID{}, fmt.Errorf(
			"binding anchor %s has no transient node", occurrenceID,
		)
	}
	best := identity.OccurrenceID{}
	bestWidth := int(^uint(0) >> 1)
	for scope, owner := range scopeOwners {
		if !scope.Contains(node.Pos()) {
			continue
		}
		width := owner.Span().End() - owner.Span().Start()
		if width < bestWidth {
			best = owner
			bestWidth = width
		}
	}
	if best.IsZero() {
		return identity.OccurrenceID{}, fmt.Errorf(
			"binding anchor %s has no canonical scope", occurrenceID,
		)
	}
	return best, nil
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
		occurrence, present := verifier.expected.occurrences[current]
		if !present {
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

func independentLexicalBinding(object types.Object) bool {
	switch object := object.(type) {
	case *types.PkgName, *types.Label:
		return true
	case *types.TypeName:
		_, typeParameter := object.Type().(*types.TypeParam)
		return typeParameter
	case *types.Var:
		return !object.IsField() &&
			(object.Pkg() == nil ||
				object.Parent() != object.Pkg().Scope())
	default:
		return false
	}
}

func independentPackageName(object types.Object) bool {
	_, packageName := object.(*types.PkgName)
	return packageName
}

func independentTypelessBinding(
	role identity.SemanticBindingRole,
) bool {
	return role == identity.SemanticBindingImport ||
		role == identity.SemanticBindingLabel
}

func (verifier *checkerSemanticVerifier) bindingDefinition(
	anchor identity.OccurrenceID,
	scope identity.OccurrenceID,
) identity.DefinitionID {
	if owner := verifier.expected.owners[anchor]; !owner.IsZero() {
		return owner
	}
	if owner := verifier.expected.structuralOwners[anchor]; !owner.IsZero() {
		return owner
	}
	if owner := verifier.expected.owners[scope]; !owner.IsZero() {
		return owner
	}
	return verifier.expected.structuralOwners[scope]
}

func independentBindingOrder(
	candidate *checkerBindingCandidate,
) string {
	if !candidate.source.IsZero() {
		return fmt.Sprintf(
			"%020d|%020d|%s",
			candidate.source.Span().Start(),
			candidate.source.Span().End(),
			candidate.name,
		)
	}
	position := 0
	if candidate.object != nil {
		position = int(candidate.object.Pos())
	}
	return fmt.Sprintf("%020d|%s", position, candidate.name)
}

func (verifier *checkerSemanticVerifier) verifyBindingRecord(
	record semantic.Binding,
	expected *checkerBindingCandidate,
) error {
	if record.Package() != verifier.expected.id ||
		record.Definition() != expected.definition ||
		record.Role() != expected.role ||
		record.Name() != expected.name ||
		record.Source() != expected.source {
		return fmt.Errorf(
			"semantic=%s/%s/%s/%q/%s checker=%s/%s/%s/%q/%s",
			record.Package(),
			record.Definition(),
			record.Role(),
			record.Name(),
			record.Source(),
			verifier.expected.id,
			expected.definition,
			expected.role,
			expected.name,
			expected.source,
		)
	}
	if expected.typ == nil {
		if !record.Type().IsZero() {
			return fmt.Errorf("typeless checker binding carries a type")
		}
		return nil
	}
	return verifier.types.verify(record.Type(), expected.typ)
}

func bindingIdentityDifference(
	expected map[identity.SemanticBindingID]*checkerBindingCandidate,
	actual map[identity.SemanticBindingID]semantic.Binding,
) []string {
	var out []string
	for id := range expected {
		if _, present := actual[id]; !present {
			out = append(out, id.String())
		}
	}
	sort.Strings(out)
	return out
}

func semanticBindingIdentityDifference(
	actual map[identity.SemanticBindingID]semantic.Binding,
	expected map[identity.SemanticBindingID]*checkerBindingCandidate,
) []string {
	var out []string
	for id := range actual {
		if _, present := expected[id]; !present {
			out = append(out, id.String())
		}
	}
	sort.Strings(out)
	return out
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
