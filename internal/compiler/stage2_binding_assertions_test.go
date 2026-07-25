package compiler

import (
	"fmt"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func requireMixedExplicitAndImplicitBindings(
	t *testing.T,
	bindings []semantic.Binding,
) {
	t.Helper()
	var bindingEvidence []string
	for _, binding := range bindings {
		bindingEvidence = append(
			bindingEvidence,
			fmt.Sprintf(
				"%q/%s/%s",
				binding.Name(),
				binding.Role(),
				binding.ID(),
			),
		)
	}
	var named, other semantic.Binding
	for _, binding := range bindings {
		switch {
		case binding.Name() == "named" &&
			binding.Role() == identity.SemanticBindingParameter:
			named = binding
		case binding.Name() == "other" &&
			binding.Role() == identity.SemanticBindingImport:
			other = binding
		}
	}
	if named.ID().IsZero() ||
		named.ID().Ordinal() != 0 ||
		other.ID().IsZero() ||
		other.ID().Ordinal() != 1 {
		t.Fatalf(
			"selected bindings named=%s other=%s, want ordinals 0/1",
			named.ID(), other.ID(),
		)
	}
	var left semantic.Binding
	unnamedParameters := map[identity.OccurrenceID][]semantic.Binding{}
	for _, binding := range bindings {
		switch {
		case binding.Name() == "" &&
			binding.Role() == identity.SemanticBindingParameter:
			unnamedParameters[binding.ID().Owner()] = append(
				unnamedParameters[binding.ID().Owner()],
				binding,
			)
		case binding.ID().Owner() == other.ID().Owner() &&
			binding.Name() == "left" &&
			binding.Role() == identity.SemanticBindingImport:
			left = binding
		}
	}
	if left.ID().IsZero() ||
		left.ID().Ordinal() != 0 {
		t.Fatalf(
			"preceding implicit import left=%s, want ordinal 0; bindings=%v",
			left.ID(), bindingEvidence,
		)
	}
	var unnamedPair bool
	for _, group := range unnamedParameters {
		if len(group) != 2 {
			continue
		}
		ordinals := map[int]bool{}
		for _, binding := range group {
			ordinals[binding.ID().Ordinal()] = true
		}
		if ordinals[0] && ordinals[1] {
			unnamedPair = true
			break
		}
	}
	if !unnamedPair {
		t.Fatalf(
			"two-slot unnamed parameter binding group is absent; bindings=%v",
			bindingEvidence,
		)
	}
}
