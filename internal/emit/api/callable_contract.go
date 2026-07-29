package api

import (
	"go/ast"
	"go/types"
	"slices"
)

type CallableFacetKind uint8

const (
	CallableFacetInvalid         CallableFacetKind = 0
	CallableFacetSource          CallableFacetKind = 1
	CallableFacetFunctionLiteral CallableFacetKind = 2
	CallableFacetABI             CallableFacetKind = 3
)

func (k CallableFacetKind) Valid() bool {
	return k >= CallableFacetSource && k <= CallableFacetABI
}

type CallableFacet struct {
	owner    ArtifactOwner
	kind     CallableFacetKind
	function *types.Func
	literal  *ast.FuncLit
	abi      *GeneratedArtifact
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
		owner: MustGeneratedArtifactOwner(artifact),
		kind:  CallableFacetABI,
		abi:   artifact,
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
			f.abi == nil &&
			signature != nil
	case CallableFacetFunctionLiteral:
		_, sourceOwned := f.owner.Source()
		_, _, initializerOwned := f.owner.PackageInitializer()
		return (sourceOwned || initializerOwned) &&
			f.function == nil &&
			f.literal != nil &&
			f.literal.Type != nil &&
			f.literal.Body != nil &&
			f.abi == nil
	case CallableFacetABI:
		generated, generatedOwned := f.owner.Generated()
		return generatedOwned &&
			f.function == nil &&
			f.literal == nil &&
			f.abi == generated &&
			f.abi.Kind() == GeneratedArtifactCallableABI &&
			f.abi.Valid()
	default:
		return false
	}
}

func (f CallableFacet) empty() bool {
	return !f.owner.Valid() &&
		f.kind == CallableFacetInvalid &&
		f.function == nil &&
		f.literal == nil &&
		f.abi == nil
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
	return f.abi, f.Valid() && f.kind == CallableFacetABI
}

func functionType(function *types.Func) (*types.Signature, bool) {
	if function == nil {
		return nil, false
	}
	signature, ok := function.Type().(*types.Signature)
	return signature, ok
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
	for _, request := range requests {
		if request.Kind() == RootRequestInvalid {
			return CallableABIReference{}, &RootRequestError{
				Reason: "callable ABI reference request is invalid",
			}
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
	for _, request := range requests {
		if request.Kind() == RootRequestInvalid {
			return CooperativeCallableObservation{}, &RootRequestError{
				Reason: "cooperative callable observation has an invalid request",
			}
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
	return c.cooperativeResolver.ObserveCooperativeCallable(
		c.artifactOwner,
		facet,
	)
}

func (c Context) CooperativeRequest() (RootRequest, error) {
	if !c.callableFacet.Valid() {
		return RootRequest{}, &ContextError{
			Reason: "cooperative operation has no callable facet",
		}
	}
	return NewCooperativeCallableRequest(c.callableFacet)
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
