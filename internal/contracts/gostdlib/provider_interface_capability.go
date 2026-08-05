package gostdlib

type ProviderInterfaceCapabilityUsage string

const (
	ProviderInterfaceCapabilityUsageInvalid          ProviderInterfaceCapabilityUsage = ""
	ProviderInterfaceCapabilityUsageProviderInternal ProviderInterfaceCapabilityUsage = "provider-internal"
	ProviderInterfaceCapabilityUsageGeneratedBridge  ProviderInterfaceCapabilityUsage = "generated-bridge"
)

func (u ProviderInterfaceCapabilityUsage) Valid() bool {
	return u == ProviderInterfaceCapabilityUsageProviderInternal ||
		u == ProviderInterfaceCapabilityUsageGeneratedBridge
}

type ProviderInterfaceCapabilityDocument struct {
	Usage                 ProviderInterfaceCapabilityUsage `json:"usage"`
	BaseSourceIdentity    string                           `json:"baseSourceIdentity"`
	BaseExport            string                           `json:"baseExport"`
	ProfileSourceIdentity string                           `json:"profileSourceIdentity"`
	ProfileKey            string                           `json:"profileKey"`
	TargetSourceIdentity  string                           `json:"targetSourceIdentity"`
	TargetExport          string                           `json:"targetExport"`
	ViewExport            string                           `json:"viewExport"`
	ImplementationOwner   string                           `json:"implementationOwner"`
	ViewFingerprint       string                           `json:"viewFingerprint"`
}

type ProviderInterfaceCapability struct {
	module   FacetModuleDocument
	document ProviderInterfaceCapabilityDocument
	profile  ProviderCallableProfileDocument
	base     ProviderCallableProfileInterfaceDocument
	target   ProviderCallableProfileInterfaceDocument
}

func newProviderInterfaceCapability(
	module FacetModuleDocument,
	document ProviderInterfaceCapabilityDocument,
	profile ProviderCallableProfileDocument,
	base ProviderCallableProfileInterfaceDocument,
	target ProviderCallableProfileInterfaceDocument,
) ProviderInterfaceCapability {
	return ProviderInterfaceCapability{
		module:   facetModuleIdentity(module),
		document: document,
		profile:  cloneProviderCallableProfile(profile),
		base:     cloneProviderCallableProfileInterface(base),
		target:   cloneProviderCallableProfileInterface(target),
	}
}

func resolveProviderInterfaceCapability(
	module FacetModuleDocument,
	document ProviderInterfaceCapabilityDocument,
) ProviderInterfaceCapability {
	var profile ProviderCallableProfileDocument
	for _, candidate := range module.CallableProfiles {
		if candidate.SourceIdentity == document.ProfileSourceIdentity &&
			candidate.ProfileKey == document.ProfileKey {
			profile = candidate
			break
		}
	}
	var base ProviderCallableProfileInterfaceDocument
	var target ProviderCallableProfileInterfaceDocument
	for _, candidate := range profile.Interfaces {
		if candidate.SourceIdentity == document.BaseSourceIdentity &&
			candidate.Export == document.BaseExport {
			base = candidate
		}
		if candidate.SourceIdentity == document.TargetSourceIdentity &&
			candidate.Export == document.TargetExport {
			target = candidate
		}
	}
	if document.Usage == ProviderInterfaceCapabilityUsageProviderInternal ||
		base.Export == "" {
		for _, candidate := range module.ProviderInterfaces {
			if candidate.SourceIdentity == document.BaseSourceIdentity &&
				candidate.Export == document.BaseExport {
				base = providerBindingProfileInterface(candidate)
				break
			}
		}
	}
	return newProviderInterfaceCapability(module, document, profile, base, target)
}

func (c ProviderInterfaceCapability) Valid() bool {
	return c.module.Specifier != "" &&
		c.document.Usage.Valid() &&
		c.document.BaseSourceIdentity != "" &&
		c.document.BaseExport != "" &&
		c.document.ProfileSourceIdentity != "" &&
		c.document.ProfileKey != "" &&
		c.document.TargetSourceIdentity != "" &&
		c.document.TargetExport != "" &&
		c.document.ViewExport != "" &&
		c.profile.SourceIdentity == c.document.ProfileSourceIdentity &&
		c.profile.ProfileKey == c.document.ProfileKey &&
		c.base.SourceIdentity == c.document.BaseSourceIdentity &&
		c.base.Export == c.document.BaseExport &&
		c.target.SourceIdentity == c.document.TargetSourceIdentity &&
		c.target.Export == c.document.TargetExport
}

func (c ProviderInterfaceCapability) Usage() ProviderInterfaceCapabilityUsage {
	return c.document.Usage
}

func (c ProviderInterfaceCapability) BaseInterface() ProviderCallableProfileInterface {
	return ProviderCallableProfileInterface{
		document: cloneProviderCallableProfileInterface(c.base),
	}
}

func (c ProviderInterfaceCapability) ProfileInterfaces() []ProviderCallableProfileInterface {
	result := make([]ProviderCallableProfileInterface, len(c.profile.Interfaces))
	for index, selected := range c.profile.Interfaces {
		result[index] = ProviderCallableProfileInterface{
			document: cloneProviderCallableProfileInterface(selected),
		}
	}
	return result
}

func (c ProviderInterfaceCapability) BaseSourceIdentity() string {
	return c.document.BaseSourceIdentity
}

func (c ProviderInterfaceCapability) BaseExport() string {
	return c.document.BaseExport
}

func (c ProviderInterfaceCapability) ProfileSourceIdentity() string {
	return c.document.ProfileSourceIdentity
}

func (c ProviderInterfaceCapability) ProfileKey() string {
	return c.document.ProfileKey
}

func (c ProviderInterfaceCapability) TargetSourceIdentity() string {
	return c.document.TargetSourceIdentity
}

func (c ProviderInterfaceCapability) TargetExport() string {
	return c.document.TargetExport
}

func (c ProviderInterfaceCapability) ViewExport() string {
	return c.document.ViewExport
}

func (c ProviderInterfaceCapability) ModuleSpecifier() string {
	return c.module.Specifier
}

func (c ProviderInterfaceCapability) ImplementationOwner() string {
	return c.document.ImplementationOwner
}

func (c ProviderInterfaceCapability) ViewFingerprint() string {
	return c.document.ViewFingerprint
}

func (c ProviderInterfaceCapability) TargetInterface() ProviderCallableProfileInterface {
	return ProviderCallableProfileInterface{
		document: cloneProviderCallableProfileInterface(c.target),
	}
}
