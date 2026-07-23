package stagecheck

import (
	"fmt"
	"go/ast"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (verifier *checkerSemanticVerifier) verifyBindingCaptures() error {
	expected := map[identity.SemanticBindingID]map[identity.DefinitionID]bool{}
	addUse := func(
		bindingID identity.SemanticBindingID,
		occurrenceID identity.OccurrenceID,
	) error {
		binding := verifier.bindings[bindingID]
		if binding.ID().IsZero() ||
			binding.Definition().IsZero() {
			return nil
		}
		consumer := verifier.expected.owners[occurrenceID]
		if consumer.IsZero() ||
			consumer == binding.Definition() {
			return nil
		}
		if !verifier.definitionContains(
			binding.Definition(), consumer,
		) {
			return fmt.Errorf(
				"binding %s is used by unrelated definition %s",
				binding.ID(), consumer,
			)
		}
		if expected[binding.ID()] == nil {
			expected[binding.ID()] =
				map[identity.DefinitionID]bool{}
		}
		expected[binding.ID()][consumer] = true
		return nil
	}
	for _, occurrenceID := range verifier.expected.order {
		resolution, present := verifier.resolutions[occurrenceID]
		if !present ||
			resolution.Kind() != semantic.ResolutionBinding {
			continue
		}
		node, present := verifier.index.OccurrenceNode(occurrenceID)
		identifier, identifierNode := node.(*ast.Ident)
		if !present || !identifierNode {
			continue
		}
		if _, use := verifier.view.UseOf(identifier); !use {
			continue
		}
		if err := addUse(
			resolution.Binding(), occurrenceID,
		); err != nil {
			return err
		}
	}
	for _, operation := range verifier.operations {
		if !operation.ID().Source() {
			continue
		}
		object := operation.Spec().Object
		if object.Kind() != semantic.ObjectReferenceBinding {
			continue
		}
		node, present := verifier.index.OccurrenceNode(
			operation.Occurrence(),
		)
		identifier, identifierNode := node.(*ast.Ident)
		if !present || !identifierNode {
			continue
		}
		if _, use := verifier.view.UseOf(identifier); !use {
			continue
		}
		if err := addUse(
			object.Binding(), operation.Occurrence(),
		); err != nil {
			return err
		}
	}
	for bindingID, record := range verifier.bindings {
		var want []identity.DefinitionID
		for definition := range expected[bindingID] {
			want = append(want, definition)
		}
		sort.Slice(want, func(left, right int) bool {
			return want[left].String() < want[right].String()
		})
		if !slices.Equal(record.CapturedBy(), want) {
			return fmt.Errorf(
				"binding %s captures=%v, checker-derived=%v",
				bindingID, record.CapturedBy(), want,
			)
		}
	}
	return nil
}

func (verifier *checkerSemanticVerifier) definitionContains(
	outer identity.DefinitionID,
	inner identity.DefinitionID,
) bool {
	for current := inner; !current.IsZero(); {
		if current == outer {
			return true
		}
		current = verifier.expected.parents[current]
	}
	return false
}
