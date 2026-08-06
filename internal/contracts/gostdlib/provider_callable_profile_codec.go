package gostdlib

import (
	"fmt"
	"slices"
)

func validateProviderCallableProfile(
	profile ProviderCallableProfileDocument,
	field string,
) error {
	switch {
	case profile.SourceIdentity == "":
		return manifestError(field+".sourceIdentity", "value is empty")
	case !validDigest(profile.ProfileKey):
		return manifestError(field+".profileKey", "value is not a sha256 digest")
	case profile.Export == "":
		return manifestError(field+".export", "value is empty")
	case len(profile.CanonicalParameters) == 0:
		return manifestError(field+".canonicalParameters", "no canonical parameter exists")
	case len(profile.Interfaces)+len(profile.CallableParameters) == 0:
		return manifestError(field, "transported interfaces and callables are absent")
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
	parameters := make(map[string]struct{}, len(profile.CanonicalValues))
	for index, value := range profile.CanonicalValues {
		if value.SourceIdentity == "" || value.TargetParameter == "" {
			return manifestError(
				fmt.Sprintf("%s.canonicalValues[%d]", field, index),
				"source identity or target parameter is empty",
			)
		}
		if _, duplicate := parameters[value.TargetParameter]; duplicate {
			return manifestError(
				field+".canonicalValues",
				"target parameter is duplicated",
			)
		}
		parameters[value.TargetParameter] = struct{}{}
	}
	keyInterfaces := make(
		[]ProviderCallableProfileKeyInterface,
		0,
		len(profile.Interfaces),
	)
	interfaceIdentities := make(map[string]struct{}, len(profile.Interfaces))
	protocolIdentities := make(map[string]struct{})
	previous := ""
	for index, selected := range profile.Interfaces {
		selectedField := fmt.Sprintf("%s.interfaces[%d]", field, index)
		if err := validateProviderCallableProfileInterface(
			selected,
			selectedField,
		); err != nil {
			return err
		}
		identity := selected.SourceIdentity
		if identity <= previous {
			return manifestError(field+".interfaces", "values are not strictly ordered")
		}
		previous = identity
		interfaceIdentities[identity] = struct{}{}
		if selected.Protocol != nil {
			protocolIdentities[identity] = struct{}{}
			if selected.ProtocolValueParameter == nil {
				return manifestError(
					selectedField+".protocolValueParameter",
					"value is absent",
				)
			}
			if _, found := slices.BinarySearch(
				profile.CanonicalParameters,
				*selected.ProtocolValueParameter,
			); !found {
				return manifestError(
					field+".canonicalParameters",
					"protocol value parameter is not a canonical root",
				)
			}
			parameters, err := ProviderProtocolCallableParameters(*selected.Protocol)
			if err != nil {
				return err
			}
			for _, parameter := range parameters {
				if _, found := slices.BinarySearch(
					profile.CanonicalParameters,
					parameter,
				); !found {
					return manifestError(
						field+".canonicalParameters",
						"protocol callable parameter is not a canonical root",
					)
				}
			}
		}
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
	viewKeys := make(map[string]struct{}, len(profile.CapabilityViews))
	viewParameters := make(map[string]struct{}, len(profile.CapabilityViews))
	for index, view := range profile.CapabilityViews {
		viewField := fmt.Sprintf("%s.capabilityViews[%d]", field, index)
		if view.BaseSourceIdentity == "" || view.TargetSourceIdentity == "" ||
			view.TargetParameter == "" ||
			view.BaseSourceIdentity == view.TargetSourceIdentity {
			return manifestError(viewField, "capability view is incomplete")
		}
		if _, found := interfaceIdentities[view.BaseSourceIdentity]; !found {
			return manifestError(
				viewField+".baseSourceIdentity",
				"value has no profile-interface evidence",
			)
		}
		if _, found := interfaceIdentities[view.TargetSourceIdentity]; !found {
			return manifestError(
				viewField+".targetSourceIdentity",
				"value has no profile-interface evidence",
			)
		}
		key := view.BaseSourceIdentity + "\x00" + view.TargetSourceIdentity
		if _, duplicate := viewKeys[key]; duplicate {
			return manifestError(
				field+".capabilityViews",
				"base/target pair is duplicated",
			)
		}
		if _, duplicate := viewParameters[view.TargetParameter]; duplicate {
			return manifestError(
				field+".capabilityViews",
				"target parameter is duplicated",
			)
		}
		viewKeys[key] = struct{}{}
		viewParameters[view.TargetParameter] = struct{}{}
	}
	keyCallables := make(
		[]ProviderCallableProfileKeyCallable,
		0,
		len(profile.CallableParameters),
	)
	if err := validateCallableParameters(
		profile.CallableParameters,
		field+".callableParameters",
		false,
	); err != nil {
		return err
	}
	for index, selected := range profile.CallableParameters {
		callableField := fmt.Sprintf("%s.callableParameters[%d]", field, index)
		if _, found := slices.BinarySearch(
			profile.CanonicalParameters,
			selected.Parameter,
		); !found {
			return manifestError(
				callableField+".parameter",
				"value is not a canonical parameter root",
			)
		}
		keyCallables = append(keyCallables, ProviderCallableProfileKeyCallable{
			Parameter: selected.Parameter,
			Effect:    selected.Effect,
		})
	}
	if _, err := providerProfileBoundaryEffect(
		profile.Interfaces,
		profile.CallableParameters,
		field,
	); err != nil {
		return err
	}
	seenTypeArguments := make(map[string]struct{}, len(profile.CanonicalTypeArguments))
	for index, identity := range profile.CanonicalTypeArguments {
		if _, ok := interfaceIdentities[identity]; !ok {
			return manifestError(
				fmt.Sprintf("%s.canonicalTypeArguments[%d]", field, index),
				"value has no profile-interface evidence",
			)
		}
		if _, duplicate := seenTypeArguments[identity]; duplicate {
			return manifestError(
				fmt.Sprintf("%s.canonicalTypeArguments[%d]", field, index),
				"value is duplicated",
			)
		}
		seenTypeArguments[identity] = struct{}{}
	}
	key, err := BuildImplementedResultProfileKey(
		keyInterfaces,
		keyCallables,
		profile.ImplementedResultInterfaces,
	)
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
	if profile.Required != (len(protocolIdentities) != 0) {
		return manifestError(
			field+".required",
			"value disagrees with semantic protocol evidence",
		)
	}
	for identity := range protocolIdentities {
		if _, guarded := seenGuards[identity]; !guarded {
			return manifestError(
				field+".guardInterfaces",
				"semantic protocol is not a generated guard",
			)
		}
	}
	seenContracts := make(map[string]struct{}, len(profile.ContractInterfaces))
	for index, identity := range profile.ContractInterfaces {
		contractField := fmt.Sprintf("%s.contractInterfaces[%d]", field, index)
		if _, ok := interfaceIdentities[identity]; !ok {
			return manifestError(contractField, "value has no profile-interface evidence")
		}
		if _, duplicate := seenContracts[identity]; duplicate {
			return manifestError(contractField, "value is duplicated")
		}
		seenContracts[identity] = struct{}{}
	}
	previousBridge := ""
	for index, identity := range profile.FromProviderInterfaces {
		bridgeField := fmt.Sprintf("%s.fromProviderInterfaces[%d]", field, index)
		if identity <= previousBridge {
			return manifestError(
				field+".fromProviderInterfaces",
				"values are empty, duplicated, or not strictly ordered",
			)
		}
		previousBridge = identity
		if _, ok := interfaceIdentities[identity]; !ok {
			return manifestError(bridgeField, "value has no profile-interface evidence")
		}
	}
	previousImplemented := ""
	for index, identity := range profile.ImplementedResultInterfaces {
		implementedField := fmt.Sprintf(
			"%s.implementedResultInterfaces[%d]",
			field,
			index,
		)
		if identity <= previousImplemented {
			return manifestError(
				field+".implementedResultInterfaces",
				"values are empty, duplicated, or not strictly ordered",
			)
		}
		previousImplemented = identity
		if _, ok := seenContracts[identity]; !ok {
			return manifestError(
				implementedField,
				"value is not a contract interface",
			)
		}
	}
	return nil
}

func providerProfileBoundaryEffect(
	interfaces []ProviderCallableProfileInterfaceDocument,
	callables []ProviderCallableParameterDocument,
	field string,
) (EffectKind, error) {
	effect := EffectInvalid
	for interfaceIndex, selected := range interfaces {
		for methodIndex, method := range selected.ProviderInterface.Methods {
			methodField := fmt.Sprintf(
				"%s[%d].providerInterface.methods[%d].effect",
				field,
				interfaceIndex,
				methodIndex,
			)
			if method.Effect != EffectSynchronous && method.Effect != EffectAwaitable {
				return EffectInvalid, manifestError(
					methodField,
					"transported method is neither direct nor awaitable",
				)
			}
			if effect == EffectInvalid {
				effect = method.Effect
				continue
			}
			if method.Effect != effect {
				return EffectInvalid, manifestError(
					field+".interfaces",
					"profile mixes direct and cooperative transported methods",
				)
			}
		}
	}
	for index, callable := range callables {
		callableField := fmt.Sprintf(
			"%s.callableParameters[%d].effect",
			field,
			index,
		)
		if callable.Effect != EffectSynchronous && callable.Effect != EffectAwaitable {
			return EffectInvalid, manifestError(
				callableField,
				"transported callable is neither direct nor awaitable",
			)
		}
		if effect == EffectInvalid {
			effect = callable.Effect
			continue
		}
		if callable.Effect != effect {
			return EffectInvalid, manifestError(
				field,
				"profile mixes direct and cooperative transported callables",
			)
		}
	}
	if effect == EffectInvalid {
		return EffectInvalid, manifestError(
			field,
			"profile has no transported method effect",
		)
	}
	return effect, nil
}

func sameProviderCallableProfileInterface(
	left ProviderCallableProfileInterfaceDocument,
	right ProviderCallableProfileInterfaceDocument,
) bool {
	protocolsEqual := left.Protocol == nil && right.Protocol == nil
	if left.Protocol != nil && right.Protocol != nil {
		protocolsEqual = sameProviderProtocolInterface(*left.Protocol, *right.Protocol)
	}
	return left.SourceIdentity == right.SourceIdentity &&
		left.Export == right.Export &&
		protocolsEqual &&
		sameOptionalIndex(
			left.ProtocolValueParameter,
			right.ProtocolValueParameter,
		) &&
		left.TargetFingerprint == right.TargetFingerprint &&
		left.ProviderInterface.Mode == right.ProviderInterface.Mode &&
		slices.Equal(
			left.ProviderInterface.Methods,
			right.ProviderInterface.Methods,
		)
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
	if selected.Protocol != nil {
		if selected.ProtocolValueParameter == nil ||
			*selected.ProtocolValueParameter < 0 {
			return manifestError(
				field+".protocolValueParameter",
				"value is absent or negative",
			)
		}
		canonical, err := CanonicalProviderProtocolInterface(*selected.Protocol)
		if err != nil {
			return err
		}
		identity, err := BuildProviderProtocolInterfaceIdentity(canonical)
		if err != nil {
			return err
		}
		if identity != selected.SourceIdentity {
			return manifestError(
				field+".sourceIdentity",
				"value does not match protocol evidence",
			)
		}
	} else if selected.ProtocolValueParameter != nil {
		return manifestError(
			field+".protocolValueParameter",
			"named interface has a protocol value parameter",
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

func sameOptionalIndex(left *int, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
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
