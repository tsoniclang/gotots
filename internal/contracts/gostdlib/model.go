package gostdlib

import (
	"slices"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

const (
	SchemaVersion = 27
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

type DefinedValueRepresentationKind string

const (
	DefinedValueRepresentationInvalid    DefinedValueRepresentationKind = ""
	DefinedValueRepresentationCanonical  DefinedValueRepresentationKind = "canonical"
	DefinedValueRepresentationOperations DefinedValueRepresentationKind = "operations"
)

func (k DefinedValueRepresentationKind) Valid() bool {
	return k == DefinedValueRepresentationCanonical ||
		k == DefinedValueRepresentationOperations
}

type GenericTypeArgumentFacet string

const (
	GenericTypeArgumentInvalid          GenericTypeArgumentFacet = ""
	GenericTypeArgumentLogical          GenericTypeArgumentFacet = "logical"
	GenericTypeArgumentStorage          GenericTypeArgumentFacet = "storage"
	GenericTypeArgumentContainerStorage GenericTypeArgumentFacet = "container-storage"
	GenericTypeArgumentPointer          GenericTypeArgumentFacet = "pointer"
)

func (f GenericTypeArgumentFacet) Valid() bool {
	return f == GenericTypeArgumentLogical ||
		f == GenericTypeArgumentStorage ||
		f == GenericTypeArgumentContainerStorage ||
		f == GenericTypeArgumentPointer
}

type GenericTypeArgumentDocument struct {
	TypeParameter int                      `json:"typeParameter"`
	Facet         GenericTypeArgumentFacet `json:"facet"`
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
	CGOEnabled       bool                  `json:"cgoEnabled"`
	BuildTags        []string              `json:"buildTags"`
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
	Identity             string                              `json:"identity"`
	Kind                 BindingKind                         `json:"kind"`
	Access               AccessKind                          `json:"access"`
	Representation       RepresentationKind                  `json:"representation,omitempty"`
	DefinedValue         DefinedValueRepresentationKind      `json:"definedValue,omitempty"`
	Effect               EffectKind                          `json:"effect,omitempty"`
	Export               string                              `json:"export"`
	Member               string                              `json:"member,omitempty"`
	GenericTypeArguments []GenericTypeArgumentDocument       `json:"genericTypeArguments,omitempty"`
	GenericOperations    []GenericOperationDocument          `json:"genericOperations,omitempty"`
	CallableParameters   []ProviderCallableParameterDocument `json:"callableParameters,omitempty"`
	ProviderInterface    *ProviderInterfaceDocument          `json:"providerInterface,omitempty"`
	StructFields         []ProviderStructFieldDocument       `json:"structFields,omitempty"`
	SourceSignature      string                              `json:"sourceSignature"`
	SourceValue          string                              `json:"sourceValue,omitempty"`
	SourceLocation       string                              `json:"sourceLocation"`
	ImplementationOwner  string                              `json:"implementationOwner"`
	TargetFingerprint    string                              `json:"targetFingerprint"`
}

type Manifest struct {
	document             Document
	payload              []byte
	bindings             map[string]Binding
	facets               map[facetLookup]Facet
	representations      map[providerRepresentationLookup]ProviderRepresentation
	providerInterfaces   map[string]ProviderInterfaceBinding
	providerCapabilities map[string][]ProviderInterfaceCapability
	callableProfiles     map[providerCallableProfileLookup]ProviderCallableProfile
	statefulProfiles     map[providerStatefulProfileLookup]ProviderStatefulProfile
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

func (m Manifest) BuildProfile() (environmentcontract.BuildProfile, bool) {
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		m.document.GoVersion,
		m.document.GOOS,
		m.document.GOARCH,
		m.document.CGOEnabled,
		m.document.BuildTags,
	)
	return profile, err == nil
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

func (m Manifest) GenericCallableKernel(
	sourceIdentity string,
) (Facet, bool) {
	selected, ok := m.facets[facetLookup{
		sourceIdentity: sourceIdentity,
		kind:           FacetGenericCallableKernel,
		capability:     string(FacetCapabilityKernel),
	}]
	return selected, ok
}

func (m Manifest) ProviderRepresentation(
	module string,
	export string,
) (ProviderRepresentation, bool) {
	selected, ok := m.representations[providerRepresentationLookup{
		module: module,
		export: export,
	}]
	return selected, ok
}

func (m Manifest) ProviderInterface(
	sourceIdentity string,
) (ProviderInterfaceBinding, bool) {
	selected, ok := m.providerInterfaces[sourceIdentity]
	return selected, ok
}

func (m Manifest) ProviderInterfaceCapabilities(
	baseSourceIdentity string,
) []ProviderInterfaceCapability {
	source := m.providerCapabilities[baseSourceIdentity]
	result := make([]ProviderInterfaceCapability, len(source))
	copy(result, source)
	return result
}

func (m Manifest) ProviderCallableProfile(
	sourceIdentity string,
	profileKey string,
) (ProviderCallableProfile, bool) {
	selected, ok := m.callableProfiles[providerCallableProfileLookup{
		sourceIdentity: sourceIdentity,
		profileKey:     profileKey,
	}]
	return selected, ok
}

func (m Manifest) ProviderCallableProfiles(
	sourceIdentity string,
) []ProviderCallableProfile {
	var result []ProviderCallableProfile
	for lookup, selected := range m.callableProfiles {
		if lookup.sourceIdentity == sourceIdentity {
			result = append(result, selected)
		}
	}
	slices.SortFunc(result, func(left, right ProviderCallableProfile) int {
		return strings.Compare(left.ProfileKey(), right.ProfileKey())
	})
	return result
}

func (m Manifest) ProviderStatefulProfile(
	sourceIdentity string,
	profileKey string,
) (ProviderStatefulProfile, bool) {
	selected, ok := m.statefulProfiles[providerStatefulProfileLookup{
		sourceIdentity: sourceIdentity,
		profileKey:     profileKey,
	}]
	return selected, ok
}

func (m Manifest) ProviderStatefulProfiles(
	sourceIdentity string,
) []ProviderStatefulProfile {
	var result []ProviderStatefulProfile
	for lookup, selected := range m.statefulProfiles {
		if lookup.sourceIdentity == sourceIdentity {
			result = append(result, selected)
		}
	}
	slices.SortFunc(result, func(left, right ProviderStatefulProfile) int {
		return strings.Compare(left.ProfileKey(), right.ProfileKey())
	})
	return result
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

func (b Binding) DefinedValueRepresentation() DefinedValueRepresentationKind {
	return b.binding.DefinedValue
}

func (b Binding) Effect() EffectKind {
	return b.binding.Effect
}

func (b Binding) Export() string {
	return b.binding.Export
}

func (b Binding) Member() string {
	return b.binding.Member
}

func (b Binding) GenericOperations() []GenericOperationDocument {
	return cloneGenericOperations(b.binding.GenericOperations)
}

func (b Binding) GenericTypeArguments() []GenericTypeArgumentDocument {
	return slices.Clone(b.binding.GenericTypeArguments)
}

func (b Binding) CallableParameters() []ProviderCallableParameterDocument {
	return slices.Clone(b.binding.CallableParameters)
}

func (b Binding) ProviderInterface() (ProviderInterface, bool) {
	if b.binding.ProviderInterface == nil {
		return ProviderInterface{}, false
	}
	return newProviderInterface(*b.binding.ProviderInterface), true
}

func (b Binding) StructFields() []ProviderStructField {
	result := make([]ProviderStructField, len(b.binding.StructFields))
	for index, selected := range b.binding.StructFields {
		result[index] = ProviderStructField{document: selected}
	}
	return result
}

func (b Binding) StructField(member string) (ProviderStructField, bool) {
	index, found := slices.BinarySearchFunc(
		b.binding.StructFields,
		member,
		func(field ProviderStructFieldDocument, selected string) int {
			return strings.Compare(field.Member, selected)
		},
	)
	if !found {
		return ProviderStructField{}, false
	}
	return ProviderStructField{document: b.binding.StructFields[index]}, true
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
	result.BuildTags = slices.Clone(source.BuildTags)
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
	result.Bindings = make([]BindingDocument, len(source.Bindings))
	for index, binding := range source.Bindings {
		result.Bindings[index] = binding
		result.Bindings[index].GenericTypeArguments =
			slices.Clone(binding.GenericTypeArguments)
		result.Bindings[index].GenericOperations =
			cloneGenericOperations(binding.GenericOperations)
		result.Bindings[index].CallableParameters =
			slices.Clone(binding.CallableParameters)
		result.Bindings[index].StructFields =
			slices.Clone(binding.StructFields)
		if binding.ProviderInterface != nil {
			cloned := cloneProviderInterface(*binding.ProviderInterface)
			result.Bindings[index].ProviderInterface = &cloned
		}
	}
	return result
}
