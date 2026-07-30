package api

import (
	"go/types"
	"slices"
	"sort"
	"strings"
)

type GenericCallableABISelection struct {
	artifact    *GeneratedArtifact
	cooperative bool
}

func NewGenericCallableABISelection(
	artifact *GeneratedArtifact,
	cooperative bool,
) (GenericCallableABISelection, error) {
	if artifact == nil ||
		!artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactCallableABI {
		return GenericCallableABISelection{}, &InvariantError{
			Role:   RoleCallArgument,
			Reason: "generic callable ABI selection is invalid",
		}
	}
	return GenericCallableABISelection{
		artifact:    artifact,
		cooperative: cooperative,
	}, nil
}

func (s GenericCallableABISelection) Artifact() *GeneratedArtifact {
	return s.artifact
}

func (s GenericCallableABISelection) Cooperative() bool {
	return s.cooperative
}

func (s GenericCallableABISelection) valid() bool {
	return s.artifact != nil &&
		s.artifact.Valid() &&
		s.artifact.Kind() == GeneratedArtifactCallableABI
}

type GenericCallableProfileSelection struct {
	abis        []GenericCallableABISelection
	key         string
	cooperative bool
}

func NewGenericCallableProfileSelection(
	selections []GenericCallableABISelection,
) (GenericCallableProfileSelection, error) {
	merged := make(map[*GeneratedArtifact]bool, len(selections))
	for _, selection := range selections {
		if !selection.valid() {
			return GenericCallableProfileSelection{}, &InvariantError{
				Role:   RoleCallArgument,
				Reason: "generic callable profile selection is invalid",
			}
		}
		merged[selection.artifact] =
			merged[selection.artifact] || selection.cooperative
	}
	canonical := make(
		[]GenericCallableABISelection,
		0,
		len(merged),
	)
	for artifact, cooperative := range merged {
		canonical = append(canonical, GenericCallableABISelection{
			artifact:    artifact,
			cooperative: cooperative,
		})
	}
	sort.Slice(canonical, func(left, right int) bool {
		return canonical[left].artifact.ArtifactKey() <
			canonical[right].artifact.ArtifactKey()
	})
	var key strings.Builder
	for _, selection := range canonical {
		if key.Len() != 0 {
			key.WriteByte('|')
		}
		key.WriteString(selection.artifact.ArtifactKey())
		if selection.cooperative {
			key.WriteString("=cooperative")
		} else {
			key.WriteString("=synchronous")
		}
	}
	if key.Len() == 0 {
		key.WriteString("synchronous")
	}
	profile := GenericCallableProfileSelection{
		abis: slices.Clone(canonical),
		key:  key.String(),
	}
	for _, selection := range canonical {
		profile.cooperative =
			profile.cooperative || selection.cooperative
	}
	return profile, nil
}

func (s GenericCallableProfileSelection) Valid() bool {
	if s.key == "" {
		return false
	}
	previous := ""
	cooperative := false
	for _, selection := range s.abis {
		if !selection.valid() ||
			previous >= selection.artifact.ArtifactKey() {
			return false
		}
		previous = selection.artifact.ArtifactKey()
		cooperative = cooperative || selection.cooperative
	}
	return cooperative == s.cooperative
}

func (s GenericCallableProfileSelection) ABIs() []GenericCallableABISelection {
	return slices.Clone(s.abis)
}

func (s GenericCallableProfileSelection) Key() string {
	return s.key
}

func (s GenericCallableProfileSelection) Cooperative() bool {
	return s.cooperative
}

func (s GenericCallableProfileSelection) ABI(
	artifact *GeneratedArtifact,
) (bool, bool) {
	if artifact == nil {
		return false, false
	}
	index, found := slices.BinarySearchFunc(
		s.abis,
		artifact.ArtifactKey(),
		func(
			selection GenericCallableABISelection,
			key string,
		) int {
			return strings.Compare(selection.artifact.ArtifactKey(), key)
		},
	)
	if !found || s.abis[index].artifact != artifact {
		return false, false
	}
	return s.abis[index].cooperative, true
}

type GenericCallableProfile struct {
	owner     *types.Func
	selection GenericCallableProfileSelection
	suffix    string
}

func NewGenericCallableProfile(
	owner *types.Func,
	selection GenericCallableProfileSelection,
	suffix string,
) (*GenericCallableProfile, error) {
	if owner == nil ||
		owner.Origin() != owner ||
		len(GenericDeclarationParameters(owner)) == 0 ||
		!selection.Valid() ||
		!selection.Cooperative() ||
		suffix == "" {
		return nil, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic callable profile is invalid",
		}
	}
	return &GenericCallableProfile{
		owner:     owner,
		selection: selection,
		suffix:    suffix,
	}, nil
}

func (p *GenericCallableProfile) Owner() *types.Func {
	if p == nil {
		return nil
	}
	return p.owner
}

func (p *GenericCallableProfile) Selection() GenericCallableProfileSelection {
	if p == nil {
		return GenericCallableProfileSelection{}
	}
	return p.selection
}

func (p *GenericCallableProfile) Key() string {
	if p == nil {
		return ""
	}
	return p.selection.Key()
}

func (p *GenericCallableProfile) Suffix() string {
	if p == nil {
		return ""
	}
	return p.suffix
}

func (p *GenericCallableProfile) Valid() bool {
	return p != nil &&
		p.owner != nil &&
		p.owner.Origin() == p.owner &&
		len(GenericDeclarationParameters(p.owner)) != 0 &&
		p.selection.Valid() &&
		p.selection.Cooperative() &&
		p.suffix != ""
}

func SelectGenericCallableProfiles(
	owner *types.Func,
	requirements []DeclarationRequirement,
) ([]*GenericCallableProfile, error) {
	if owner == nil {
		return nil, &InvariantError{
			Role:   RoleCallCallee,
			Reason: "generic callable profile owner is nil",
		}
	}
	owner = owner.Origin()
	selected := make(map[string]*GenericCallableProfile)
	for _, requirement := range requirements {
		if requirement.Kind() !=
			DeclarationRequirementGenericCallableProfile {
			continue
		}
		profile, ok := requirement.GenericCallableProfile()
		if !ok || profile.Owner() != owner {
			return nil, &InvariantError{
				Role:   RoleCallCallee,
				Reason: "generic callable profile owner is inconsistent",
			}
		}
		if existing := selected[profile.Key()]; existing != nil &&
			existing != profile {
			return nil, &InvariantError{
				Role:   RoleCallCallee,
				Reason: "generic callable profile identity is duplicated",
			}
		}
		selected[profile.Key()] = profile
	}
	profiles := make([]*GenericCallableProfile, 0, len(selected))
	for _, profile := range selected {
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(left, right int) bool {
		return profiles[left].Key() < profiles[right].Key()
	})
	return profiles, nil
}

func NewGenericCallableProfileRequirement(
	profile *GenericCallableProfile,
) (DeclarationRequirement, error) {
	if !profile.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generic callable profile requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:          MustSourceArtifactOwner(profile.Owner()),
		kind:           DeclarationRequirementGenericCallableProfile,
		genericProfile: profile,
	}, nil
}

func NewGenericCallableProfileRequest(
	profile *GenericCallableProfile,
) (RootRequest, error) {
	requirement, err := NewGenericCallableProfileRequirement(profile)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) GenericCallableProfile() (
	*GenericCallableProfile,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementGenericCallableProfile {
		return nil, false
	}
	return r.genericProfile, true
}
