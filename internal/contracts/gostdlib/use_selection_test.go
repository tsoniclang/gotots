package gostdlib

import (
	"strings"
	"testing"
)

func TestFacetUseSelectionValidatesKindCapabilityCompatibility(t *testing.T) {
	valid := []struct {
		kind       FacetKind
		capability FacetCapability
	}{
		{FacetNamedStructOperations, FacetCapabilityZero},
		{FacetNamedStructOperations, FacetCapabilityCopy},
		{FacetDefinedValueOperations, FacetCapabilityProject},
		{FacetDefinedValueOperations, FacetCapabilityWrap},
		{FacetRecoveryCallable, FacetCapabilityRecovery},
		{FacetGenericCallableKernel, FacetCapabilityKernel},
		{FacetReflectionTypeOperations, FacetCapabilityMetadata},
	}
	for _, entry := range valid {
		selection, err := NewFacetUseSelection(entry.kind, entry.capability)
		if err != nil {
			t.Fatalf(
				"admissible facet %s/%s rejected: %v",
				entry.kind,
				entry.capability,
				err,
			)
		}
		kind, capability, ok := selection.Facet()
		if !ok || kind != entry.kind || capability != entry.capability {
			t.Fatalf(
				"facet selection lost typed identity: %s/%s/%v",
				kind,
				capability,
				ok,
			)
		}
	}
	invalid := []struct {
		kind       FacetKind
		capability FacetCapability
	}{
		{FacetInvalid, FacetCapabilityZero},
		{FacetKind("made-up"), FacetCapabilityZero},
		{FacetNamedStructOperations, FacetCapabilityMetadata},
		{FacetNamedStructOperations, FacetCapabilityKernel},
		{FacetReflectionTypeOperations, FacetCapabilityZero},
		{FacetGenericCallableKernel, FacetCapabilityRecovery},
		{FacetRecoveryCallable, FacetCapabilityKernel},
		{FacetDefinedValueOperations, FacetCapabilityCopy},
		{FacetNamedStructOperations, FacetCapabilityInvalid},
	}
	for _, entry := range invalid {
		if _, err := NewFacetUseSelection(
			entry.kind,
			entry.capability,
		); err == nil {
			t.Fatalf(
				"inadmissible facet %s/%s accepted",
				entry.kind,
				entry.capability,
			)
		}
	}
}

func TestProfileUseSelectionValidatesCanonicalDigestKeys(t *testing.T) {
	canonical := strings.Repeat("0f", 32)
	for _, construct := range []func(string) (UseSelection, error){
		NewCallableProfileUseSelection,
		NewStatefulProfileUseSelection,
	} {
		selection, err := construct(canonical)
		if err != nil {
			t.Fatalf("canonical profile key rejected: %v", err)
		}
		key, ok := selection.ProfileKey()
		if !ok || key != canonical {
			t.Fatalf("profile selection lost its key: %q/%v", key, ok)
		}
		for _, malformed := range []string{
			"",
			"short",
			strings.Repeat("0", 63),
			strings.Repeat("0", 65),
			strings.Repeat("0", 63) + "G",
			strings.Repeat("0", 63) + "F",
		} {
			if _, err := construct(malformed); err == nil {
				t.Fatalf("malformed profile key %q accepted", malformed)
			}
		}
	}
}

func TestNoUseSelectionCarriesNoEvidence(t *testing.T) {
	selection := NoUseSelection()
	if selection.Kind() != UseSelectionNone ||
		selection.EvidenceKey() != "" {
		t.Fatalf(
			"zero selection carries evidence: %v %q",
			selection.Kind(),
			selection.EvidenceKey(),
		)
	}
	if _, _, ok := selection.Facet(); ok {
		t.Fatal("zero selection exposed a facet identity")
	}
	if _, ok := selection.ProfileKey(); ok {
		t.Fatal("zero selection exposed a profile key")
	}
}
