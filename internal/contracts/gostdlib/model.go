package gostdlib

import "slices"

const (
	SchemaVersion = 2
	PackageName   = "@gotots/gostdlib"
)

type BindingKind string

const (
	BindingInvalid  BindingKind = ""
	BindingConstant BindingKind = "constant"
	BindingType     BindingKind = "type"
	BindingVariable BindingKind = "variable"
	BindingFunction BindingKind = "function"
	BindingBuiltin  BindingKind = "builtin"
)

func (k BindingKind) Valid() bool {
	return k == BindingConstant || k == BindingType ||
		k == BindingVariable || k == BindingFunction ||
		k == BindingBuiltin
}

type AccessKind string

const (
	AccessInvalid        AccessKind = ""
	AccessExport         AccessKind = "export"
	AccessStateMember    AccessKind = "state-member"
	AccessStaticMethod   AccessKind = "static-method"
	AccessInstanceMethod AccessKind = "instance-method"
)

func (k AccessKind) Valid() bool {
	return k == AccessExport || k == AccessStateMember ||
		k == AccessStaticMethod || k == AccessInstanceMethod
}

type RepresentationKind string

const (
	RepresentationInvalid RepresentationKind = ""
	RepresentationDirect  RepresentationKind = "direct"
)

func (k RepresentationKind) Valid() bool {
	return k == RepresentationDirect
}

type Document struct {
	SchemaVersion    int                   `json:"schemaVersion"`
	PackageName      string                `json:"packageName"`
	PackageVersion   string                `json:"packageVersion"`
	Backend          string                `json:"backend"`
	GoVersion        string                `json:"goVersion"`
	MinimumGoVersion string                `json:"minimumGoVersion"`
	MaximumGoVersion string                `json:"maximumGoVersion"`
	GOOS             string                `json:"goos"`
	GOARCH           string                `json:"goarch"`
	RuntimeDigest    string                `json:"runtimeDigest"`
	ProviderDigest   string                `json:"providerDigest"`
	Modules          []ModuleDocument      `json:"modules"`
	FacetModules     []FacetModuleDocument `json:"facetModules,omitempty"`
	ManifestDigest   string                `json:"manifestDigest,omitempty"`
}

type ModuleDocument struct {
	GoImportPath string            `json:"goImportPath"`
	Specifier    string            `json:"specifier"`
	SourcePath   string            `json:"sourcePath"`
	Bindings     []BindingDocument `json:"bindings"`
}

type BindingDocument struct {
	Identity            string             `json:"identity"`
	Kind                BindingKind        `json:"kind"`
	Access              AccessKind         `json:"access"`
	Representation      RepresentationKind `json:"representation,omitempty"`
	Export              string             `json:"export"`
	Member              string             `json:"member,omitempty"`
	SourceSignature     string             `json:"sourceSignature"`
	SourceValue         string             `json:"sourceValue,omitempty"`
	SourceLocation      string             `json:"sourceLocation"`
	ImplementationOwner string             `json:"implementationOwner"`
	TargetFingerprint   string             `json:"targetFingerprint"`
}

type Manifest struct {
	document Document
	payload  []byte
	bindings map[string]Binding
	facets   map[facetLookup]Facet
}

func (m Manifest) Digest() string {
	return m.document.ManifestDigest
}

func (m Manifest) PackageName() string {
	return m.document.PackageName
}

func (m Manifest) PackageVersion() string {
	return m.document.PackageVersion
}

func (m Manifest) Backend() string {
	return m.document.Backend
}

func (m Manifest) GoVersion() string {
	return m.document.GoVersion
}

func (m Manifest) MinimumGoVersion() string {
	return m.document.MinimumGoVersion
}

func (m Manifest) MaximumGoVersion() string {
	return m.document.MaximumGoVersion
}

func (m Manifest) GOOS() string {
	return m.document.GOOS
}

func (m Manifest) GOARCH() string {
	return m.document.GOARCH
}

func (m Manifest) RuntimeDigest() string {
	return m.document.RuntimeDigest
}

func (m Manifest) ProviderDigest() string {
	return m.document.ProviderDigest
}

func (m Manifest) Modules() []Module {
	result := make([]Module, len(m.document.Modules))
	for index, module := range m.document.Modules {
		result[index] = Module{document: cloneModule(module)}
	}
	return result
}

func (m Manifest) Binding(identity string) (Binding, bool) {
	selected, ok := m.bindings[identity]
	return selected, ok
}

func (m Manifest) FacetModules() []FacetModule {
	result := make([]FacetModule, len(m.document.FacetModules))
	for index, module := range m.document.FacetModules {
		result[index] = FacetModule{document: cloneFacetModule(module)}
	}
	return result
}

func (m Manifest) Facet(
	sourceIdentity string,
	kind FacetKind,
	capability FacetCapability,
) (Facet, bool) {
	selected, ok := m.facets[facetLookup{
		sourceIdentity: sourceIdentity,
		kind:           kind,
		capability:     string(capability),
	}]
	return selected, ok
}

func (m Manifest) GenericCallableFacet(
	sourceIdentity string,
	profileKey string,
) (Facet, bool) {
	selected, ok := m.facets[facetLookup{
		sourceIdentity: sourceIdentity,
		kind:           FacetGenericCallableProfile,
		capability:     profileKey,
	}]
	return selected, ok
}

type Module struct {
	document ModuleDocument
}

func (m Module) GoImportPath() string {
	return m.document.GoImportPath
}

func (m Module) Specifier() string {
	return m.document.Specifier
}

func (m Module) SourcePath() string {
	return m.document.SourcePath
}

func (m Module) Bindings() []Binding {
	result := make([]Binding, len(m.document.Bindings))
	for index, binding := range m.document.Bindings {
		result[index] = newBinding(m.document, binding)
	}
	return result
}

type Binding struct {
	module  ModuleDocument
	binding BindingDocument
}

func newBinding(module ModuleDocument, binding BindingDocument) Binding {
	module.Bindings = nil
	return Binding{module: module, binding: binding}
}

func (b Binding) Identity() string {
	return b.binding.Identity
}

func (b Binding) Kind() BindingKind {
	return b.binding.Kind
}

func (b Binding) Access() AccessKind {
	return b.binding.Access
}

func (b Binding) Representation() RepresentationKind {
	return b.binding.Representation
}

func (b Binding) Export() string {
	return b.binding.Export
}

func (b Binding) Member() string {
	return b.binding.Member
}

func (b Binding) SourceSignature() string {
	return b.binding.SourceSignature
}

func (b Binding) SourceValue() string {
	return b.binding.SourceValue
}

func (b Binding) SourceLocation() string {
	return b.binding.SourceLocation
}

func (b Binding) ImplementationOwner() string {
	return b.binding.ImplementationOwner
}

func (b Binding) TargetFingerprint() string {
	return b.binding.TargetFingerprint
}

func (b Binding) GoImportPath() string {
	return b.module.GoImportPath
}

func (b Binding) ModuleSpecifier() string {
	return b.module.Specifier
}

func cloneDocument(source Document) Document {
	result := source
	result.Modules = make([]ModuleDocument, len(source.Modules))
	for index, module := range source.Modules {
		result.Modules[index] = cloneModule(module)
	}
	result.FacetModules = make([]FacetModuleDocument, len(source.FacetModules))
	for index, module := range source.FacetModules {
		result.FacetModules[index] = cloneFacetModule(module)
	}
	return result
}

func cloneModule(source ModuleDocument) ModuleDocument {
	result := source
	result.Bindings = slices.Clone(source.Bindings)
	return result
}
