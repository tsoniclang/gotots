package naming

import (
	"go/ast"
	"go/types"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

// EnvironmentObserver is the non-optional root environment selection
// observer consumed by every file name owner. Every target selection whose
// object may be environment-owned must synchronously observe the canonical
// object with one closed demand and typed provider selection before its
// target is returned; a selection route that can produce an environment
// target without this observation is invalid.
type EnvironmentObserver interface {
	RequireUse(
		object types.Object,
		demand environmentcontract.UseDemand,
		selection gostdlib.UseSelection,
	) error
	ObserveImplementation(
		object types.Object,
		demand environmentcontract.UseDemand,
		route environmentcontract.ImplementationRoute,
	) error
}

// referenceDemand classifies the closed environment use demand of one
// ordinary reference from the referenced object kind and import phase.
func referenceDemand(
	object types.Object,
	phase api.ImportPhase,
) environmentcontract.UseDemand {
	if phase == api.ImportPhaseType {
		return environmentcontract.UseDemandTypeContract
	}
	if _, ok := object.(*types.Func); ok {
		return environmentcontract.UseDemandCallable
	}
	return environmentcontract.UseDemandValue
}

func (n *File) requireUse(
	object types.Object,
	demand environmentcontract.UseDemand,
) error {
	return n.observer.RequireUse(object, demand, gostdlib.NoUseSelection())
}

// ObserveEnvironmentImplementation forwards a compiler-intrinsic or
// generated-runtime-facet implementation route to the non-optional root
// environment observer.
func (n *File) ObserveEnvironmentImplementation(
	object types.Object,
	demand environmentcontract.UseDemand,
	route environmentcontract.ImplementationRoute,
) error {
	return n.observer.ObserveImplementation(object, demand, route)
}

type targetBinding struct {
	name                         string
	sourceFile                   *ast.File
	sourcePath                   string
	moduleExport                 bool
	kind                         targetBindingKind
	providerModule               string
	providerExport               string
	providerMember               string
	providerAccess               gostdlib.AccessKind
	providerRepresentation       bool
	providerTypeRepresentation   gostdlib.RepresentationKind
	providerDefinedValue         gostdlib.DefinedValueRepresentationKind
	providerEffect               gostdlib.EffectKind
	providerGenericTypeArguments []gostdlib.GenericTypeArgumentDocument
	providerGenericOperations    []gostdlib.GenericOperationDocument
}

type targetBindingKind uint8

const (
	targetBindingLocal targetBindingKind = iota
	targetBindingSource
	targetBindingEnvironment
	targetBindingProvider
	targetBindingMissingProvider
)

func (b targetBinding) scheduled() bool {
	return b.kind == targetBindingSource ||
		b.kind == targetBindingEnvironment ||
		b.kind == targetBindingProvider
}

func (b targetBinding) sourceOwned() bool {
	return b.kind == targetBindingSource
}

type packageVariableBinding struct {
	fieldName    string
	statePath    string
	assemblyPath string
}

type anonymousStructBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type mapSpecializationBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type interfaceAdapterBinding struct {
	owner *api.GeneratedArtifact
	name  string
	key   string
}

type anonymousInterfaceBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type interfaceMethodTokenBinding struct {
	owner  *api.GeneratedArtifact
	method *types.Func
	name   string
}

type interfaceMethodCallableBinding struct {
	owner  *api.GeneratedArtifact
	method *types.Func
	name   string
}

type interfaceDynamicTypeTokenBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type providerInterfaceBridgeBinding struct {
	owner *api.GeneratedArtifact
	name  string
	key   string
}

type providerInterfaceCapabilityBinding struct {
	key         string
	certificate gostdlib.ProviderInterfaceCapability
	base        *types.Named
	target      *types.Interface
	targetKey   string
	demandKey   string
}

type providerStatefulRepresentationBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type interfaceContractDemand struct {
	source *types.Interface
	target *types.Interface
}

type interfaceReflectionDemand struct {
	source         *types.Interface
	reflectionType *types.TypeName
}

type genericCapabilityBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type genericConcretizationBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type callableABIBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type deferredCallableRegistryBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type pointerRepresentationBinding struct {
	owner *api.GeneratedArtifact
}

type reflectionTypeBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type unsafeCodecBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type Target struct {
	Name       string
	SourcePath string
}

type Registry struct {
	provider                            standardLibraryProvider
	providerImportNameByModule          map[string]string
	byObject                            map[types.Object]targetBinding
	memberNameByObject                  map[*types.Var]string
	packageVariables                    map[*types.Var]packageVariableBinding
	assemblyPathByPackage               map[*types.Package]string
	importQualifierByPackage            map[*types.Package]string
	anonymousStructs                    map[string]anonymousStructBinding
	anonymousStructNames                map[string]string
	mapSpecializations                  map[string]mapSpecializationBinding
	mapSpecializationNames              map[string]string
	interfaceAdapters                   map[string]interfaceAdapterBinding
	interfaceAdapterNames               map[string]string
	anonymousInterfaces                 map[string]anonymousInterfaceBinding
	anonymousInterfaceNames             map[string]string
	interfaceMethodCallables            map[string]interfaceMethodCallableBinding
	interfaceMethodCallableNames        map[string]string
	interfaceMethodTokens               map[string]interfaceMethodTokenBinding
	interfaceMethodNames                map[string]string
	interfaceDynamicTypes               map[string]interfaceDynamicTypeTokenBinding
	interfaceDynamicNames               map[string]string
	providerInterfaceBridges            map[string]providerInterfaceBridgeBinding
	providerInterfaceBridgeNames        map[string]string
	providerInterfaceCapabilities       map[string]map[string]providerInterfaceCapabilityBinding
	providerInterfaceCapabilityDemands  map[string]providerInterfaceCapabilityBinding
	providerInterfaceBridgesByContract  map[string]map[string]struct{}
	providerStatefulRepresentations     map[string]providerStatefulRepresentationBinding
	providerStatefulRepresentationNames map[string]string
	providerObjectByIdentity            map[string]types.Object
	interfaceContracts                  map[string]*types.Interface
	interfaceAdaptersByContract         map[string]map[string]struct{}
	interfaceContractDemands            map[string]map[string]interfaceContractDemand
	interfaceReflectionDemands          map[string]interfaceReflectionDemand
	genericCapabilities                 map[string]genericCapabilityBinding
	genericCapabilityNames              map[string]string
	genericConcretizations              map[string]genericConcretizationBinding
	genericConcretizationNames          map[string]string
	callableABIs                        map[string]callableABIBinding
	callableABINames                    map[string]string
	deferredCallableRegistries          map[string]deferredCallableRegistryBinding
	deferredCallableRegistryNames       map[string]string
	pointerRepresentations              map[string]pointerRepresentationBinding
	reflectionTypes                     map[string]reflectionTypeBinding
	reflectionTypeNames                 map[string]string
	unsafeCodecs                        map[string]unsafeCodecBinding
	unsafeCodecNames                    map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		byObject:                            make(map[types.Object]targetBinding),
		providerImportNameByModule:          make(map[string]string),
		memberNameByObject:                  make(map[*types.Var]string),
		packageVariables:                    make(map[*types.Var]packageVariableBinding),
		assemblyPathByPackage:               make(map[*types.Package]string),
		importQualifierByPackage:            make(map[*types.Package]string),
		anonymousStructs:                    make(map[string]anonymousStructBinding),
		anonymousStructNames:                make(map[string]string),
		mapSpecializations:                  make(map[string]mapSpecializationBinding),
		mapSpecializationNames:              make(map[string]string),
		interfaceAdapters:                   make(map[string]interfaceAdapterBinding),
		interfaceAdapterNames:               make(map[string]string),
		anonymousInterfaces:                 make(map[string]anonymousInterfaceBinding),
		anonymousInterfaceNames:             make(map[string]string),
		interfaceMethodCallables:            make(map[string]interfaceMethodCallableBinding),
		interfaceMethodCallableNames:        make(map[string]string),
		interfaceMethodTokens:               make(map[string]interfaceMethodTokenBinding),
		interfaceMethodNames:                make(map[string]string),
		interfaceDynamicTypes:               make(map[string]interfaceDynamicTypeTokenBinding),
		interfaceDynamicNames:               make(map[string]string),
		providerInterfaceBridges:            make(map[string]providerInterfaceBridgeBinding),
		providerInterfaceBridgeNames:        make(map[string]string),
		providerInterfaceCapabilities:       make(map[string]map[string]providerInterfaceCapabilityBinding),
		providerInterfaceCapabilityDemands:  make(map[string]providerInterfaceCapabilityBinding),
		providerInterfaceBridgesByContract:  make(map[string]map[string]struct{}),
		providerStatefulRepresentations:     make(map[string]providerStatefulRepresentationBinding),
		providerStatefulRepresentationNames: make(map[string]string),
		providerObjectByIdentity:            make(map[string]types.Object),
		interfaceContracts:                  make(map[string]*types.Interface),
		interfaceAdaptersByContract:         make(map[string]map[string]struct{}),
		interfaceContractDemands:            make(map[string]map[string]interfaceContractDemand),
		interfaceReflectionDemands:          make(map[string]interfaceReflectionDemand),
		genericCapabilities:                 make(map[string]genericCapabilityBinding),
		genericCapabilityNames:              make(map[string]string),
		genericConcretizations:              make(map[string]genericConcretizationBinding),
		genericConcretizationNames:          make(map[string]string),
		callableABIs:                        make(map[string]callableABIBinding),
		callableABINames:                    make(map[string]string),
		deferredCallableRegistries:          make(map[string]deferredCallableRegistryBinding),
		deferredCallableRegistryNames:       make(map[string]string),
		pointerRepresentations:              make(map[string]pointerRepresentationBinding),
		reflectionTypes:                     make(map[string]reflectionTypeBinding),
		reflectionTypeNames:                 make(map[string]string),
		unsafeCodecs:                        make(map[string]unsafeCodecBinding),
		unsafeCodecNames:                    make(map[string]string),
	}
}

func (r *Registry) Target(object types.Object) (Target, bool) {
	if r == nil {
		return Target{}, false
	}
	binding, ok := r.byObject[object]
	if !ok {
		return Target{}, false
	}
	return Target{
		Name:       binding.name,
		SourcePath: binding.sourcePath,
	}, true
}

func (r *Registry) HasProviderCoverageOwner(object types.Object) bool {
	if r == nil || r.provider == nil || !r.provider.Valid() {
		return false
	}
	binding, ok := r.byObject[object]
	return ok && (binding.kind == targetBindingProvider ||
		binding.kind == targetBindingMissingProvider)
}

// EnvironmentSelectionRoute reports the canonical implementation route
// derived from one environment-owned declaration binding. The second result
// is false when the object has no environment binding. A certified provider
// binding selects the provider route; a contract-only binding selects the
// explicit boundary route; a selected declaration whose provider binding is
// missing reports an invalid route so the caller fails closed.
func (r *Registry) EnvironmentSelectionRoute(
	object types.Object,
) (environmentcontract.ImplementationRoute, bool) {
	if r == nil {
		return environmentcontract.RouteInvalid, false
	}
	binding, ok := r.byObject[object]
	if !ok {
		return environmentcontract.RouteInvalid, false
	}
	switch binding.kind {
	case targetBindingProvider:
		return environmentcontract.RouteProvider, true
	case targetBindingEnvironment:
		return environmentcontract.RouteBoundary, true
	case targetBindingMissingProvider:
		return environmentcontract.RouteInvalid, true
	default:
		return environmentcontract.RouteInvalid, false
	}
}

func (r *Registry) ImportQualifier(sourcePackage *types.Package) string {
	if r == nil {
		return ""
	}
	return r.importQualifierByPackage[sourcePackage]
}

func (r *Registry) GeneratedArtifact(
	kind api.GeneratedArtifactKind,
	artifactKey string,
) (*api.GeneratedArtifact, bool) {
	if r == nil || !kind.Valid() || artifactKey == "" {
		return nil, false
	}
	switch kind {
	case api.GeneratedArtifactAnonymousStruct:
		binding, ok := r.anonymousStructs[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactMapSpecialization:
		binding, ok := r.mapSpecializations[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactInterfaceAdapter:
		binding, ok := r.interfaceAdapters[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactAnonymousInterface:
		binding, ok := r.anonymousInterfaces[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactInterfaceMethodCallable:
		binding, ok := r.interfaceMethodCallables[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactInterfaceMethodToken:
		binding, ok := r.interfaceMethodTokens[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		binding, ok := r.interfaceDynamicTypes[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactProviderInterfaceBridge:
		binding, ok := r.providerInterfaceBridges[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactProviderStatefulRepresentation:
		binding, ok := r.providerStatefulRepresentations[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactGenericCapability:
		binding, ok := r.genericCapabilities[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactGenericConcretization:
		binding, ok := r.genericConcretizations[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactCallableABI:
		binding, ok := r.callableABIs[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactDeferredCallableRegistry:
		binding, ok := r.deferredCallableRegistries[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactPointerRepresentation:
		binding, ok := r.pointerRepresentations[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactReflectionType:
		binding, ok := r.reflectionTypes[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactUnsafeCodec:
		binding, ok := r.unsafeCodecs[artifactKey]
		return binding.owner, ok && binding.owner != nil
	default:
		return nil, false
	}
}

func (r *Registry) GeneratedArtifacts(
	kind api.GeneratedArtifactKind,
) []*api.GeneratedArtifact {
	if r == nil || !kind.Valid() {
		return nil
	}
	var artifacts []*api.GeneratedArtifact
	switch kind {
	case api.GeneratedArtifactAnonymousStruct:
		for _, binding := range r.anonymousStructs {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactMapSpecialization:
		for _, binding := range r.mapSpecializations {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactInterfaceAdapter:
		for _, binding := range r.interfaceAdapters {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactAnonymousInterface:
		for _, binding := range r.anonymousInterfaces {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactInterfaceMethodCallable:
		for _, binding := range r.interfaceMethodCallables {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactInterfaceMethodToken:
		for _, binding := range r.interfaceMethodTokens {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		for _, binding := range r.interfaceDynamicTypes {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactProviderInterfaceBridge:
		for _, binding := range r.providerInterfaceBridges {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactProviderStatefulRepresentation:
		for _, binding := range r.providerStatefulRepresentations {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactGenericCapability:
		for _, binding := range r.genericCapabilities {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactGenericConcretization:
		for _, binding := range r.genericConcretizations {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactCallableABI:
		for _, binding := range r.callableABIs {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactDeferredCallableRegistry:
		for _, binding := range r.deferredCallableRegistries {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactPointerRepresentation:
		for _, binding := range r.pointerRepresentations {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactReflectionType:
		for _, binding := range r.reflectionTypes {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactUnsafeCodec:
		for _, binding := range r.unsafeCodecs {
			artifacts = append(artifacts, binding.owner)
		}
	}
	sort.Slice(artifacts, func(left, right int) bool {
		return artifacts[left].ArtifactKey() < artifacts[right].ArtifactKey()
	})
	return artifacts
}

func (r *Registry) reserve(
	object types.Object,
	binding targetBinding,
) error {
	if r == nil {
		return &api.NameError{Name: objectName(object), Reason: "declaration registry is nil"}
	}
	if existing, ok := r.byObject[object]; ok {
		if existing.sourceFile != binding.sourceFile ||
			existing.sourcePath != binding.sourcePath ||
			existing.name != binding.name ||
			existing.kind != binding.kind ||
			existing.providerModule != binding.providerModule ||
			existing.providerExport != binding.providerExport ||
			existing.providerMember != binding.providerMember ||
			existing.providerAccess != binding.providerAccess ||
			existing.providerRepresentation != binding.providerRepresentation ||
			existing.providerTypeRepresentation != binding.providerTypeRepresentation ||
			existing.providerDefinedValue != binding.providerDefinedValue {
			return &api.NameError{
				Name:   objectName(object),
				Reason: "declaration has conflicting target ownership",
			}
		}
		return nil
	}
	r.byObject[object] = binding
	return nil
}
