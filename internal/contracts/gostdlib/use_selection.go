package gostdlib

import "fmt"

// UseSelectionError reports an invalid provider use-selection identity.
type UseSelectionError struct {
	Reason string
}

func (e *UseSelectionError) Error() string {
	return "provider use selection: " + e.Reason
}

// UseSelectionKind identifies which certified provider surface one
// environment use selected beyond the declaration's ordinary binding.
type UseSelectionKind uint8

const (
	UseSelectionNone UseSelectionKind = iota
	UseSelectionFacet
	UseSelectionCallableProfile
	UseSelectionStatefulProfile
)

func (k UseSelectionKind) String() string {
	switch k {
	case UseSelectionNone:
		return "none"
	case UseSelectionFacet:
		return "facet"
	case UseSelectionCallableProfile:
		return "callable-profile"
	case UseSelectionStatefulProfile:
		return "stateful-profile"
	default:
		return "invalid"
	}
}

// UseSelection is the closed typed identity of one certified provider facet
// or profile selected by an environment use. The zero value selects the
// declaration's ordinary certified binding. Values are comparable and
// deduplicate structurally; serialization belongs only to the evidence
// codec boundary.
type UseSelection struct {
	kind       UseSelectionKind
	facetKind  FacetKind
	capability FacetCapability
	profileKey string
}

func NoUseSelection() UseSelection {
	return UseSelection{}
}

// NewFacetUseSelection validates one certified facet identity, including
// the closed facet-kind/capability compatibility.
func NewFacetUseSelection(
	facetKind FacetKind,
	capability FacetCapability,
) (UseSelection, error) {
	if !facetKind.Valid() {
		return UseSelection{}, &UseSelectionError{
			Reason: fmt.Sprintf("facet kind %q is invalid", string(facetKind)),
		}
	}
	compatible := false
	switch facetKind {
	case FacetNamedStructOperations:
		compatible = capability.NamedStructOperation()
	case FacetDefinedValueOperations:
		compatible = capability.DefinedValueOperation()
	case FacetRecoveryCallable:
		compatible = capability == FacetCapabilityRecovery
	case FacetGenericCallableKernel:
		compatible = capability == FacetCapabilityKernel
	case FacetReflectionTypeOperations:
		compatible = capability == FacetCapabilityMetadata
	}
	if !compatible {
		return UseSelection{}, &UseSelectionError{
			Reason: fmt.Sprintf(
				"facet capability %q is not admissible for kind %q",
				string(capability),
				string(facetKind),
			),
		}
	}
	return UseSelection{
		kind:       UseSelectionFacet,
		facetKind:  facetKind,
		capability: capability,
	}, nil
}

// NewCallableProfileUseSelection validates one canonical callable-profile
// digest key.
func NewCallableProfileUseSelection(profileKey string) (UseSelection, error) {
	if err := validateProfileKey(profileKey); err != nil {
		return UseSelection{}, err
	}
	return UseSelection{
		kind:       UseSelectionCallableProfile,
		profileKey: profileKey,
	}, nil
}

// NewStatefulProfileUseSelection validates one canonical stateful-profile
// digest key.
func NewStatefulProfileUseSelection(profileKey string) (UseSelection, error) {
	if err := validateProfileKey(profileKey); err != nil {
		return UseSelection{}, err
	}
	return UseSelection{
		kind:       UseSelectionStatefulProfile,
		profileKey: profileKey,
	}, nil
}

// validateProfileKey accepts only the canonical 64-hex certified profile
// digest form.
func validateProfileKey(profileKey string) error {
	if len(profileKey) != 64 {
		return &UseSelectionError{
			Reason: fmt.Sprintf(
				"profile key %q is not a 64-hex canonical digest",
				profileKey,
			),
		}
	}
	for _, digit := range profileKey {
		if (digit < '0' || digit > '9') && (digit < 'a' || digit > 'f') {
			return &UseSelectionError{
				Reason: fmt.Sprintf(
					"profile key %q contains a non-hex digit",
					profileKey,
				),
			}
		}
	}
	return nil
}

func (s UseSelection) Kind() UseSelectionKind {
	return s.kind
}

// Facet returns the certified facet identity of a facet selection.
func (s UseSelection) Facet() (FacetKind, FacetCapability, bool) {
	if s.kind != UseSelectionFacet {
		return FacetInvalid, FacetCapabilityInvalid, false
	}
	return s.facetKind, s.capability, true
}

// ProfileKey returns the canonical profile digest of a profile selection.
func (s UseSelection) ProfileKey() (string, bool) {
	if s.kind != UseSelectionCallableProfile &&
		s.kind != UseSelectionStatefulProfile {
		return "", false
	}
	return s.profileKey, true
}

// EvidenceKey is the canonical serialized identity of one provider
// selection. It exists for the evidence codec boundary and deterministic
// ordering only; typed accessors carry the structural evidence.
func (s UseSelection) EvidenceKey() string {
	switch s.kind {
	case UseSelectionFacet:
		return "facet:" + string(s.facetKind) + ":" + string(s.capability)
	case UseSelectionCallableProfile:
		return "callable-profile:" + s.profileKey
	case UseSelectionStatefulProfile:
		return "stateful-profile:" + s.profileKey
	default:
		return ""
	}
}
