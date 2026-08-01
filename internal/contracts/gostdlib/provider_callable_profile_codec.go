package gostdlib

import "fmt"

func validateProviderCallableProfile(
	profile ProviderCallableProfileDocument,
	field string,
	interfaces map[string]ProviderCallableProfileInterfaceDocument,
) error {
	switch {
	case profile.SourceIdentity == "":
		return manifestError(field+".sourceIdentity", "value is empty")
	case !validDigest(profile.ProfileKey):
		return manifestError(field+".profileKey", "value is not a sha256 digest")
	case profile.Export == "":
		return manifestError(field+".export", "value is empty")
	case len(profile.CanonicalParameters) == 0:
		return manifestError(field+".canonicalParameters", "set is empty")
	case len(profile.Interfaces) == 0:
		return manifestError(field+".interfaces", "set is empty")
	case !profile.Effect.Valid():
		return manifestError(field+".effect", "value is invalid")
	case !sourcePath(profile.ImplementationOwner):
		return manifestError(
			field+".implementationOwner",
			"value is not a provider source path",
		)
	case !validDigest(profile.TargetFingerprint):
		return manifestError(
			field+".targetFingerprint",
			"value is not a sha256 digest",
		)
	}
	if err := validateProfileIndexes(
		profile.CanonicalParameters,
		field+".canonicalParameters",
	); err != nil {
		return err
	}
	if err := validateProfileIndexes(
		profile.CanonicalResults,
		field+".canonicalResults",
	); err != nil {
		return err
	}
	keyInterfaces := make(
		[]ProviderCallableProfileKeyInterface,
		0,
		len(profile.Interfaces),
	)
	interfaceIdentities := make(map[string]struct{}, len(profile.Interfaces))
	previous := ""
	for index, identity := range profile.Interfaces {
		selectedField := fmt.Sprintf("%s.interfaces[%d]", field, index)
		if identity == "" {
			return manifestError(selectedField, "value is empty")
		}
		if identity <= previous {
			return manifestError(field+".interfaces", "values are not strictly ordered")
		}
		previous = identity
		selected, ok := interfaces[identity]
		if !ok {
			return manifestError(selectedField, "callable-interface definition is absent")
		}
		interfaceIdentities[identity] = struct{}{}
		methods := make(
			[]ProviderCallableProfileKeyMethod,
			0,
			len(selected.ProviderInterface.Methods),
		)
		for _, method := range selected.ProviderInterface.Methods {
			if method.Kind != ProviderInterfaceMethodCallable {
				return manifestError(
					selectedField+".providerInterface.methods",
					"profile interface contains a runtime-only method",
				)
			}
			methods = append(methods, ProviderCallableProfileKeyMethod{
				SourceIdentity: method.SourceIdentity,
				Effect:         method.Effect,
			})
		}
		keyInterfaces = append(keyInterfaces, ProviderCallableProfileKeyInterface{
			SourceIdentity: identity,
			Methods:        methods,
		})
	}
	key, err := BuildProviderCallableProfileKey(keyInterfaces)
	if err != nil {
		return err
	}
	if key != profile.ProfileKey {
		return manifestError(field+".profileKey", "value does not match interface evidence")
	}
	seenGuards := make(map[string]struct{}, len(profile.GuardInterfaces))
	for index, identity := range profile.GuardInterfaces {
		guardField := fmt.Sprintf("%s.guardInterfaces[%d]", field, index)
		if _, ok := interfaceIdentities[identity]; !ok {
			return manifestError(guardField, "value has no profile-interface evidence")
		}
		if _, duplicate := seenGuards[identity]; duplicate {
			return manifestError(guardField, "value is duplicated")
		}
		seenGuards[identity] = struct{}{}
	}
	return nil
}

func validateProviderCallableProfileInterface(
	selected ProviderCallableProfileInterfaceDocument,
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
	case selected.ProviderInterface.Mode != ProviderInterfaceModeBridge:
		return manifestError(
			field+".providerInterface.mode",
			"profile interface must expose the complete callable method set",
		)
	}
	if err := validateProviderInterface(
		selected.ProviderInterface,
		field+".providerInterface",
	); err != nil {
		return err
	}
	for _, method := range selected.ProviderInterface.Methods {
		if method.Kind != ProviderInterfaceMethodCallable {
			return manifestError(
				field+".providerInterface.methods",
				"profile interface contains a runtime-only method",
			)
		}
	}
	return nil
}

func validateProfileIndexes(source []int, field string) error {
	previous := -1
	for _, selected := range source {
		if selected < 0 || selected <= previous {
			return manifestError(
				field,
				"values are negative, duplicated, or not strictly ordered",
			)
		}
		previous = selected
	}
	return nil
}
