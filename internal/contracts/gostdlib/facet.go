package gostdlib

import (
	"slices"
	"strings"
)

type FacetKind string

const (
	FacetInvalid                FacetKind = ""
	FacetNamedStructOperations  FacetKind = "named-struct-operations"
	FacetDefinedValueOperations FacetKind = "defined-value-operations"
	FacetRecoveryCallable       FacetKind = "recovery-callable"
	FacetGenericCallableKernel  FacetKind = "generic-callable-kernel"
)

func (k FacetKind) Valid() bool {
	return k == FacetNamedStructOperations ||
		k == FacetDefinedValueOperations ||
		k == FacetRecoveryCallable ||
		k == FacetGenericCallableKernel
}

type FacetCapability string

const (
	FacetCapabilityInvalid        FacetCapability = ""
	FacetCapabilityMake           FacetCapability = "make"
	FacetCapabilityZero           FacetCapability = "zero"
	FacetCapabilityCopy           FacetCapability = "copy"
	FacetCapabilityEqual          FacetCapability = "equal"
	FacetCapabilityHash           FacetCapability = "hash"
	FacetCapabilityConvert        FacetCapability = "convert"
	FacetCapabilityStorage        FacetCapability = "storage"
	FacetCapabilityAssign         FacetCapability = "assign"
	FacetCapabilityRepresentation FacetCapability = "representation"
	FacetCapabilityRecovery       FacetCapability = "recovery"
	FacetCapabilityKernel         FacetCapability = "kernel"
	FacetCapabilityProject        FacetCapability = "project"
	FacetCapabilityWrap           FacetCapability = "wrap"
)

func (c FacetCapability) NamedStructOperation() bool {
	switch c {
	case FacetCapabilityMake,
		FacetCapabilityZero,
		FacetCapabilityCopy,
		FacetCapabilityEqual,
		FacetCapabilityHash,
		FacetCapabilityConvert,
		FacetCapabilityStorage,
		FacetCapabilityAssign,
		FacetCapabilityRepresentation:
		return true
	default:
		return false
	}
}

func (c FacetCapability) DefinedValueOperation() bool {
	return c == FacetCapabilityProject || c == FacetCapabilityWrap
}

type EffectKind string

const (
	EffectInvalid      EffectKind = ""
	EffectSynchronous  EffectKind = "sync"
	EffectAsynchronous EffectKind = "async"
	EffectAwaitable    EffectKind = "awaitable"
)

func (k EffectKind) Valid() bool {
	return k == EffectSynchronous ||
		k == EffectAsynchronous ||
		k == EffectAwaitable
}

func (k EffectKind) MaySuspend() bool {
	return k == EffectAsynchronous || k == EffectAwaitable
}

type FacetModuleDocument struct {
	Specifier          string                             `json:"specifier"`
	SourcePath         string                             `json:"sourcePath"`
	Representations    []ProviderRepresentationDocument   `json:"representations,omitempty"`
	ProviderInterfaces []ProviderInterfaceBindingDocument `json:"providerInterfaces,omitempty"`
	CallableProfiles   []ProviderCallableProfileDocument  `json:"callableProfiles,omitempty"`
	StatefulProfiles   []ProviderStatefulProfileDocument  `json:"statefulProfiles,omitempty"`
	Facets             []FacetDocument                    `json:"facets"`
}

type ProviderRepresentationDocument struct {
	Export              string                                 `json:"export"`
	SourceTypes         []string                               `json:"sourceTypes"`
	SourceInterfaces    []string                               `json:"sourceInterfaces"`
	Methods             []ProviderRepresentationMethodDocument `json:"methods"`
	ImplementationOwner string                                 `json:"implementationOwner"`
	TargetFingerprint   string                                 `json:"targetFingerprint"`
}

type ProviderRepresentationMethodDocument struct {
	SourceIdentity      string     `json:"sourceIdentity"`
	Member              string     `json:"member"`
	Effect              EffectKind `json:"effect"`
	SourceSignature     string     `json:"sourceSignature"`
	SourceLocation      string     `json:"sourceLocation"`
	ImplementationOwner string     `json:"implementationOwner"`
	TargetFingerprint   string     `json:"targetFingerprint"`
}

type FacetDocument struct {
	Kind                       FacetKind                     `json:"kind"`
	SourceIdentity             string                        `json:"sourceIdentity"`
	Capabilities               []FacetCapability             `json:"capabilities,omitempty"`
	Export                     string                        `json:"export"`
	StorageExport              string                        `json:"storageExport,omitempty"`
	RepresentationExport       string                        `json:"representationExport,omitempty"`
	Effect                     EffectKind                    `json:"effect,omitempty"`
	GenericTypeArguments       []GenericTypeArgumentDocument `json:"genericTypeArguments,omitempty"`
	ImplementationOwner        string                        `json:"implementationOwner"`
	StorageImplementationOwner string                        `json:"storageImplementationOwner,omitempty"`
	TargetFingerprint          string                        `json:"targetFingerprint"`
	StorageTargetFingerprint   string                        `json:"storageTargetFingerprint,omitempty"`
}

type facetLookup struct {
	sourceIdentity string
	kind           FacetKind
	capability     string
}

type providerRepresentationLookup struct {
	module string
	export string
}

type FacetModule struct {
	document FacetModuleDocument
}

func facetModuleIdentity(source FacetModuleDocument) FacetModuleDocument {
	return FacetModuleDocument{
		Specifier:  source.Specifier,
		SourcePath: source.SourcePath,
	}
}

func (m FacetModule) Specifier() string {
	return m.document.Specifier
}

func (m FacetModule) SourcePath() string {
	return m.document.SourcePath
}

func (m FacetModule) Facets() []Facet {
	result := make([]Facet, len(m.document.Facets))
	for index, facet := range m.document.Facets {
		result[index] = newFacet(m.document, facet)
	}
	return result
}

func (m FacetModule) Representations() []ProviderRepresentation {
	result := make([]ProviderRepresentation, len(m.document.Representations))
	for index, representation := range m.document.Representations {
		result[index] = newProviderRepresentation(m.document, representation)
	}
	return result
}

func (m FacetModule) ProviderInterfaces() []ProviderInterfaceBinding {
	result := make([]ProviderInterfaceBinding, len(m.document.ProviderInterfaces))
	for index, selected := range m.document.ProviderInterfaces {
		result[index] = newProviderInterfaceBinding(m.document, selected)
	}
	return result
}

func (m FacetModule) CallableProfiles() []ProviderCallableProfile {
	result := make([]ProviderCallableProfile, len(m.document.CallableProfiles))
	for index, profile := range m.document.CallableProfiles {
		result[index] = newProviderCallableProfile(m.document, profile)
	}
	return result
}

func (m FacetModule) StatefulProfiles() []ProviderStatefulProfile {
	result := make([]ProviderStatefulProfile, len(m.document.StatefulProfiles))
	for index, profile := range m.document.StatefulProfiles {
		result[index] = newProviderStatefulProfile(m.document, profile)
	}
	return result
}

type Facet struct {
	module         FacetModuleDocument
	facet          FacetDocument
	representation ProviderRepresentationDocument
}

func newFacet(module FacetModuleDocument, facet FacetDocument) Facet {
	result := Facet{module: facetModuleIdentity(module), facet: facet}
	if facet.RepresentationExport == "" {
		return result
	}
	for _, representation := range module.Representations {
		if representation.Export == facet.RepresentationExport {
			result.representation = cloneProviderRepresentation(representation)
			break
		}
	}
	return result
}

func (f Facet) Kind() FacetKind {
	return f.facet.Kind
}

func (f Facet) SourceIdentity() string {
	return f.facet.SourceIdentity
}

func (f Facet) Capabilities() []FacetCapability {
	return slices.Clone(f.facet.Capabilities)
}

func (f Facet) ModuleSpecifier() string {
	return f.module.Specifier
}

func (f Facet) Export() string {
	return f.facet.Export
}

func (f Facet) StorageExport() string {
	return f.facet.StorageExport
}

func (f Facet) Representation() (ProviderRepresentation, bool) {
	if f.facet.RepresentationExport == "" {
		return ProviderRepresentation{}, false
	}
	if f.representation.Export != f.facet.RepresentationExport {
		return ProviderRepresentation{}, false
	}
	return newProviderRepresentation(f.module, f.representation), true
}

func (f Facet) Effect() EffectKind {
	return f.facet.Effect
}

func (f Facet) GenericTypeArguments() []GenericTypeArgumentDocument {
	return slices.Clone(f.facet.GenericTypeArguments)
}

func (f Facet) ImplementationOwner() string {
	return f.facet.ImplementationOwner
}

func (f Facet) StorageImplementationOwner() string {
	return f.facet.StorageImplementationOwner
}

func (f Facet) TargetFingerprint() string {
	return f.facet.TargetFingerprint
}

func (f Facet) StorageTargetFingerprint() string {
	return f.facet.StorageTargetFingerprint
}

func cloneFacetModule(source FacetModuleDocument) FacetModuleDocument {
	result := source
	result.Representations = make(
		[]ProviderRepresentationDocument,
		len(source.Representations),
	)
	for index, representation := range source.Representations {
		result.Representations[index] = cloneProviderRepresentation(representation)
	}
	result.ProviderInterfaces = make(
		[]ProviderInterfaceBindingDocument,
		len(source.ProviderInterfaces),
	)
	for index, selected := range source.ProviderInterfaces {
		result.ProviderInterfaces[index] =
			cloneProviderInterfaceBinding(selected)
	}
	result.CallableProfiles = make(
		[]ProviderCallableProfileDocument,
		len(source.CallableProfiles),
	)
	for index, profile := range source.CallableProfiles {
		result.CallableProfiles[index] = cloneProviderCallableProfile(profile)
	}
	result.StatefulProfiles = make(
		[]ProviderStatefulProfileDocument,
		len(source.StatefulProfiles),
	)
	for index, profile := range source.StatefulProfiles {
		result.StatefulProfiles[index] = cloneProviderStatefulProfile(profile)
	}
	result.Facets = make([]FacetDocument, len(source.Facets))
	for index, facet := range source.Facets {
		result.Facets[index] = facet
		result.Facets[index].Capabilities = slices.Clone(facet.Capabilities)
		result.Facets[index].GenericTypeArguments = slices.Clone(
			facet.GenericTypeArguments,
		)
	}
	return result
}

type ProviderRepresentation struct {
	module         FacetModuleDocument
	representation ProviderRepresentationDocument
}

func newProviderRepresentation(
	module FacetModuleDocument,
	representation ProviderRepresentationDocument,
) ProviderRepresentation {
	return ProviderRepresentation{
		module:         facetModuleIdentity(module),
		representation: cloneProviderRepresentation(representation),
	}
}

func (r ProviderRepresentation) ModuleSpecifier() string {
	return r.module.Specifier
}

func (r ProviderRepresentation) Export() string {
	return r.representation.Export
}

func (r ProviderRepresentation) SourceTypes() []string {
	return slices.Clone(r.representation.SourceTypes)
}

func (r ProviderRepresentation) SourceInterfaces() []string {
	return slices.Clone(r.representation.SourceInterfaces)
}

func (r ProviderRepresentation) Method(
	sourceIdentity string,
) (ProviderRepresentationMethod, bool) {
	index, found := slices.BinarySearchFunc(
		r.representation.Methods,
		sourceIdentity,
		func(method ProviderRepresentationMethodDocument, identity string) int {
			return strings.Compare(method.SourceIdentity, identity)
		},
	)
	if !found {
		return ProviderRepresentationMethod{}, false
	}
	return ProviderRepresentationMethod{
		document: r.representation.Methods[index],
	}, true
}

func (r ProviderRepresentation) ImplementationOwner() string {
	return r.representation.ImplementationOwner
}

func (r ProviderRepresentation) TargetFingerprint() string {
	return r.representation.TargetFingerprint
}

type ProviderRepresentationMethod struct {
	document ProviderRepresentationMethodDocument
}

func (m ProviderRepresentationMethod) SourceIdentity() string {
	return m.document.SourceIdentity
}

func (m ProviderRepresentationMethod) Member() string {
	return m.document.Member
}

func (m ProviderRepresentationMethod) Effect() EffectKind {
	return m.document.Effect
}

func (m ProviderRepresentationMethod) SourceSignature() string {
	return m.document.SourceSignature
}

func (m ProviderRepresentationMethod) SourceLocation() string {
	return m.document.SourceLocation
}

func (m ProviderRepresentationMethod) ImplementationOwner() string {
	return m.document.ImplementationOwner
}

func (m ProviderRepresentationMethod) TargetFingerprint() string {
	return m.document.TargetFingerprint
}

func cloneProviderRepresentation(
	source ProviderRepresentationDocument,
) ProviderRepresentationDocument {
	result := source
	result.SourceTypes = slices.Clone(source.SourceTypes)
	result.SourceInterfaces = slices.Clone(source.SourceInterfaces)
	result.Methods = slices.Clone(source.Methods)
	return result
}
