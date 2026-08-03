package naming

import (
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

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
