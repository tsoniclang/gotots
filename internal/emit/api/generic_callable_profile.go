package api

import (
	"go/types"
	"sort"
)

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
