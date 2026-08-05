package gostdlib

import "fmt"

func validateProviderInterfaceBinding(
	selected ProviderInterfaceBindingDocument,
	field string,
) error {
	switch {
	case selected.SourceIdentity == "":
		return manifestError(field+".sourceIdentity", "value is empty")
	case selected.Export == "":
		return manifestError(field+".export", "value is empty")
	case !validDigest(selected.TargetFingerprint):
		return manifestError(
			field+".targetFingerprint",
			"value is not a sha256 digest",
		)
	}
	return validateProviderInterface(
		selected.ProviderInterface,
		field+".providerInterface",
	)
}

func validateProviderInterface(
	providerInterface ProviderInterfaceDocument,
	field string,
) error {
	if !providerInterface.Mode.Valid() {
		return manifestError(field+".mode", "value is invalid")
	}
	if len(providerInterface.Methods) == 0 {
		return manifestError(field+".methods", "set is empty")
	}
	runtimeOnly := 0
	for index, method := range providerInterface.Methods {
		methodField := fmt.Sprintf("%s.methods[%d]", field, index)
		switch {
		case method.SourceIdentity == "":
			return manifestError(methodField+".sourceIdentity", "value is empty")
		case index != 0 && method.SourceIdentity <=
			providerInterface.Methods[index-1].SourceIdentity:
			return manifestError(field+".methods", "methods are not strictly ordered")
		case !method.Kind.Valid():
			return manifestError(methodField+".kind", "value is invalid")
		case method.SourceSignature == "":
			return manifestError(methodField+".sourceSignature", "value is empty")
		case method.ContractSignature == "":
			return manifestError(methodField+".contractSignature", "value is empty")
		case method.SourceLocation == "":
			return manifestError(methodField+".sourceLocation", "value is empty")
		}
		switch method.Kind {
		case ProviderInterfaceMethodCallable:
			if method.Member == "" || !method.Effect.Valid() ||
				!sourcePath(method.ImplementationOwner) ||
				!validDigest(method.TargetFingerprint) {
				return manifestError(
					methodField,
					"callable method evidence is incomplete",
				)
			}
		case ProviderInterfaceMethodRuntimeOnly:
			runtimeOnly++
			if method.Member != "" || method.Effect != EffectInvalid ||
				method.ImplementationOwner != "" ||
				method.TargetFingerprint != "" {
				return manifestError(
					methodField,
					"runtime-only method carries a public target",
				)
			}
		}
	}
	if providerInterface.Mode == ProviderInterfaceModeBridge && runtimeOnly != 0 {
		return manifestError(
			field+".mode",
			"bridge interface has runtime-only methods",
		)
	}
	if providerInterface.Mode == ProviderInterfaceModeSealedNative && runtimeOnly == 0 {
		return manifestError(
			field+".mode",
			"sealed-native interface has no sealing method",
		)
	}
	return nil
}
