package api

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"sort"
)

type GenericCapabilityReference struct {
	artifact *GeneratedArtifact
	name     string
	requests []RootRequest
}

func NewGenericCapabilityReference(
	artifact *GeneratedArtifact,
	name string,
	requests ...RootRequest,
) (GenericCapabilityReference, error) {
	if artifact == nil ||
		artifact.Kind() != GeneratedArtifactGenericCapability ||
		!artifact.Valid() ||
		name == "" ||
		name != artifact.TargetName() {
		return GenericCapabilityReference{}, &NameError{
			Reason: "generic-capability reference is invalid",
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return GenericCapabilityReference{}, &RootRequestError{
			Reason: "generic-capability reference request is invalid",
		}
	}
	return GenericCapabilityReference{
		artifact: artifact,
		name:     name,
		requests: slices.Clone(requests),
	}, nil
}

func (r GenericCapabilityReference) Artifact() *GeneratedArtifact {
	return r.artifact
}

func (r GenericCapabilityReference) Name() string {
	return r.name
}

func (r GenericCapabilityReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

type GenericOperationReference struct {
	contract *GenericOperationContract
	name     string
	requests []RootRequest
}

func NewGenericOperationReference(
	contract *GenericOperationContract,
	name string,
	requests ...RootRequest,
) (GenericOperationReference, error) {
	if !contract.Valid() ||
		name == "" ||
		name != contract.TargetName() {
		return GenericOperationReference{}, &NameError{
			Reason: "generic-operation reference is invalid",
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return GenericOperationReference{}, &RootRequestError{
			Reason: "generic-operation reference request is invalid",
		}
	}
	return GenericOperationReference{
		contract: contract,
		name:     name,
		requests: slices.Clone(requests),
	}, nil
}

func (r GenericOperationReference) Contract() *GenericOperationContract {
	return r.contract
}

func (r GenericOperationReference) Name() string {
	return r.name
}

func (r GenericOperationReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

type CallableABIReference struct {
	artifact *GeneratedArtifact
	requests []RootRequest
}

func NewCallableABIReference(
	artifact *GeneratedArtifact,
	requests ...RootRequest,
) (CallableABIReference, error) {
	_, ok := artifact.CallableABI()
	if !ok {
		return CallableABIReference{}, &RootRequestError{
			Reason: "callable ABI reference is invalid",
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return CallableABIReference{}, &RootRequestError{
			Reason: "callable ABI reference request is invalid",
		}
	}
	return CallableABIReference{
		artifact: artifact,
		requests: slices.Clone(requests),
	}, nil
}

func (r CallableABIReference) Artifact() *GeneratedArtifact {
	return r.artifact
}

func (r CallableABIReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

type CooperativeCallableResolver interface {
	ObserveCooperativeCallable(
		ArtifactOwner,
		CallableFacet,
	) (CooperativeCallableObservation, error)
}

type CooperativeCallableObservation struct {
	cooperative bool
	requests    []RootRequest
}

func NewCooperativeCallableObservation(
	cooperative bool,
	requests ...RootRequest,
) (CooperativeCallableObservation, error) {
	if err := validateReferenceRequests(requests); err != nil {
		return CooperativeCallableObservation{}, &RootRequestError{
			Reason: "cooperative callable observation has an invalid request",
		}
	}
	return CooperativeCallableObservation{
		cooperative: cooperative,
		requests:    slices.Clone(requests),
	}, nil
}

func (o CooperativeCallableObservation) Cooperative() bool {
	return o.cooperative
}

func (o CooperativeCallableObservation) Requests() []RootRequest {
	return slices.Clone(o.requests)
}

func validateReferenceRequests(requests []RootRequest) error {
	return WalkRootRequests(requests, func(RootRequest) error {
		return nil
	})
}

func (c Context) WithCooperativeCallableResolver(
	resolver CooperativeCallableResolver,
) Context {
	if resolver == nil {
		panic("cooperative callable resolver is nil")
	}
	c.cooperativeResolver = resolver
	return c
}

func (c Context) WithCooperativeCallable(
	facet CallableFacet,
	cooperative bool,
) Context {
	if !facet.Valid() || facet.Owner() != c.artifactOwner {
		panic("cooperative callable facet is inconsistent")
	}
	c.callableFacet = facet
	c.cooperative = cooperative
	return c
}

func (c Context) WithCooperativeCallableABI(
	facet CallableFacet,
	cooperative bool,
) Context {
	if !c.artifactOwner.Valid() ||
		!facet.Valid() ||
		facet.Kind() != CallableFacetABI {
		panic("cooperative callable ABI boundary is invalid")
	}
	c.callableFacet = facet
	c.cooperative = cooperative
	return c
}

func (c Context) WithGenericCallableProfile(
	profile *GenericCallableProfile,
) Context {
	if !c.artifactOwner.Valid() ||
		!profile.Valid() ||
		c.artifactOwner != MustSourceArtifactOwner(profile.Owner()) {
		panic("generic callable profile boundary is invalid")
	}
	c.genericCallableProfile = profile
	return c
}

func (c Context) GenericCallableProfile() (
	*GenericCallableProfile,
	bool,
) {
	return c.genericCallableProfile, c.genericCallableProfile != nil
}

func (c Context) CallableABIFacet(
	artifact *GeneratedArtifact,
) (CallableFacet, error) {
	if c.genericCallableProfile != nil {
		return NewGenericProfileCallableABIFacet(
			c.genericCallableProfile,
			artifact,
		)
	}
	return NewCallableABIFacet(artifact)
}

func (c Context) IsCooperative() bool {
	return c.cooperative
}

func (c Context) ObserveCooperativeCallable(
	facet CallableFacet,
) (CooperativeCallableObservation, error) {
	if c.cooperativeResolver == nil {
		return CooperativeCallableObservation{}, &ContextError{
			Reason: "cooperative callable resolver is unavailable",
		}
	}
	if !facet.Valid() {
		return CooperativeCallableObservation{}, &ContextError{
			Reason: "cooperative callable facet is invalid",
		}
	}
	if !c.artifactOwner.Valid() {
		return CooperativeCallableObservation{}, &ContextError{
			Reason: "cooperative callable consumer has no artifact owner",
		}
	}
	observation, err := c.cooperativeResolver.ObserveCooperativeCallable(
		c.artifactOwner,
		facet,
	)
	if err != nil || c.genericCallableProfile == nil {
		return observation, err
	}
	artifact, abi := facet.ABI()
	if !abi {
		return observation, nil
	}
	cooperative, selected :=
		c.genericCallableProfile.Selection().ABI(artifact)
	if !selected {
		return observation, nil
	}
	return NewCooperativeCallableObservation(
		cooperative,
		observation.Requests()...,
	)
}

func (c Context) CooperativeRequest() (RootRequest, error) {
	if !c.callableFacet.Valid() {
		return RootRequest{}, &ContextError{
			Reason: fmt.Sprintf(
				"cooperative operation in %s (%s) has no callable facet",
				c.artifactOwner.Name(),
				c.role,
			),
		}
	}
	return NewCooperativeCallableRequest(c.callableFacet)
}

func (c Context) WithStaticallySelectedCallable() Context {
	if c.staticallySelectedCallable {
		panic("callable is already statically selected")
	}
	c.staticallySelectedCallable = true
	return c
}

func (c Context) TakeStaticallySelectedCallable() (Context, bool) {
	selected := c.staticallySelectedCallable
	c.staticallySelectedCallable = false
	return c, selected
}

func (c Context) WithDetachedInvocation() Context {
	if c.detachedInvocation {
		panic("detached invocation is already selected")
	}
	c.detachedInvocation = true
	return c
}

func (c Context) TakeDetachedInvocation() (Context, bool) {
	selected := c.detachedInvocation
	c.detachedInvocation = false
	return c, selected
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

type ControlLabel struct {
	name        string
	breakable   bool
	continuable bool
}

func NewControlLabel(
	name string,
	breakable bool,
	continuable bool,
) (ControlLabel, error) {
	if name == "" || continuable && !breakable {
		return ControlLabel{}, &InvariantError{
			Role:   RoleLabelTarget,
			Reason: "control-label target is invalid",
		}
	}
	return ControlLabel{
		name:        name,
		breakable:   breakable,
		continuable: continuable,
	}, nil
}

func (l ControlLabel) Valid() bool {
	return l.name != "" && (!l.continuable || l.breakable)
}

func (l ControlLabel) Name() string {
	return l.name
}

func (l ControlLabel) Breakable() bool {
	return l.breakable
}

func (l ControlLabel) Continuable() bool {
	return l.continuable
}

type IteratorRangeState int8

const (
	IteratorRangeStateExhausted IteratorRangeState = -2
	IteratorRangeStatePanicked  IteratorRangeState = -1
	IteratorRangeStateDone      IteratorRangeState = 0
	IteratorRangeStateReady     IteratorRangeState = 1
	IteratorRangeStateReturned  IteratorRangeState = 2
)

func (s IteratorRangeState) Literal() string {
	switch s {
	case IteratorRangeStateExhausted:
		return "-2"
	case IteratorRangeStatePanicked:
		return "-1"
	case IteratorRangeStateDone:
		return "0"
	case IteratorRangeStateReady:
		return "1"
	case IteratorRangeStateReturned:
		return "2"
	default:
		return ""
	}
}

type IteratorRangeControl struct {
	source     *ast.RangeStmt
	stateName  string
	resultName string
	returning  bool
}

func NewIteratorRangeControl(
	source *ast.RangeStmt,
	stateName string,
	resultName string,
	returning bool,
) (IteratorRangeControl, error) {
	if source == nil ||
		source.X == nil ||
		source.Body == nil ||
		stateName == "" ||
		(!returning && resultName != "") {
		return IteratorRangeControl{}, &InvariantError{
			Role:   RoleRangeBody,
			Reason: "iterator-range control is invalid",
		}
	}
	return IteratorRangeControl{
		source:     source,
		stateName:  stateName,
		resultName: resultName,
		returning:  returning,
	}, nil
}

func (c IteratorRangeControl) Source() *ast.RangeStmt {
	return c.source
}

func (c IteratorRangeControl) StateName() string {
	return c.stateName
}

func (c IteratorRangeControl) ResultName() string {
	return c.resultName
}

func (c IteratorRangeControl) Returning() bool {
	return c.returning
}

func (c IteratorRangeControl) Valid() bool {
	return c.source != nil &&
		c.source.X != nil &&
		c.source.Body != nil &&
		c.stateName != "" &&
		(c.returning || c.resultName == "")
}
