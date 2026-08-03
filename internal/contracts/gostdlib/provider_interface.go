package gostdlib

import (
	"slices"
)

type ProviderInterfaceMode string

const (
	ProviderInterfaceModeInvalid      ProviderInterfaceMode = ""
	ProviderInterfaceModeBridge       ProviderInterfaceMode = "bridge"
	ProviderInterfaceModeSealedNative ProviderInterfaceMode = "sealed-native"
)

func (m ProviderInterfaceMode) Valid() bool {
	return m == ProviderInterfaceModeBridge ||
		m == ProviderInterfaceModeSealedNative
}

type ProviderInterfaceMethodKind string

const (
	ProviderInterfaceMethodInvalid     ProviderInterfaceMethodKind = ""
	ProviderInterfaceMethodCallable    ProviderInterfaceMethodKind = "callable"
	ProviderInterfaceMethodRuntimeOnly ProviderInterfaceMethodKind = "runtime-only"
)

func (k ProviderInterfaceMethodKind) Valid() bool {
	return k == ProviderInterfaceMethodCallable ||
		k == ProviderInterfaceMethodRuntimeOnly
}

type ProviderInterfaceDocument struct {
	Mode    ProviderInterfaceMode             `json:"mode"`
	Methods []ProviderInterfaceMethodDocument `json:"methods"`
}

const (
	LanguageErrorInterfaceIdentity = "go:universe|error"
	LanguageErrorMethodIdentity    = "go:universe|error|method=Error"
)

type ProviderInterfaceBindingDocument struct {
	SourceIdentity    string                    `json:"sourceIdentity"`
	Export            string                    `json:"export"`
	ProviderInterface ProviderInterfaceDocument `json:"providerInterface"`
	TargetFingerprint string                    `json:"targetFingerprint"`
}

type ProviderInterfaceBinding struct {
	module   FacetModuleDocument
	document ProviderInterfaceBindingDocument
}

func newProviderInterfaceBinding(
	module FacetModuleDocument,
	document ProviderInterfaceBindingDocument,
) ProviderInterfaceBinding {
	return ProviderInterfaceBinding{
		module:   facetModuleIdentity(module),
		document: cloneProviderInterfaceBinding(document),
	}
}

func (b ProviderInterfaceBinding) SourceIdentity() string {
	return b.document.SourceIdentity
}

func (b ProviderInterfaceBinding) ModuleSpecifier() string {
	return b.module.Specifier
}

func (b ProviderInterfaceBinding) Export() string {
	return b.document.Export
}

func (b ProviderInterfaceBinding) ProviderInterface() ProviderInterface {
	return newProviderInterface(b.document.ProviderInterface)
}

func (b ProviderInterfaceBinding) TargetFingerprint() string {
	return b.document.TargetFingerprint
}

type ProviderInterfaceMethodDocument struct {
	SourceIdentity      string                      `json:"sourceIdentity"`
	Kind                ProviderInterfaceMethodKind `json:"kind"`
	Member              string                      `json:"member,omitempty"`
	Effect              EffectKind                  `json:"effect,omitempty"`
	SourceSignature     string                      `json:"sourceSignature"`
	SourceLocation      string                      `json:"sourceLocation"`
	ImplementationOwner string                      `json:"implementationOwner,omitempty"`
	TargetFingerprint   string                      `json:"targetFingerprint,omitempty"`
}

type ProviderInterface struct {
	document ProviderInterfaceDocument
}

func newProviderInterface(document ProviderInterfaceDocument) ProviderInterface {
	return ProviderInterface{document: cloneProviderInterface(document)}
}

func (i ProviderInterface) Methods() []ProviderInterfaceMethod {
	result := make([]ProviderInterfaceMethod, len(i.document.Methods))
	for index, method := range i.document.Methods {
		result[index] = ProviderInterfaceMethod{document: method}
	}
	return result
}

func (i ProviderInterface) Mode() ProviderInterfaceMode {
	return i.document.Mode
}

func (i ProviderInterface) Method(
	sourceIdentity string,
) (ProviderInterfaceMethod, bool) {
	index, found := slices.BinarySearchFunc(
		i.document.Methods,
		sourceIdentity,
		func(method ProviderInterfaceMethodDocument, identity string) int {
			switch {
			case method.SourceIdentity < identity:
				return -1
			case method.SourceIdentity > identity:
				return 1
			default:
				return 0
			}
		},
	)
	if !found {
		return ProviderInterfaceMethod{}, false
	}
	return ProviderInterfaceMethod{document: i.document.Methods[index]}, true
}

type ProviderInterfaceMethod struct {
	document ProviderInterfaceMethodDocument
}

func (m ProviderInterfaceMethod) SourceIdentity() string {
	return m.document.SourceIdentity
}

func (m ProviderInterfaceMethod) Kind() ProviderInterfaceMethodKind {
	return m.document.Kind
}

func (m ProviderInterfaceMethod) Member() string {
	return m.document.Member
}

func (m ProviderInterfaceMethod) Effect() EffectKind {
	return m.document.Effect
}

func (m ProviderInterfaceMethod) SourceSignature() string {
	return m.document.SourceSignature
}

func (m ProviderInterfaceMethod) SourceLocation() string {
	return m.document.SourceLocation
}

func (m ProviderInterfaceMethod) ImplementationOwner() string {
	return m.document.ImplementationOwner
}

func (m ProviderInterfaceMethod) TargetFingerprint() string {
	return m.document.TargetFingerprint
}

func cloneProviderInterface(
	source ProviderInterfaceDocument,
) ProviderInterfaceDocument {
	result := source
	result.Methods = slices.Clone(source.Methods)
	return result
}

func cloneProviderInterfaceBinding(
	source ProviderInterfaceBindingDocument,
) ProviderInterfaceBindingDocument {
	result := source
	result.ProviderInterface = cloneProviderInterface(source.ProviderInterface)
	return result
}
