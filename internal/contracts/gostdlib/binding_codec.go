package gostdlib

import "fmt"

func validateBinding(binding BindingDocument, field string) error {
	requiresEffect := binding.Kind == BindingFunction
	allowsDefinedCallableEffect := binding.Kind == BindingType &&
		binding.DefinedValue == DefinedValueRepresentationCanonical
	switch {
	case binding.Identity == "":
		return manifestError(field+".identity", "value is empty")
	case !binding.Kind.Valid():
		return manifestError(field+".kind", "value is invalid")
	case !binding.Access.Valid():
		return manifestError(field+".access", "value is invalid")
	case binding.Export == "":
		return manifestError(field+".export", "value is empty")
	case binding.Access == AccessExport && binding.Member != "":
		return manifestError(field+".member", "export access has a member")
	case binding.Access != AccessExport && binding.Member == "":
		return manifestError(field+".member", "member access has no member")
	case binding.Kind == BindingType && !binding.Representation.Valid():
		return manifestError(field+".representation", "type representation is invalid")
	case binding.Kind != BindingType && binding.Representation != RepresentationInvalid:
		return manifestError(field+".representation", "non-type has a representation")
	case binding.Kind != BindingType &&
		binding.DefinedValue != DefinedValueRepresentationInvalid:
		return manifestError(
			field+".definedValue",
			"non-type has a defined-value representation",
		)
	case binding.Kind == BindingType &&
		binding.DefinedValue != DefinedValueRepresentationInvalid &&
		!binding.DefinedValue.Valid():
		return manifestError(
			field+".definedValue",
			"defined-value representation is invalid",
		)
	case requiresEffect && !binding.Effect.Valid():
		return manifestError(field+".effect", "callable effect is invalid")
	case !requiresEffect && !allowsDefinedCallableEffect &&
		binding.Effect != EffectInvalid:
		return manifestError(field+".effect", "non-callable has an effect")
	case allowsDefinedCallableEffect && binding.Effect != EffectInvalid &&
		!binding.Effect.Valid():
		return manifestError(field+".effect", "defined-callable effect is invalid")
	case binding.ProviderInterface != nil &&
		(binding.Kind != BindingType || binding.Access != AccessExport ||
			binding.Representation != RepresentationDirect):
		return manifestError(
			field+".providerInterface",
			"provider-interface evidence does not belong to a direct exported type",
		)
	case len(binding.StructFields) != 0 &&
		(binding.Kind != BindingType || binding.Access != AccessExport ||
			binding.Representation != RepresentationDirect):
		return manifestError(
			field+".structFields",
			"struct-field evidence does not belong to a direct exported type",
		)
	case binding.SourceSignature == "":
		return manifestError(field+".sourceSignature", "value is empty")
	case binding.SourceLocation == "":
		return manifestError(field+".sourceLocation", "value is empty")
	case !sourcePath(binding.ImplementationOwner):
		return manifestError(
			field+".implementationOwner",
			"value is not a provider source path",
		)
	case !validDigest(binding.TargetFingerprint):
		return manifestError(
			field+".targetFingerprint",
			"value is not a sha256 digest",
		)
	}
	if err := validateGenericOperations(
		binding.GenericOperations,
		field+".genericOperations",
		true,
	); err != nil {
		return err
	}
	if binding.Kind != BindingFunction && len(binding.CallableParameters) != 0 {
		return manifestError(
			field+".callableParameters",
			"parameters do not belong to a callable",
		)
	}
	if err := validateCallableParameters(
		binding.CallableParameters,
		field+".callableParameters",
	); err != nil {
		return err
	}
	if binding.ProviderInterface != nil {
		if err := validateProviderInterface(
			*binding.ProviderInterface,
			field+".providerInterface",
		); err != nil {
			return err
		}
	}
	if err := validateProviderStructFields(
		binding.StructFields,
		field+".structFields",
	); err != nil {
		return err
	}
	if err := validateGenericTypeArguments(
		binding.GenericTypeArguments,
		field+".genericTypeArguments",
	); err != nil {
		return err
	}
	if len(binding.GenericTypeArguments) != 0 &&
		(binding.Kind != BindingFunction || binding.Access != AccessExport) {
		return manifestError(
			field+".genericTypeArguments",
			"type arguments do not belong to an exported function",
		)
	}
	if len(binding.GenericOperations) != 0 &&
		(binding.Kind != BindingFunction || binding.Access != AccessExport) {
		return manifestError(
			field+".genericOperations",
			"operations do not belong to an exported function",
		)
	}
	switch binding.Access {
	case AccessStateMember:
		if binding.Kind != BindingVariable || binding.Export != "state" {
			return manifestError(
				field+".access",
				"state access does not own a variable",
			)
		}
	case AccessStaticMethod, AccessInstanceMethod:
		if binding.Kind != BindingFunction {
			return manifestError(
				field+".access",
				"method access does not own a function",
			)
		}
	}
	return nil
}

func validateProviderStructFields(
	fields []ProviderStructFieldDocument,
	field string,
) error {
	for index, selected := range fields {
		selectedField := fmt.Sprintf("%s[%d]", field, index)
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
		if index != 0 && selected.Member <= fields[index-1].Member {
			return manifestError(
				field,
				"fields are not strictly ordered",
			)
		}
	}
	return nil
}

func validateGenericTypeArguments(
	arguments []GenericTypeArgumentDocument,
	field string,
) error {
	seen := make(map[GenericTypeArgumentDocument]struct{}, len(arguments))
	for _, argument := range arguments {
		if argument.TypeParameter < 0 || !argument.Facet.Valid() {
			return manifestError(
				field,
				"source parameter or representation facet is invalid",
			)
		}
		if _, duplicate := seen[argument]; duplicate {
			return manifestError(field, "target projection entry is duplicated")
		}
		seen[argument] = struct{}{}
	}
	return nil
}
