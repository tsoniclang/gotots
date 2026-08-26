package api

import (
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
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

type ExternalFunctionTargetKind uint8

const (
	ExternalFunctionTargetInvalid ExternalFunctionTargetKind = iota
	ExternalFunctionTargetModule
	ExternalFunctionTargetSource
)

type ExternalFunctionTarget struct {
	kind           ExternalFunctionTargetKind
	module         string
	export         string
	implementation *types.Func
}

type ExternalFunctionResolver interface {
	ResolveExternalFunction(*types.Func) (ExternalFunctionTarget, bool, error)
}

func NewExternalModuleFunctionTarget(
	module string,
	export string,
) (ExternalFunctionTarget, error) {
	if module == "" || export == "" {
		return ExternalFunctionTarget{}, &ContextError{
			Reason: "external module-function target is incomplete",
		}
	}
	return ExternalFunctionTarget{
		kind:   ExternalFunctionTargetModule,
		module: module,
		export: export,
	}, nil
}

func NewExternalSourceFunctionTarget(
	implementation *types.Func,
) (ExternalFunctionTarget, error) {
	if implementation == nil || implementation != implementation.Origin() {
		return ExternalFunctionTarget{}, &ContextError{
			Reason: "external source-function target is incomplete",
		}
	}
	return ExternalFunctionTarget{
		kind:           ExternalFunctionTargetSource,
		implementation: implementation,
	}, nil
}

func (t ExternalFunctionTarget) Kind() ExternalFunctionTargetKind {
	return t.kind
}

func (t ExternalFunctionTarget) Module() (string, string, bool) {
	return t.module, t.export, t.kind == ExternalFunctionTargetModule &&
		t.module != "" && t.export != "" && t.implementation == nil
}

func (t ExternalFunctionTarget) Source() (*types.Func, bool) {
	return t.implementation, t.kind == ExternalFunctionTargetSource &&
		t.implementation != nil && t.module == "" && t.export == ""
}

func (c Context) WithExternalFunctionResolver(
	resolver ExternalFunctionResolver,
) Context {
	if resolver == nil {
		panic("external function resolver is nil")
	}
	c.externalFunctionResolver = resolver
	return c
}

func (c Context) ResolveExternalFunction(
	function *types.Func,
) (ExternalFunctionTarget, bool, error) {
	if c.externalFunctionResolver == nil {
		return ExternalFunctionTarget{}, false, &ContextError{
			Reason: "external function resolver is unavailable",
		}
	}
	return c.externalFunctionResolver.ResolveExternalFunction(function)
}

func validateReferenceRequests(requests []RootRequest) error {
	return WalkUniqueRootRequestPayloads(requests, func(RootRequest) error {
		return nil
	})
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

func (o *GeneratedArtifact) Kind() GeneratedArtifactKind {
	if o == nil {
		return GeneratedArtifactInvalid
	}
	return o.kind
}

func (o *GeneratedArtifact) SourceType() types.Type {
	if o == nil {
		return nil
	}
	return o.sourceType
}

func (o *GeneratedArtifact) StructType() (*types.Struct, bool) {
	if o == nil || o.kind != GeneratedArtifactAnonymousStruct {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Struct)
	return source, ok
}

func (o *GeneratedArtifact) MapType() (*types.Map, bool) {
	if o == nil || o.kind != GeneratedArtifactMapSpecialization {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Map)
	return source, ok
}

func (o *GeneratedArtifact) InterfaceAdapterType() (types.Type, bool) {
	if o == nil || o.kind != GeneratedArtifactInterfaceAdapter {
		return nil, false
	}
	return o.sourceType, interfaceAdapterType(o.sourceType)
}

func (o *GeneratedArtifact) InterfaceType() (*types.Interface, bool) {
	if o == nil || o.kind != GeneratedArtifactAnonymousInterface {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).Underlying().(*types.Interface)
	return source, ok && source.IsMethodSet()
}

func (o *GeneratedArtifact) InterfaceMethodSignature() (*types.Signature, bool) {
	if o == nil || o.kind != GeneratedArtifactInterfaceMethodToken {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Signature)
	return source, ok && source.Recv() == nil
}

func (o *GeneratedArtifact) InterfaceMethodCallableSignature() (
	*types.Signature,
	bool,
) {
	if o == nil || o.kind != GeneratedArtifactInterfaceMethodCallable {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Signature)
	return source, ok && source.Recv() == nil
}

func (o *GeneratedArtifact) InterfaceMethodRuntime() (RuntimeSymbol, bool) {
	if o == nil || o.kind != GeneratedArtifactInterfaceMethodToken {
		return RuntimeInvalid, false
	}
	return o.runtime, true
}

func (o *GeneratedArtifact) InterfaceDynamicType() (types.Type, bool) {
	if o == nil || o.kind != GeneratedArtifactInterfaceDynamicTypeToken {
		return nil, false
	}
	return o.sourceType, interfaceAdapterType(o.sourceType)
}

func (o *GeneratedArtifact) ReflectionType() (
	types.Type,
	*types.TypeName,
	bool,
) {
	if o == nil || o.kind != GeneratedArtifactReflectionType ||
		o.reflectionType == nil {
		return nil, nil, false
	}
	return o.sourceType, o.reflectionType, true
}

func (o *GeneratedArtifact) GenericCapability() (
	*types.Signature,
	GenericOperationSelection,
	bool,
) {
	if o == nil || o.kind != GeneratedArtifactGenericCapability {
		return nil, GenericOperationSelection{}, false
	}
	signature, ok := types.Unalias(o.sourceType).(*types.Signature)
	return signature, o.generic, ok && o.generic.Valid()
}

func (o *GeneratedArtifact) CallableABI() (*types.Signature, bool) {
	if o == nil || o.kind != GeneratedArtifactCallableABI {
		return nil, false
	}
	signature, ok := types.Unalias(o.sourceType).(*types.Signature)
	return signature, ok && signature.Recv() == nil
}

func (o *GeneratedArtifact) DeferredCallableRegistry() (
	*types.Signature,
	bool,
) {
	if o == nil || o.kind != GeneratedArtifactDeferredCallableRegistry {
		return nil, false
	}
	signature, ok := types.Unalias(o.sourceType).(*types.Signature)
	return signature, ok && signature.Recv() == nil &&
		!ContainsGenericTypeParameter(signature)
}

func (o *GeneratedArtifact) GenericConcretization() (
	*GenericConcretization,
	bool,
) {
	if o == nil || o.kind != GeneratedArtifactGenericConcretization ||
		!o.concretization.Valid() {
		return nil, false
	}
	return o.concretization, true
}

func (o *GeneratedArtifact) ProviderInterfaceBridgeType() (*types.Named, bool) {
	if o == nil || o.kind != GeneratedArtifactProviderInterfaceBridge {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Named)
	if !ok {
		return nil, false
	}
	_, interfaceType := source.Underlying().(*types.Interface)
	return source, interfaceType && source.Obj() != nil
}

func (o *GeneratedArtifact) ProviderProfileInterfaceBridge() (
	*types.Named,
	[]gostdlib.ProviderCallableProfileInterface,
	bool,
) {
	if o == nil || o.kind != GeneratedArtifactProviderInterfaceBridge ||
		len(o.providerProfile) == 0 {
		return nil, nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Named)
	if !ok || source.Obj() == nil {
		return nil, nil, false
	}
	contract, ok := source.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return nil, nil, false
	}
	return source, slices.Clone(o.providerProfile), true
}

func (o *GeneratedArtifact) ProviderStatefulRepresentationType() (*types.Named, bool) {
	if o == nil || o.kind != GeneratedArtifactProviderStatefulRepresentation {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Named)
	if !ok || source.Obj() == nil {
		return nil, false
	}
	_, interfaceType := source.Underlying().(*types.Interface)
	return source, !interfaceType
}
