package stagecheck

import (
	"fmt"
	"go/ast"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func (
	verifier *checkerSemanticVerifier,
) verifyOperationsAndBindingCaptures(
	expected map[identity.SemanticBindingID]map[identity.DefinitionID]bool,
	resolvedOperations int,
) error {
	sourceOperations := 0
	implicitOperations := 0
	if err := verifier.visitOperations(func(
		operation semantic.Operation,
	) error {
		if !operation.ID().Source() {
			implicitOperations++
			if operation.Kind() !=
				semantic.OperationPackageInitialization ||
				operation.ID().ImplicitOp() !=
					identity.ImplicitDefinitionPackageInit {
				return fmt.Errorf(
					"invalid implicit operation origin %s",
					operation.ID(),
				)
			}
			return nil
		}
		sourceOperations++
		occurrence, present := verifier.expected.occurrences.get(
			operation.Occurrence(),
		)
		if !present ||
			occurrence.domain !=
				catalog.ResolutionDomainExecutable ||
			operation.Definition() !=
				verifier.expected.definitionID(occurrence.owner) ||
			operation.Syntax() != occurrence.Kind() ||
			operation.Role() != occurrence.Role() ||
			operation.Token() != occurrence.Token() {
			return fmt.Errorf(
				"source operation origin differs for %s",
				operation.ID(),
			)
		}
		object := operation.Object()
		if object.Kind() != semantic.ObjectReferenceBinding {
			return nil
		}
		node, present := verifier.index.OccurrenceNode(
			operation.Occurrence(),
		)
		identifier, identifierNode := node.(*ast.Ident)
		if !present || !identifierNode {
			return nil
		}
		if _, use := verifier.view.UseOf(identifier); !use {
			return nil
		}
		return verifier.addCheckerCaptureUse(
			expected,
			object.Binding(), operation.Occurrence(),
		)
	}); err != nil {
		return err
	}
	if sourceOperations != resolvedOperations {
		return fmt.Errorf(
			"%d source operations differ from %d operation resolutions",
			sourceOperations,
			resolvedOperations,
		)
	}
	implicitDefinitions := 0
	for definition := range verifier.expected.definitions {
		if definition.ImplicitOp().Valid() &&
			verifier.expected.executable[definition] {
			implicitDefinitions++
		}
	}
	if implicitOperations != implicitDefinitions {
		return fmt.Errorf(
			"%d implicit operations differ from %d implicit definitions",
			implicitOperations,
			implicitDefinitions,
		)
	}
	for bindingID := range verifier.bindings {
		var want []identity.DefinitionID
		for definition := range expected[bindingID] {
			want = append(want, definition)
		}
		sort.Slice(want, func(left, right int) bool {
			return want[left].Compare(want[right]) < 0
		})
		if !slices.Equal(verifier.bindingCaptures[bindingID], want) {
			return fmt.Errorf(
				"binding %s captures=%v, checker-derived=%v",
				bindingID, verifier.bindingCaptures[bindingID], want,
			)
		}
	}
	return nil
}

func (verifier *checkerSemanticVerifier) recordDirectCaptureUse(
	expected map[identity.SemanticBindingID]map[identity.DefinitionID]bool,
	resolution semantic.OccurrenceResolution,
	occurrenceID identity.OccurrenceID,
	node ast.Node,
) error {
	if resolution.Kind() != semantic.ResolutionBinding {
		return nil
	}
	identifier, identifierNode := node.(*ast.Ident)
	if !identifierNode {
		return nil
	}
	if _, use := verifier.view.UseOf(identifier); !use {
		return nil
	}
	return verifier.addCheckerCaptureUse(
		expected, resolution.Binding(), occurrenceID,
	)
}

func (verifier *checkerSemanticVerifier) addCheckerCaptureUse(
	expected map[identity.SemanticBindingID]map[identity.DefinitionID]bool,
	bindingID identity.SemanticBindingID,
	occurrenceID identity.OccurrenceID,
) error {
	binding := verifier.bindings[bindingID]
	if binding == nil || binding.definition.IsZero() {
		return nil
	}
	record, present := verifier.expected.occurrences.get(occurrenceID)
	if !present {
		return fmt.Errorf(
			"capture use names absent occurrence %s", occurrenceID,
		)
	}
	consumer := verifier.expected.definitionID(record.owner)
	if consumer.IsZero() ||
		consumer == binding.definition {
		return nil
	}
	if !verifier.definitionContains(
		binding.definition, consumer,
	) {
		return fmt.Errorf(
			"binding %s is used by unrelated definition %s",
			binding.id, consumer,
		)
	}
	if expected[binding.id] == nil {
		expected[binding.id] =
			map[identity.DefinitionID]bool{}
	}
	expected[binding.id][consumer] = true
	return nil
}

func (verifier *checkerSemanticVerifier) definitionContains(
	outer identity.DefinitionID,
	inner identity.DefinitionID,
) bool {
	outerInterval, outerPresent := verifier.containment[outer]
	innerInterval, innerPresent := verifier.containment[inner]
	return outerPresent &&
		innerPresent &&
		outerInterval.enter <= innerInterval.enter &&
		innerInterval.leave <= outerInterval.leave
}
