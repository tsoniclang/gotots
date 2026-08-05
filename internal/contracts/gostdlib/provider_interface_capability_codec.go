package gostdlib

import "fmt"

func validateProviderInterfaceCapabilities(
	capabilities []ProviderInterfaceCapabilityDocument,
	moduleField string,
	callableProfiles map[string]ProviderCallableProfileDocument,
	providerInterfaces map[string]ProviderInterfaceBindingDocument,
	owners map[string]struct{},
) error {
	previous := ""
	for index, capability := range capabilities {
		field := fmt.Sprintf(
			"%s.providerInterfaceCapabilities[%d]",
			moduleField,
			index,
		)
		key := capability.BaseSourceIdentity + "\x00" +
			capability.TargetSourceIdentity + "\x00" + capability.TargetExport
		switch {
		case !capability.Usage.Valid():
			return manifestError(field+".usage", "value is invalid")
		case capability.BaseSourceIdentity == "":
			return manifestError(field+".baseSourceIdentity", "value is empty")
		case capability.BaseExport == "":
			return manifestError(field+".baseExport", "value is empty")
		case capability.ProfileSourceIdentity == "":
			return manifestError(field+".profileSourceIdentity", "value is empty")
		case capability.ProfileKey == "":
			return manifestError(field+".profileKey", "value is empty")
		case capability.TargetSourceIdentity == "":
			return manifestError(field+".targetSourceIdentity", "value is empty")
		case capability.TargetExport == "":
			return manifestError(field+".targetExport", "value is empty")
		case capability.ViewExport == "":
			return manifestError(field+".viewExport", "value is empty")
		case !sourcePath(capability.ImplementationOwner):
			return manifestError(
				field+".implementationOwner",
				"value is not a provider source path",
			)
		case !validDigest(capability.ViewFingerprint):
			return manifestError(
				field+".viewFingerprint",
				"value is not a sha256 digest",
			)
		case previous != "" && key <= previous:
			return manifestError(
				moduleField+".providerInterfaceCapabilities",
				"values are not strictly ordered",
			)
		}
		previous = key
		profile, ok := callableProfiles[capability.ProfileSourceIdentity+"\x00"+capability.ProfileKey]
		if !ok {
			return manifestError(
				field+".profileKey",
				"callable profile does not exact-join this module",
			)
		}
		matchedProviderBase := providerInterfaces[capability.BaseSourceIdentity+"\x00"+
			capability.BaseExport].Export != ""
		matchedBase := false
		matchedTarget := false
		for _, target := range profile.Interfaces {
			if target.SourceIdentity == capability.BaseSourceIdentity &&
				target.Export == capability.BaseExport {
				matchedBase = true
			}
			if target.SourceIdentity == capability.TargetSourceIdentity &&
				target.Export == capability.TargetExport {
				matchedTarget = true
			}
		}
		switch capability.Usage {
		case ProviderInterfaceCapabilityUsageProviderInternal:
			matchedBase = matchedProviderBase
		case ProviderInterfaceCapabilityUsageGeneratedBridge:
			matchedBase = matchedBase != matchedProviderBase
		}
		if !matchedBase || !matchedTarget {
			return manifestError(
				field+".targetSourceIdentity",
				"base or target interface does not exact-join the callable profile",
			)
		}
		if _, duplicate := owners[capability.ViewExport]; duplicate {
			return manifestError(field+".viewExport", "target owner is duplicated")
		}
		owners[capability.ViewExport] = struct{}{}
	}
	return nil
}
