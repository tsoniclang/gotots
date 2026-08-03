package gostdlib

import "fmt"

func validateProviderStatefulProfile(
	profile ProviderStatefulProfileDocument,
	field string,
) error {
	switch {
	case profile.SourceIdentity == "":
		return manifestError(field+".sourceIdentity", "value is empty")
	case !validDigest(profile.ProfileKey):
		return manifestError(field+".profileKey", "value is not a sha256 digest")
	case profile.Export == "":
		return manifestError(field+".export", "value is empty")
	case len(profile.Interfaces) == 0:
		return manifestError(field+".interfaces", "set is empty")
	case len(profile.TypeArguments) == 0:
		return manifestError(field+".typeArguments", "set is empty")
	case len(profile.Methods) == 0:
		return manifestError(field+".methods", "set is empty")
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
	keyInterfaces := make(
		[]ProviderCallableProfileKeyInterface,
		0,
		len(profile.Interfaces),
	)
	interfaceIdentities := make(map[string]struct{}, len(profile.Interfaces))
	for index, selected := range profile.Interfaces {
		selectedField := fmt.Sprintf("%s.interfaces[%d]", field, index)
		if err := validateProviderCallableProfileInterface(
			selected,
			selectedField,
		); err != nil {
			return err
		}
		identity := selected.SourceIdentity
		if index != 0 && identity <= profile.Interfaces[index-1].SourceIdentity {
			return manifestError(
				field+".interfaces",
				"values are empty, duplicated, or not strictly ordered",
			)
		}
		interfaceIdentities[identity] = struct{}{}
		methods := make(
			[]ProviderCallableProfileKeyMethod,
			0,
			len(selected.ProviderInterface.Methods),
		)
		for _, method := range selected.ProviderInterface.Methods {
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
	if _, err := providerProfileBoundaryEffect(
		profile.Interfaces,
		field+".interfaces",
	); err != nil {
		return err
	}
	seenTypeArguments := make(map[string]struct{}, len(profile.TypeArguments))
	for index, identity := range profile.TypeArguments {
		if _, ok := interfaceIdentities[identity]; !ok {
			return manifestError(
				fmt.Sprintf("%s.typeArguments[%d]", field, index),
				"value has no retained-interface evidence",
			)
		}
		if _, duplicate := seenTypeArguments[identity]; duplicate {
			return manifestError(
				fmt.Sprintf("%s.typeArguments[%d]", field, index),
				"value is duplicated",
			)
		}
		seenTypeArguments[identity] = struct{}{}
	}
	if len(seenTypeArguments) != len(interfaceIdentities) {
		return manifestError(
			field+".typeArguments",
			"values do not exact-join retained interfaces",
		)
	}
	wantKey, err := BuildProviderCallableProfileKey(keyInterfaces)
	if err != nil {
		return err
	}
	if profile.ProfileKey != wantKey {
		return manifestError(field+".profileKey", "value does not match interface evidence")
	}
	for index, capability := range profile.Operations {
		if !capability.NamedStructOperation() ||
			capability == FacetCapabilityRepresentation {
			return manifestError(
				fmt.Sprintf("%s.operations[%d]", field, index),
				"value is not a concrete named-struct operation",
			)
		}
		if index != 0 && capability <= profile.Operations[index-1] {
			return manifestError(
				field+".operations",
				"values are duplicated or not strictly ordered",
			)
		}
	}
	for index, selected := range profile.Fields {
		selectedField := fmt.Sprintf("%s.fields[%d]", field, index)
		switch {
		case selected.Member == "":
			return manifestError(selectedField+".member", "value is empty")
		case selected.Ordinal < 0:
			return manifestError(selectedField+".ordinal", "value is negative")
		case selected.SourceSignature == "":
			return manifestError(selectedField+".sourceSignature", "value is empty")
		case selected.SourceLocation == "":
			return manifestError(selectedField+".sourceLocation", "value is empty")
		case !sourcePath(selected.ImplementationOwner):
			return manifestError(
				selectedField+".implementationOwner",
				"value is not a provider source path",
			)
		case !validDigest(selected.TargetFingerprint):
			return manifestError(
				selectedField+".targetFingerprint",
				"value is not a sha256 digest",
			)
		}
		if index != 0 && selected.Member <= profile.Fields[index-1].Member {
			return manifestError(
				field+".fields",
				"fields are not strictly ordered",
			)
		}
	}
	for index, method := range profile.Methods {
		methodField := fmt.Sprintf("%s.methods[%d]", field, index)
		switch {
		case method.SourceIdentity == "":
			return manifestError(methodField+".sourceIdentity", "value is empty")
		case method.Member == "":
			return manifestError(methodField+".member", "value is empty")
		case !method.Effect.Valid():
			return manifestError(methodField+".effect", "value is invalid")
		case method.SourceSignature == "":
			return manifestError(methodField+".sourceSignature", "value is empty")
		case method.SourceLocation == "":
			return manifestError(methodField+".sourceLocation", "value is empty")
		case !sourcePath(method.ImplementationOwner):
			return manifestError(
				methodField+".implementationOwner",
				"value is not a provider source path",
			)
		case !validDigest(method.InstanceTargetFingerprint):
			return manifestError(
				methodField+".instanceTargetFingerprint",
				"value is not a sha256 digest",
			)
		case !validDigest(method.StaticTargetFingerprint):
			return manifestError(
				methodField+".staticTargetFingerprint",
				"value is not a sha256 digest",
			)
		}
		if index != 0 && method.SourceIdentity <=
			profile.Methods[index-1].SourceIdentity {
			return manifestError(field+".methods", "methods are not strictly ordered")
		}
	}
	return nil
}

func validateProviderRepresentationMethod(
	method ProviderRepresentationMethodDocument,
	field string,
) error {
	switch {
	case method.SourceIdentity == "":
		return manifestError(field+".sourceIdentity", "value is empty")
	case method.Member == "":
		return manifestError(field+".member", "value is empty")
	case !method.Effect.Valid():
		return manifestError(field+".effect", "value is invalid")
	case method.SourceSignature == "":
		return manifestError(field+".sourceSignature", "value is empty")
	case method.SourceLocation == "":
		return manifestError(field+".sourceLocation", "value is empty")
	case !sourcePath(method.ImplementationOwner):
		return manifestError(
			field+".implementationOwner",
			"value is not a provider source path",
		)
	case !validDigest(method.TargetFingerprint):
		return manifestError(
			field+".targetFingerprint",
			"value is not a sha256 digest",
		)
	}
	return nil
}
