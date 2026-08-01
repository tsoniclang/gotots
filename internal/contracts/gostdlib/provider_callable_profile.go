package gostdlib

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"sort"
	"strings"
)

type providerCallableProfileLookup struct {
	sourceIdentity string
	profileKey     string
}

type ProviderCallableProfileDocument struct {
	SourceIdentity      string     `json:"sourceIdentity"`
	ProfileKey          string     `json:"profileKey"`
	Export              string     `json:"export"`
	Receiver            bool       `json:"receiver,omitempty"`
	CanonicalParameters []int      `json:"canonicalParameters"`
	CanonicalResults    []int      `json:"canonicalResults"`
	GuardInterfaces     []string   `json:"guardInterfaces,omitempty"`
	Interfaces          []string   `json:"interfaces"`
	Effect              EffectKind `json:"effect"`
	ImplementationOwner string     `json:"implementationOwner"`
	TargetFingerprint   string     `json:"targetFingerprint"`
}

type ProviderCallableProfileInterfaceDocument struct {
	SourceIdentity    string                    `json:"sourceIdentity"`
	Export            string                    `json:"export"`
	ProviderInterface ProviderInterfaceDocument `json:"providerInterface"`
	TargetFingerprint string                    `json:"targetFingerprint"`
}

type ProviderCallableProfile struct {
	module  FacetModuleDocument
	profile ProviderCallableProfileDocument
}

func (p ProviderCallableProfile) Valid() bool {
	return p.profile.SourceIdentity != "" &&
		p.profile.ProfileKey != "" &&
		p.module.Specifier != "" &&
		p.profile.Export != "" &&
		len(p.profile.CanonicalParameters) != 0 &&
		len(p.profile.Interfaces) != 0 &&
		p.profile.Effect.Valid() &&
		p.profile.ImplementationOwner != "" &&
		p.profile.TargetFingerprint != ""
}

func newProviderCallableProfile(
	module FacetModuleDocument,
	profile ProviderCallableProfileDocument,
) ProviderCallableProfile {
	module.Facets = nil
	module.Representations = nil
	module.CallableProfiles = nil
	return ProviderCallableProfile{
		module:  module,
		profile: cloneProviderCallableProfile(profile),
	}
}

func (p ProviderCallableProfile) SourceIdentity() string {
	return p.profile.SourceIdentity
}

func (p ProviderCallableProfile) ProfileKey() string {
	return p.profile.ProfileKey
}

func (p ProviderCallableProfile) ModuleSpecifier() string {
	return p.module.Specifier
}

func (p ProviderCallableProfile) Export() string {
	return p.profile.Export
}

func (p ProviderCallableProfile) Receiver() bool {
	return p.profile.Receiver
}

func (p ProviderCallableProfile) CanonicalParameters() []int {
	return slices.Clone(p.profile.CanonicalParameters)
}

func (p ProviderCallableProfile) CanonicalResults() []int {
	return slices.Clone(p.profile.CanonicalResults)
}

func (p ProviderCallableProfile) GuardInterfaces() []string {
	return slices.Clone(p.profile.GuardInterfaces)
}

func (p ProviderCallableProfile) Interfaces() []ProviderCallableProfileInterface {
	result := make([]ProviderCallableProfileInterface, 0, len(p.profile.Interfaces))
	for _, identity := range p.profile.Interfaces {
		selected, ok := p.interfaceDocument(identity)
		if !ok {
			return nil
		}
		result = append(result, ProviderCallableProfileInterface{
			document: cloneProviderCallableProfileInterface(selected),
		})
	}
	return result
}

func (p ProviderCallableProfile) Interface(
	sourceIdentity string,
) (ProviderCallableProfileInterface, bool) {
	if _, found := slices.BinarySearch(p.profile.Interfaces, sourceIdentity); !found {
		return ProviderCallableProfileInterface{}, false
	}
	selected, found := p.interfaceDocument(sourceIdentity)
	if !found {
		return ProviderCallableProfileInterface{}, false
	}
	return ProviderCallableProfileInterface{
		document: cloneProviderCallableProfileInterface(selected),
	}, true
}

func (p ProviderCallableProfile) interfaceDocument(
	sourceIdentity string,
) (ProviderCallableProfileInterfaceDocument, bool) {
	index, found := slices.BinarySearchFunc(
		p.module.CallableInterfaces,
		sourceIdentity,
		func(
			selected ProviderCallableProfileInterfaceDocument,
			identity string,
		) int {
			return strings.Compare(selected.SourceIdentity, identity)
		},
	)
	if !found {
		return ProviderCallableProfileInterfaceDocument{}, false
	}
	return p.module.CallableInterfaces[index], true
}

func (p ProviderCallableProfile) Effect() EffectKind {
	return p.profile.Effect
}

func (p ProviderCallableProfile) ImplementationOwner() string {
	return p.profile.ImplementationOwner
}

func (p ProviderCallableProfile) TargetFingerprint() string {
	return p.profile.TargetFingerprint
}

type ProviderCallableProfileInterface struct {
	document ProviderCallableProfileInterfaceDocument
}

func (i ProviderCallableProfileInterface) SourceIdentity() string {
	return i.document.SourceIdentity
}

func (i ProviderCallableProfileInterface) Export() string {
	return i.document.Export
}

func (i ProviderCallableProfileInterface) ProviderInterface() ProviderInterface {
	return newProviderInterface(i.document.ProviderInterface)
}

func (i ProviderCallableProfileInterface) TargetFingerprint() string {
	return i.document.TargetFingerprint
}

type ProviderCallableProfileKeyInterface struct {
	SourceIdentity string
	Methods        []ProviderCallableProfileKeyMethod
}

type ProviderCallableProfileKeyMethod struct {
	SourceIdentity string
	Effect         EffectKind
}

func BuildProviderCallableProfileKey(
	source []ProviderCallableProfileKeyInterface,
) (string, error) {
	interfaces := make([]ProviderCallableProfileKeyInterface, len(source))
	for index, selected := range source {
		interfaces[index] = ProviderCallableProfileKeyInterface{
			SourceIdentity: selected.SourceIdentity,
			Methods:        slices.Clone(selected.Methods),
		}
		sort.Slice(interfaces[index].Methods, func(left, right int) bool {
			return interfaces[index].Methods[left].SourceIdentity <
				interfaces[index].Methods[right].SourceIdentity
		})
	}
	sort.Slice(interfaces, func(left, right int) bool {
		return interfaces[left].SourceIdentity < interfaces[right].SourceIdentity
	})
	var payload strings.Builder
	payload.WriteString("provider-callable-profile/v1\n")
	previousInterface := ""
	for _, selected := range interfaces {
		if selected.SourceIdentity == "" ||
			selected.SourceIdentity <= previousInterface ||
			len(selected.Methods) == 0 {
			return "", &ManifestError{
				Field:  "providerCallableProfileKey.interfaces",
				Reason: "interfaces are empty, duplicated, or incomplete",
			}
		}
		previousInterface = selected.SourceIdentity
		payload.WriteString("interface=")
		payload.WriteString(selected.SourceIdentity)
		payload.WriteByte('\n')
		previousMethod := ""
		for _, method := range selected.Methods {
			if method.SourceIdentity == "" ||
				method.SourceIdentity <= previousMethod ||
				!method.Effect.Valid() {
				return "", &ManifestError{
					Field:  "providerCallableProfileKey.methods",
					Reason: "methods are empty, duplicated, unordered, or invalid",
				}
			}
			previousMethod = method.SourceIdentity
			payload.WriteString("method=")
			payload.WriteString(method.SourceIdentity)
			payload.WriteString("|effect=")
			payload.WriteString(string(method.Effect))
			payload.WriteByte('\n')
		}
	}
	digest := sha256.Sum256([]byte(payload.String()))
	return hex.EncodeToString(digest[:]), nil
}

func cloneProviderCallableProfile(
	source ProviderCallableProfileDocument,
) ProviderCallableProfileDocument {
	result := source
	result.CanonicalParameters = slices.Clone(source.CanonicalParameters)
	result.CanonicalResults = slices.Clone(source.CanonicalResults)
	result.GuardInterfaces = slices.Clone(source.GuardInterfaces)
	result.Interfaces = slices.Clone(source.Interfaces)
	return result
}

func cloneProviderCallableProfileInterface(
	source ProviderCallableProfileInterfaceDocument,
) ProviderCallableProfileInterfaceDocument {
	result := source
	result.ProviderInterface = cloneProviderInterface(source.ProviderInterface)
	return result
}
