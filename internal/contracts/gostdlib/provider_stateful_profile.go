package gostdlib

import (
	"slices"
	"strings"
)

type providerStatefulProfileLookup struct {
	sourceIdentity string
	profileKey     string
}

type ProviderStatefulProfileDocument struct {
	SourceIdentity      string                                     `json:"sourceIdentity"`
	ProfileKey          string                                     `json:"profileKey"`
	Export              string                                     `json:"export"`
	Interfaces          []ProviderCallableProfileInterfaceDocument `json:"interfaces"`
	TypeArguments       []string                                   `json:"typeArguments"`
	Operations          []FacetCapability                          `json:"operations,omitempty"`
	Fields              []ProviderStatefulProfileFieldDocument     `json:"fields"`
	Methods             []ProviderStatefulProfileMethodDocument    `json:"methods"`
	ImplementationOwner string                                     `json:"implementationOwner"`
	TargetFingerprint   string                                     `json:"targetFingerprint"`
}

type ProviderStatefulProfileFieldDocument struct {
	Member              string `json:"member"`
	Ordinal             int    `json:"ordinal"`
	Embedded            bool   `json:"embedded"`
	SourceSignature     string `json:"sourceSignature"`
	SourceLocation      string `json:"sourceLocation"`
	ImplementationOwner string `json:"implementationOwner"`
	TargetFingerprint   string `json:"targetFingerprint"`
}

type ProviderStatefulProfileMethodDocument struct {
	SourceIdentity            string     `json:"sourceIdentity"`
	Member                    string     `json:"member"`
	Effect                    EffectKind `json:"effect"`
	SourceSignature           string     `json:"sourceSignature"`
	SourceLocation            string     `json:"sourceLocation"`
	ImplementationOwner       string     `json:"implementationOwner"`
	InstanceTargetFingerprint string     `json:"instanceTargetFingerprint"`
	StaticTargetFingerprint   string     `json:"staticTargetFingerprint"`
}

type ProviderStatefulProfile struct {
	module  FacetModuleDocument
	profile ProviderStatefulProfileDocument
}

func newProviderStatefulProfile(
	module FacetModuleDocument,
	profile ProviderStatefulProfileDocument,
) ProviderStatefulProfile {
	return ProviderStatefulProfile{
		module:  facetModuleIdentity(module),
		profile: cloneProviderStatefulProfile(profile),
	}
}

func (p ProviderStatefulProfile) Valid() bool {
	return p.profile.SourceIdentity != "" &&
		p.profile.ProfileKey != "" &&
		p.module.Specifier != "" &&
		p.profile.Export != "" &&
		len(p.profile.Interfaces) != 0 &&
		len(p.profile.TypeArguments) != 0 &&
		len(p.profile.Methods) != 0 &&
		p.profile.ImplementationOwner != "" &&
		p.profile.TargetFingerprint != ""
}

func (p ProviderStatefulProfile) SourceIdentity() string {
	return p.profile.SourceIdentity
}

func (p ProviderStatefulProfile) ProfileKey() string {
	return p.profile.ProfileKey
}

func (p ProviderStatefulProfile) ModuleSpecifier() string {
	return p.module.Specifier
}

func (p ProviderStatefulProfile) Export() string {
	return p.profile.Export
}

func (p ProviderStatefulProfile) Interfaces() []ProviderCallableProfileInterface {
	result := make([]ProviderCallableProfileInterface, 0, len(p.profile.Interfaces))
	for _, selected := range p.profile.Interfaces {
		result = append(result, ProviderCallableProfileInterface{
			document: cloneProviderCallableProfileInterface(selected),
		})
	}
	return result
}

func (p ProviderStatefulProfile) TypeArguments() []string {
	return slices.Clone(p.profile.TypeArguments)
}

func (p ProviderStatefulProfile) Operations() []FacetCapability {
	return slices.Clone(p.profile.Operations)
}

func (p ProviderStatefulProfile) SupportsOperation(
	capability FacetCapability,
) bool {
	return slices.Contains(p.profile.Operations, capability)
}

func (p ProviderStatefulProfile) Interface(
	sourceIdentity string,
) (ProviderCallableProfileInterface, bool) {
	index, found := slices.BinarySearchFunc(
		p.profile.Interfaces,
		sourceIdentity,
		func(selected ProviderCallableProfileInterfaceDocument, identity string) int {
			return strings.Compare(selected.SourceIdentity, identity)
		},
	)
	if !found {
		return ProviderCallableProfileInterface{}, false
	}
	return ProviderCallableProfileInterface{
		document: cloneProviderCallableProfileInterface(p.profile.Interfaces[index]),
	}, true
}

func (p ProviderStatefulProfile) Methods() []ProviderStatefulProfileMethod {
	result := make([]ProviderStatefulProfileMethod, len(p.profile.Methods))
	for index, selected := range p.profile.Methods {
		result[index] = ProviderStatefulProfileMethod{document: selected}
	}
	return result
}

func (p ProviderStatefulProfile) Fields() []ProviderStatefulProfileField {
	result := make([]ProviderStatefulProfileField, len(p.profile.Fields))
	for index, selected := range p.profile.Fields {
		result[index] = ProviderStatefulProfileField{document: selected}
	}
	return result
}

func (p ProviderStatefulProfile) Field(
	member string,
) (ProviderStatefulProfileField, bool) {
	index, found := slices.BinarySearchFunc(
		p.profile.Fields,
		member,
		func(field ProviderStatefulProfileFieldDocument, selected string) int {
			return strings.Compare(field.Member, selected)
		},
	)
	if !found {
		return ProviderStatefulProfileField{}, false
	}
	return ProviderStatefulProfileField{document: p.profile.Fields[index]}, true
}

type ProviderStatefulProfileField struct {
	document ProviderStatefulProfileFieldDocument
}

func (f ProviderStatefulProfileField) Member() string {
	return f.document.Member
}

func (f ProviderStatefulProfileField) Ordinal() int {
	return f.document.Ordinal
}

func (f ProviderStatefulProfileField) Embedded() bool {
	return f.document.Embedded
}

func (f ProviderStatefulProfileField) SourceSignature() string {
	return f.document.SourceSignature
}

func (f ProviderStatefulProfileField) SourceLocation() string {
	return f.document.SourceLocation
}

func (f ProviderStatefulProfileField) ImplementationOwner() string {
	return f.document.ImplementationOwner
}

func (f ProviderStatefulProfileField) TargetFingerprint() string {
	return f.document.TargetFingerprint
}

func (p ProviderStatefulProfile) Method(
	sourceIdentity string,
) (ProviderStatefulProfileMethod, bool) {
	index, found := slices.BinarySearchFunc(
		p.profile.Methods,
		sourceIdentity,
		func(method ProviderStatefulProfileMethodDocument, identity string) int {
			return strings.Compare(method.SourceIdentity, identity)
		},
	)
	if !found {
		return ProviderStatefulProfileMethod{}, false
	}
	return ProviderStatefulProfileMethod{document: p.profile.Methods[index]}, true
}

type ProviderStatefulProfileMethod struct {
	document ProviderStatefulProfileMethodDocument
}

func (m ProviderStatefulProfileMethod) SourceIdentity() string {
	return m.document.SourceIdentity
}

func (m ProviderStatefulProfileMethod) Member() string {
	return m.document.Member
}

func (m ProviderStatefulProfileMethod) Effect() EffectKind {
	return m.document.Effect
}

func (m ProviderStatefulProfileMethod) SourceSignature() string {
	return m.document.SourceSignature
}

func (m ProviderStatefulProfileMethod) SourceLocation() string {
	return m.document.SourceLocation
}

func (m ProviderStatefulProfileMethod) ImplementationOwner() string {
	return m.document.ImplementationOwner
}

func (m ProviderStatefulProfileMethod) InstanceTargetFingerprint() string {
	return m.document.InstanceTargetFingerprint
}

func (m ProviderStatefulProfileMethod) StaticTargetFingerprint() string {
	return m.document.StaticTargetFingerprint
}

func (p ProviderStatefulProfile) ImplementationOwner() string {
	return p.profile.ImplementationOwner
}

func (p ProviderStatefulProfile) TargetFingerprint() string {
	return p.profile.TargetFingerprint
}

func cloneProviderStatefulProfile(
	source ProviderStatefulProfileDocument,
) ProviderStatefulProfileDocument {
	result := source
	result.Interfaces = make(
		[]ProviderCallableProfileInterfaceDocument,
		len(source.Interfaces),
	)
	for index, selected := range source.Interfaces {
		result.Interfaces[index] = cloneProviderCallableProfileInterface(selected)
	}
	result.TypeArguments = slices.Clone(source.TypeArguments)
	result.Operations = slices.Clone(source.Operations)
	result.Fields = slices.Clone(source.Fields)
	result.Methods = slices.Clone(source.Methods)
	return result
}
