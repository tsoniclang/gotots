package api

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
)

type CallableFacetKind uint8

const (
	CallableFacetInvalid            CallableFacetKind = 0
	CallableFacetSource             CallableFacetKind = 1
	CallableFacetFunctionLiteral    CallableFacetKind = 2
	CallableFacetABI                CallableFacetKind = 3
	CallableFacetGenericCapability  CallableFacetKind = 4
	CallableFacetGenericOperation   CallableFacetKind = 5
	CallableFacetPackageInitializer CallableFacetKind = 6
	CallableFacetGenericProfile     CallableFacetKind = 7
)

func (k CallableFacetKind) Valid() bool {
	return k >= CallableFacetSource &&
		k <= CallableFacetGenericProfile
}

type CallableFacet struct {
	owner     ArtifactOwner
	kind      CallableFacetKind
	function  *types.Func
	literal   *ast.FuncLit
	generated *GeneratedArtifact
	operation *GenericOperationContract
	profile   *GenericCallableProfile
}

func NewSourceCallableFacet(function *types.Func) (CallableFacet, error) {
	if function == nil {
		return CallableFacet{}, &RootRequestError{
			Reason: "source callable facet owner is nil",
		}
	}
	function = function.Origin()
	signature, ok := function.Type().(*types.Signature)
	if !ok || signature == nil {
		return CallableFacet{}, &RootRequestError{
			Reason: "source callable facet owner has no signature",
		}
	}
	return CallableFacet{
		owner:    MustSourceArtifactOwner(function),
		kind:     CallableFacetSource,
		function: function,
	}, nil
}

func NewFunctionLiteralCallableFacet(
	owner ArtifactOwner,
	literal *ast.FuncLit,
) (CallableFacet, error) {
	_, sourceOwned := owner.Source()
	_, _, initializerOwned := owner.PackageInitializer()
	if (!sourceOwned && !initializerOwned) ||
		literal == nil ||
		literal.Type == nil ||
		literal.Body == nil {
		return CallableFacet{}, &RootRequestError{
			Reason: "function-literal callable facet is invalid",
		}
	}
	return CallableFacet{
		owner:   owner,
		kind:    CallableFacetFunctionLiteral,
		literal: literal,
	}, nil
}

func NewPackageInitializerCallableFacet(
	owner ArtifactOwner,
) (CallableFacet, error) {
	if _, _, ok := owner.PackageInitializer(); !ok {
		return CallableFacet{}, &RootRequestError{
			Reason: "package-initializer callable facet is invalid",
		}
	}
	return CallableFacet{
		owner: owner,
		kind:  CallableFacetPackageInitializer,
	}, nil
}

func NewGenericCallableProfileFacet(
	profile *GenericCallableProfile,
) (CallableFacet, error) {
	if !profile.Valid() {
		return CallableFacet{}, &RootRequestError{
			Reason: "generic callable profile facet is invalid",
		}
	}
	return CallableFacet{
		owner:   MustSourceArtifactOwner(profile.Owner()),
		kind:    CallableFacetGenericProfile,
		profile: profile,
	}, nil
}

func NewCallableABIFacet(
	artifact *GeneratedArtifact,
) (CallableFacet, error) {
	if artifact == nil ||
		artifact.Kind() != GeneratedArtifactCallableABI ||
		!artifact.Valid() {
		return CallableFacet{}, &RootRequestError{
			Reason: "callable ABI facet is invalid",
		}
	}
	return CallableFacet{
		owner:     MustGeneratedArtifactOwner(artifact),
		kind:      CallableFacetABI,
		generated: artifact,
	}, nil
}

func NewGenericCapabilityCallableFacet(
	artifact *GeneratedArtifact,
) (CallableFacet, error) {
	if artifact == nil ||
		artifact.Kind() != GeneratedArtifactGenericCapability ||
		!artifact.Valid() {
		return CallableFacet{}, &RootRequestError{
			Reason: "generic-capability callable facet is invalid",
		}
	}
	return CallableFacet{
		owner:     artifact.ReconstructionOwner(),
		kind:      CallableFacetGenericCapability,
		generated: artifact,
	}, nil
}

func NewGenericOperationCallableFacet(
	operation *GenericOperationContract,
) (CallableFacet, error) {
	function, functionOwned := operationOwnerFunction(operation)
	if !operation.Valid() ||
		!functionOwned ||
		operation.Consumer() != GenericFunctionOperationConsumer() {
		return CallableFacet{}, &RootRequestError{
			Reason: "generic-operation callable facet is invalid",
		}
	}
	return CallableFacet{
		owner:     MustSourceArtifactOwner(function),
		kind:      CallableFacetGenericOperation,
		operation: operation,
	}, nil
}

func (f CallableFacet) Valid() bool {
	if !f.owner.Valid() || !f.kind.Valid() {
		return false
	}
	switch f.kind {
	case CallableFacetSource:
		source, sourceOwned := f.owner.Source()
		function, callable := source.(*types.Func)
		signature, signatureOK := functionType(function)
		return sourceOwned &&
			callable &&
			signatureOK &&
			function.Origin() == function &&
			f.function == function &&
			f.literal == nil &&
			f.generated == nil &&
			f.operation == nil &&
			f.profile == nil &&
			signature != nil
	case CallableFacetFunctionLiteral:
		_, sourceOwned := f.owner.Source()
		_, _, initializerOwned := f.owner.PackageInitializer()
		return (sourceOwned || initializerOwned) &&
			f.function == nil &&
			f.literal != nil &&
			f.literal.Type != nil &&
			f.literal.Body != nil &&
			f.generated == nil &&
			f.operation == nil &&
			f.profile == nil
	case CallableFacetABI:
		generated, generatedOwned := f.owner.Generated()
		return generatedOwned &&
			f.function == nil &&
			f.literal == nil &&
			f.generated == generated &&
			f.generated.Kind() == GeneratedArtifactCallableABI &&
			f.generated.Valid() &&
			f.operation == nil &&
			f.profile == nil
	case CallableFacetGenericCapability:
		return f.generated != nil &&
			f.owner == f.generated.ReconstructionOwner() &&
			f.function == nil &&
			f.literal == nil &&
			f.generated.Kind() == GeneratedArtifactGenericCapability &&
			f.generated.Valid() &&
			f.operation == nil &&
			f.profile == nil
	case CallableFacetGenericOperation:
		source, sourceOwned := f.owner.Source()
		function, functionOwned := operationOwnerFunction(f.operation)
		return sourceOwned &&
			functionOwned &&
			source == function &&
			f.operation.Valid() &&
			f.function == nil &&
			f.literal == nil &&
			f.generated == nil &&
			f.profile == nil &&
			f.operation.Consumer() ==
				GenericFunctionOperationConsumer()
	case CallableFacetPackageInitializer:
		_, _, initializerOwned := f.owner.PackageInitializer()
		return initializerOwned &&
			f.function == nil &&
			f.literal == nil &&
			f.generated == nil &&
			f.operation == nil &&
			f.profile == nil
	case CallableFacetGenericProfile:
		source, sourceOwned := f.owner.Source()
		return sourceOwned &&
			f.profile.Valid() &&
			source == f.profile.Owner() &&
			f.function == nil &&
			f.literal == nil &&
			f.generated == nil &&
			f.operation == nil
	default:
		return false
	}
}

func (f CallableFacet) empty() bool {
	return !f.owner.Valid() &&
		f.kind == CallableFacetInvalid &&
		f.function == nil &&
		f.literal == nil &&
		f.generated == nil &&
		f.operation == nil &&
		f.profile == nil
}

func (f CallableFacet) Owner() ArtifactOwner {
	return f.owner
}

func (f CallableFacet) Kind() CallableFacetKind {
	return f.kind
}

func (f CallableFacet) SourceFunction() (*types.Func, bool) {
	return f.function, f.Valid() && f.kind == CallableFacetSource
}

func (f CallableFacet) FunctionLiteral() (*ast.FuncLit, bool) {
	return f.literal, f.Valid() && f.kind == CallableFacetFunctionLiteral
}

func (f CallableFacet) ABI() (*GeneratedArtifact, bool) {
	return f.generated, f.Valid() && f.kind == CallableFacetABI
}

func (f CallableFacet) GenericCapability() (*GeneratedArtifact, bool) {
	return f.generated,
		f.Valid() && f.kind == CallableFacetGenericCapability
}

func (f CallableFacet) GenericOperation() (
	*GenericOperationContract,
	bool,
) {
	return f.operation,
		f.Valid() && f.kind == CallableFacetGenericOperation
}

func (f CallableFacet) PackageInitializer() (ArtifactOwner, bool) {
	return f.owner,
		f.Valid() && f.kind == CallableFacetPackageInitializer
}

func (f CallableFacet) GenericProfile() (
	*GenericCallableProfile,
	bool,
) {
	return f.profile,
		f.Valid() && f.kind == CallableFacetGenericProfile
}

func functionType(function *types.Func) (*types.Signature, bool) {
	if function == nil {
		return nil, false
	}
	signature, ok := function.Type().(*types.Signature)
	return signature, ok
}

func operationOwnerFunction(
	operation *GenericOperationContract,
) (*types.Func, bool) {
	if operation == nil {
		return nil, false
	}
	function, ok := operation.Owner().(*types.Func)
	return function, ok && function != nil && function.Origin() == function
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
