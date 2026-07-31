package gostdlib

import "slices"

type FacetKind string

const (
	FacetInvalid                FacetKind = ""
	FacetNamedStructOperations  FacetKind = "named-struct-operations"
	FacetRecoveryCallable       FacetKind = "recovery-callable"
	FacetGenericCallableProfile FacetKind = "generic-callable-profile"
)

func (k FacetKind) Valid() bool {
	return k == FacetNamedStructOperations ||
		k == FacetRecoveryCallable ||
		k == FacetGenericCallableProfile
}

type FacetCapability string

const (
	FacetCapabilityInvalid  FacetCapability = ""
	FacetCapabilityMake     FacetCapability = "make"
	FacetCapabilityZero     FacetCapability = "zero"
	FacetCapabilityCopy     FacetCapability = "copy"
	FacetCapabilityEqual    FacetCapability = "equal"
	FacetCapabilityHash     FacetCapability = "hash"
	FacetCapabilityConvert  FacetCapability = "convert"
	FacetCapabilityStorage  FacetCapability = "storage"
	FacetCapabilityAssign   FacetCapability = "assign"
	FacetCapabilityRecovery FacetCapability = "recovery"
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
		FacetCapabilityAssign:
		return true
	default:
		return false
	}
}

type EffectKind string

const (
	EffectInvalid      EffectKind = ""
	EffectSynchronous  EffectKind = "sync"
	EffectAsynchronous EffectKind = "async"
)

func (k EffectKind) Valid() bool {
	return k == EffectSynchronous || k == EffectAsynchronous
}

type FacetModuleDocument struct {
	Specifier  string          `json:"specifier"`
	SourcePath string          `json:"sourcePath"`
	Facets     []FacetDocument `json:"facets"`
}

type FacetDocument struct {
	Kind                       FacetKind         `json:"kind"`
	SourceIdentity             string            `json:"sourceIdentity"`
	Capabilities               []FacetCapability `json:"capabilities,omitempty"`
	ProfileKey                 string            `json:"profileKey,omitempty"`
	Export                     string            `json:"export"`
	StorageExport              string            `json:"storageExport,omitempty"`
	Effect                     EffectKind        `json:"effect,omitempty"`
	ImplementationOwner        string            `json:"implementationOwner"`
	StorageImplementationOwner string            `json:"storageImplementationOwner,omitempty"`
	TargetFingerprint          string            `json:"targetFingerprint"`
	StorageTargetFingerprint   string            `json:"storageTargetFingerprint,omitempty"`
}

type facetLookup struct {
	sourceIdentity string
	kind           FacetKind
	capability     string
}

type FacetModule struct {
	document FacetModuleDocument
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

type Facet struct {
	module FacetModuleDocument
	facet  FacetDocument
}

func newFacet(module FacetModuleDocument, facet FacetDocument) Facet {
	module.Facets = nil
	return Facet{module: module, facet: facet}
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

func (f Facet) ProfileKey() string {
	return f.facet.ProfileKey
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

func (f Facet) Effect() EffectKind {
	return f.facet.Effect
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
	result.Facets = make([]FacetDocument, len(source.Facets))
	for index, facet := range source.Facets {
		result.Facets[index] = facet
		result.Facets[index].Capabilities = slices.Clone(facet.Capabilities)
	}
	return result
}
