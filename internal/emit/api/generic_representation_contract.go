package api

import (
	"go/ast"
	"go/types"
	"slices"
)

type GenericRepresentationFacet uint8

const (
	GenericRepresentationInvalid GenericRepresentationFacet = 0
	GenericRepresentationStorage GenericRepresentationFacet = 1
	GenericRepresentationPointer GenericRepresentationFacet = 2
)

func (f GenericRepresentationFacet) Valid() bool {
	return f == GenericRepresentationStorage ||
		f == GenericRepresentationPointer
}

func (f GenericRepresentationFacet) String() string {
	switch f {
	case GenericRepresentationStorage:
		return "storage"
	case GenericRepresentationPointer:
		return "pointer"
	default:
		return "invalid"
	}
}

func GenericRepresentationName(
	logical string,
	facet GenericRepresentationFacet,
) (string, error) {
	if logical == "" || !facet.Valid() {
		return "", &NameError{
			Name:   logical,
			Reason: "generic representation name is invalid",
		}
	}
	switch facet {
	case GenericRepresentationStorage:
		return logical + "$Storage", nil
	case GenericRepresentationPointer:
		return logical + "$Pointer", nil
	default:
		panic("validated generic representation facet is unhandled")
	}
}

type GenericRepresentationSelection struct {
	parameter *types.TypeParam
	facet     GenericRepresentationFacet
}

func (s GenericRepresentationSelection) Parameter() *types.TypeParam {
	return s.parameter
}

func (s GenericRepresentationSelection) Facet() GenericRepresentationFacet {
	return s.facet
}

type GenericRepresentationProfile struct {
	owner      types.Object
	parameters []*types.TypeParam
	masks      []uint8
}

func SelectGenericRepresentationProfile(
	owner types.Object,
	requirements []DeclarationRequirement,
) (GenericRepresentationProfile, error) {
	owner = GenericDeclarationOrigin(owner)
	parameters := GenericDeclarationParameters(owner)
	if owner == nil || len(parameters) == 0 {
		return GenericRepresentationProfile{}, &InvariantError{
			Reason: "generic representation owner is invalid",
		}
	}
	profile := GenericRepresentationProfile{
		owner:      owner,
		parameters: slices.Clone(parameters),
		masks:      make([]uint8, len(parameters)),
	}
	for _, requirement := range requirements {
		if requirement.Kind() != DeclarationRequirementGenericRepresentation {
			continue
		}
		selectedOwner, parameter, facet, ok :=
			requirement.GenericRepresentation()
		index, indexed := GenericDeclarationParameterIndex(owner, parameter)
		if !ok || selectedOwner != owner || !indexed {
			return GenericRepresentationProfile{}, &InvariantError{
				Reason: "generic representation requirement has foreign ownership",
			}
		}
		mask := uint8(1) << facet
		if profile.masks[index]&mask != 0 {
			return GenericRepresentationProfile{}, &InvariantError{
				Reason: "generic representation requirement is duplicated",
			}
		}
		profile.masks[index] |= mask
	}
	return profile, nil
}

func (p GenericRepresentationProfile) Valid() bool {
	if p.owner == nil ||
		GenericDeclarationOrigin(p.owner) != p.owner ||
		len(p.parameters) == 0 ||
		len(p.parameters) != len(p.masks) {
		return false
	}
	expected := GenericDeclarationParameters(p.owner)
	if len(expected) != len(p.parameters) {
		return false
	}
	validMask := uint8(1)<<GenericRepresentationStorage |
		uint8(1)<<GenericRepresentationPointer
	for index, parameter := range p.parameters {
		if parameter != expected[index] || p.masks[index]&^validMask != 0 {
			return false
		}
	}
	return true
}

func (p GenericRepresentationProfile) Owner() types.Object {
	if !p.Valid() {
		return nil
	}
	return p.owner
}

func (p GenericRepresentationProfile) Parameters() []*types.TypeParam {
	if !p.Valid() {
		return nil
	}
	return slices.Clone(p.parameters)
}

func (p GenericRepresentationProfile) Requires(
	parameter *types.TypeParam,
	facet GenericRepresentationFacet,
) bool {
	if !p.Valid() || !facet.Valid() {
		return false
	}
	index, ok := GenericDeclarationParameterIndex(p.owner, parameter)
	return ok && p.masks[index]&(uint8(1)<<facet) != 0
}

func (p GenericRepresentationProfile) OrderedFacets() []GenericRepresentationSelection {
	if !p.Valid() {
		return nil
	}
	var result []GenericRepresentationSelection
	for index, parameter := range p.parameters {
		for facet := GenericRepresentationStorage; facet <= GenericRepresentationPointer; facet++ {
			if p.masks[index]&(uint8(1)<<facet) == 0 {
				continue
			}
			result = append(result, GenericRepresentationSelection{
				parameter: parameter,
				facet:     facet,
			})
		}
	}
	return result
}

func NewGenericRepresentationRequirement(
	owner types.Object,
	parameter *types.TypeParam,
	facet GenericRepresentationFacet,
) (DeclarationRequirement, error) {
	owner, parameter, ok := GenericRepresentationParameter(owner, parameter)
	if !ok || !facet.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generic representation requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:            MustSourceArtifactOwner(owner),
		kind:             DeclarationRequirementGenericRepresentation,
		genericParameter: parameter,
		genericFacet:     facet,
	}, nil
}

func NewGenericRepresentationRequest(
	owner types.Object,
	parameter *types.TypeParam,
	facet GenericRepresentationFacet,
) (RootRequest, error) {
	requirement, err := NewGenericRepresentationRequirement(
		owner,
		parameter,
		facet,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) GenericRepresentation() (
	types.Object,
	*types.TypeParam,
	GenericRepresentationFacet,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementGenericRepresentation {
		return nil, nil, GenericRepresentationInvalid, false
	}
	owner, ok := r.owner.Source()
	return owner, r.genericParameter, r.genericFacet, ok
}

func GenericDeclarationParameterIndex(
	owner types.Object,
	parameter *types.TypeParam,
) (int, bool) {
	owner = GenericDeclarationOrigin(owner)
	for index, selected := range GenericDeclarationParameters(owner) {
		if selected == parameter {
			return index, true
		}
	}
	return -1, false
}

func GenericRepresentationParameter(
	owner types.Object,
	parameter *types.TypeParam,
) (types.Object, *types.TypeParam, bool) {
	origin := GenericDeclarationOrigin(owner)
	index, ok := GenericDeclarationParameterIndex(origin, parameter)
	if !ok {
		return nil, nil, false
	}
	function, callable := origin.(*types.Func)
	if !callable {
		return origin, parameter, true
	}
	typeName := ValueReceiverTypeName(function)
	if typeName == nil {
		return origin, parameter, true
	}
	typeOwner := GenericDeclarationOrigin(typeName)
	parameters := GenericDeclarationParameters(typeOwner)
	if index >= len(parameters) {
		return nil, nil, false
	}
	return typeOwner, parameters[index], true
}

func (c Context) GenericParameterRepresentation(
	source ast.Node,
	parameter *types.TypeParam,
	facet GenericRepresentationFacet,
) (TypeEmission, error) {
	logical, ok := c.GenericParameterName(parameter)
	owner, selected, owned := GenericRepresentationParameter(
		c.genericParameterOwner,
		parameter,
	)
	if !ok || !owned || !facet.Valid() {
		return TypeEmission{}, &ContextError{
			Reason: "generic parameter representation is unavailable",
		}
	}
	name, err := GenericRepresentationName(logical, facet)
	if err != nil {
		return TypeEmission{}, err
	}
	request, err := NewGenericRepresentationRequest(owner, selected, facet)
	if err != nil {
		return TypeEmission{}, err
	}
	return DirectType(
		c.factory.TypeReferenceNode(c.factory.Identifier(name), nil),
		request,
	), nil
}

func (c Context) ResolveGenericRepresentationProfile(
	owner types.Object,
) (GenericRepresentationProfile, bool, error) {
	if c.genericResolver == nil {
		return GenericRepresentationProfile{}, false, &ContextError{
			Reason: "generic representation resolver is unavailable",
		}
	}
	return c.genericResolver.ResolveGenericRepresentationProfile(owner)
}
