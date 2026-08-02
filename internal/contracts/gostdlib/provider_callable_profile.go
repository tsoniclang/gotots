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
	SourceIdentity              string                                     `json:"sourceIdentity"`
	ProfileKey                  string                                     `json:"profileKey"`
	Export                      string                                     `json:"export"`
	Required                    bool                                       `json:"required,omitempty"`
	Receiver                    bool                                       `json:"receiver,omitempty"`
	CanonicalParameters         []int                                      `json:"canonicalParameters"`
	CanonicalResults            []int                                      `json:"canonicalResults"`
	CanonicalValues             []string                                   `json:"canonicalValues,omitempty"`
	CanonicalTypeArguments      []string                                   `json:"canonicalTypeArguments,omitempty"`
	GuardInterfaces             []string                                   `json:"guardInterfaces,omitempty"`
	ContractInterfaces          []string                                   `json:"contractInterfaces,omitempty"`
	FromProviderInterfaces      []string                                   `json:"fromProviderInterfaces,omitempty"`
	ImplementedResultInterfaces []string                                   `json:"implementedResultInterfaces,omitempty"`
	Interfaces                  []ProviderCallableProfileInterfaceDocument `json:"interfaces"`
	Effect                      EffectKind                                 `json:"effect"`
	ImplementationOwner         string                                     `json:"implementationOwner"`
	TargetFingerprint           string                                     `json:"targetFingerprint"`
}

type ProviderCallableProfileInterfaceDocument struct {
	SourceIdentity         string                             `json:"sourceIdentity"`
	Export                 string                             `json:"export"`
	Protocol               *ProviderProtocolInterfaceDocument `json:"protocol,omitempty"`
	ProtocolValueParameter *int                               `json:"protocolValueParameter,omitempty"`
	ProviderInterface      ProviderInterfaceDocument          `json:"providerInterface"`
	TargetFingerprint      string                             `json:"targetFingerprint"`
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
	return ProviderCallableProfile{
		module:  facetModuleIdentity(module),
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

func (p ProviderCallableProfile) Required() bool {
	return p.profile.Required
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

func (p ProviderCallableProfile) CanonicalValues() []string {
	return slices.Clone(p.profile.CanonicalValues)
}

func (p ProviderCallableProfile) CanonicalTypeArguments() []string {
	return slices.Clone(p.profile.CanonicalTypeArguments)
}

func (p ProviderCallableProfile) GuardInterfaces() []string {
	return slices.Clone(p.profile.GuardInterfaces)
}

func (p ProviderCallableProfile) ContractInterfaces() []string {
	return slices.Clone(p.profile.ContractInterfaces)
}

func (p ProviderCallableProfile) FromProviderInterfaces() []string {
	return slices.Clone(p.profile.FromProviderInterfaces)
}

func (p ProviderCallableProfile) ImplementedResultInterfaces() []string {
	return slices.Clone(p.profile.ImplementedResultInterfaces)
}

func (p ProviderCallableProfile) Interfaces() []ProviderCallableProfileInterface {
	result := make([]ProviderCallableProfileInterface, 0, len(p.profile.Interfaces))
	for _, selected := range p.profile.Interfaces {
		result = append(result, ProviderCallableProfileInterface{
			document: cloneProviderCallableProfileInterface(selected),
		})
	}
	return result
}

func (p ProviderCallableProfile) Interface(
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

func (i ProviderCallableProfileInterface) Protocol() (
	ProviderProtocolInterfaceDocument,
	bool,
) {
	if i.document.Protocol == nil {
		return ProviderProtocolInterfaceDocument{}, false
	}
	return cloneProviderProtocolInterface(*i.document.Protocol), true
}

func (i ProviderCallableProfileInterface) ProtocolValueParameter() (int, bool) {
	if i.document.ProtocolValueParameter == nil {
		return 0, false
	}
	return *i.document.ProtocolValueParameter, true
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

func BuildImplementedResultProfileKey(
	source []ProviderCallableProfileKeyInterface,
	implementedResultInterfaces []string,
) (string, error) {
	base, err := BuildProviderCallableProfileKey(source)
	if err != nil || len(implementedResultInterfaces) == 0 {
		return base, err
	}
	if !sort.StringsAreSorted(implementedResultInterfaces) {
		return "", &ManifestError{
			Field:  "providerCallableProfileKey.implementedResultInterfaces",
			Reason: "identities are not strictly ordered",
		}
	}
	var payload strings.Builder
	payload.WriteString("provider-callable-profile-implemented-results/v1\n")
	payload.WriteString("base=")
	payload.WriteString(base)
	payload.WriteByte('\n')
	previous := ""
	for _, identity := range implementedResultInterfaces {
		if identity == "" || identity == previous {
			return "", &ManifestError{
				Field:  "providerCallableProfileKey.implementedResultInterfaces",
				Reason: "identities are empty or duplicated",
			}
		}
		previous = identity
		payload.WriteString("interface=")
		payload.WriteString(identity)
		payload.WriteByte('\n')
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
	result.CanonicalValues = slices.Clone(source.CanonicalValues)
	result.CanonicalTypeArguments = slices.Clone(source.CanonicalTypeArguments)
	result.GuardInterfaces = slices.Clone(source.GuardInterfaces)
	result.ContractInterfaces = slices.Clone(source.ContractInterfaces)
	result.FromProviderInterfaces = slices.Clone(source.FromProviderInterfaces)
	result.ImplementedResultInterfaces = slices.Clone(
		source.ImplementedResultInterfaces,
	)
	result.Interfaces = make(
		[]ProviderCallableProfileInterfaceDocument,
		len(source.Interfaces),
	)
	for index, selected := range source.Interfaces {
		result.Interfaces[index] = cloneProviderCallableProfileInterface(selected)
	}
	return result
}

func cloneProviderCallableProfileInterface(
	source ProviderCallableProfileInterfaceDocument,
) ProviderCallableProfileInterfaceDocument {
	result := source
	if source.Protocol != nil {
		protocol := cloneProviderProtocolInterface(*source.Protocol)
		result.Protocol = &protocol
	}
	if source.ProtocolValueParameter != nil {
		parameter := *source.ProtocolValueParameter
		result.ProtocolValueParameter = &parameter
	}
	result.ProviderInterface = cloneProviderInterface(source.ProviderInterface)
	return result
}
