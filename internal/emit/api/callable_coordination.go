package api

import (
	"fmt"
	"go/types"
	"slices"
)

type CallableABIReference struct {
	artifact    *GeneratedArtifact
	sourceOwner types.Object
	requests    []RootRequest
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

func NewSourceCallableABIReference(
	sourceOwner types.Object,
	artifact *GeneratedArtifact,
	requests ...RootRequest,
) (CallableABIReference, error) {
	sourceOwner = GenericDeclarationOrigin(sourceOwner)
	reference, err := NewCallableABIReference(artifact, requests...)
	if err != nil {
		return CallableABIReference{}, err
	}
	if sourceOwner == nil || sourceOwner.Pkg() == nil {
		return CallableABIReference{}, &RootRequestError{
			Reason: "source callable ABI reference owner is invalid",
		}
	}
	reference.sourceOwner = sourceOwner
	return reference, nil
}

func (r CallableABIReference) Artifact() *GeneratedArtifact {
	return r.artifact
}

func (r CallableABIReference) SourceOwner() (types.Object, bool) {
	return r.sourceOwner, r.sourceOwner != nil
}

func (r CallableABIReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

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

type CooperativeCallableResolver interface {
	ObserveCooperativeCallable(
		Context,
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
	return WalkUniqueRootRequestPayloads(requests, func(RootRequest) error {
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
	reference CallableABIReference,
) (CallableFacet, error) {
	artifact := reference.Artifact()
	if c.genericCallableProfile != nil {
		if sourceOwner, scoped := reference.SourceOwner(); scoped {
			if sourceOwner != c.genericCallableProfile.Owner() {
				return CallableFacet{}, &ContextError{
					Reason: "callable ABI scope is foreign to the generic profile",
				}
			}
			return NewGenericProfileCallableABIFacet(
				c.genericCallableProfile,
				artifact,
			)
		}
		if _, selected :=
			c.genericCallableProfile.Selection().ABI(artifact); selected {
			return NewGenericProfileCallableABIFacet(
				c.genericCallableProfile,
				artifact,
			)
		}
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
		c,
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

func (c Context) WithDeferredCallableSelection() Context {
	if c.deferredCallableSelection {
		panic("deferred callable is already selected")
	}
	c.deferredCallableSelection = true
	return c
}

func (c Context) TakeDeferredCallableSelection() (Context, bool) {
	selected := c.deferredCallableSelection
	c.deferredCallableSelection = false
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
