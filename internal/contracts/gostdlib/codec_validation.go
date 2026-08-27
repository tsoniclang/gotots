package gostdlib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"
	"strings"
)

func validateFacet(facet FacetDocument, field string) error {
	switch {
	case !facet.Kind.Valid():
		return manifestError(field+".kind", "value is invalid")
	case facet.SourceIdentity == "":
		return manifestError(field+".sourceIdentity", "value is empty")
	case facet.Export == "":
		return manifestError(field+".export", "value is empty")
	case !sourcePath(facet.ImplementationOwner):
		return manifestError(field+".implementationOwner", "value is not a provider source path")
	case !validDigest(facet.TargetFingerprint):
		return manifestError(field+".targetFingerprint", "value is not a sha256 digest")
	}
	for index, capability := range facet.Capabilities {
		if index != 0 && capability <= facet.Capabilities[index-1] {
			return manifestError(field+".capabilities", "values are not strictly ordered")
		}
	}
	switch facet.Kind {
	case FacetNamedStructOperations:
		if len(facet.Capabilities) == 0 ||
			facet.Effect != EffectInvalid || len(facet.CallableParameters) != 0 ||
			len(facet.GenericTypeArguments) != 0 ||
			facet.ResultExport != "" || facet.ResultImplementationOwner != "" ||
			facet.ResultTargetFingerprint != "" {
			return manifestError(field, "named-struct facet shape is invalid")
		}
		storage := false
		representation := false
		for _, capability := range facet.Capabilities {
			if !capability.NamedStructOperation() {
				return manifestError(field+".capabilities", "named-struct capability is invalid")
			}
			storage = storage || capability == FacetCapabilityStorage
			representation = representation ||
				capability == FacetCapabilityRepresentation
		}
		if storage != (facet.StorageExport != "") ||
			storage != (facet.StorageImplementationOwner != "") ||
			storage != (facet.StorageTargetFingerprint != "") {
			return manifestError(field, "storage target shape is invalid")
		}
		if storage && (!sourcePath(facet.StorageImplementationOwner) ||
			!validDigest(facet.StorageTargetFingerprint)) {
			return manifestError(field, "storage target evidence is invalid")
		}
		if representation != (facet.RepresentationExport != "") {
			return manifestError(field, "representation target shape is invalid")
		}
	case FacetDefinedValueOperations:
		if len(facet.Capabilities) != 2 ||
			facet.Capabilities[0] != FacetCapabilityProject ||
			facet.Capabilities[1] != FacetCapabilityWrap ||
			(facet.Effect != EffectInvalid && !facet.Effect.Valid()) ||
			len(facet.CallableParameters) != 0 ||
			len(facet.GenericTypeArguments) != 0 ||
			facet.ResultExport != "" || facet.ResultImplementationOwner != "" ||
			facet.ResultTargetFingerprint != "" ||
			facet.StorageExport != "" || facet.RepresentationExport != "" ||
			facet.StorageImplementationOwner != "" ||
			facet.StorageTargetFingerprint != "" {
			return manifestError(field, "defined-value facet shape is invalid")
		}
	case FacetRecoveryCallable:
		if len(facet.Capabilities) != 1 ||
			facet.Capabilities[0] != FacetCapabilityRecovery ||
			!facet.Effect.Valid() ||
			len(facet.CallableParameters) != 0 ||
			len(facet.GenericTypeArguments) != 0 ||
			facet.ResultExport != "" || facet.ResultImplementationOwner != "" ||
			facet.ResultTargetFingerprint != "" ||
			facet.StorageExport != "" ||
			facet.RepresentationExport != "" ||
			facet.StorageImplementationOwner != "" ||
			facet.StorageTargetFingerprint != "" {
			return manifestError(field, "recovery facet shape is invalid")
		}
	case FacetGenericCallableKernel:
		if len(facet.Capabilities) != 1 ||
			facet.Capabilities[0] != FacetCapabilityKernel ||
			!facet.Effect.Valid() ||
			len(facet.GenericTypeArguments) == 0 ||
			facet.ResultExport != "" || facet.ResultImplementationOwner != "" ||
			facet.ResultTargetFingerprint != "" ||
			facet.StorageExport != "" || facet.RepresentationExport != "" ||
			facet.StorageImplementationOwner != "" ||
			facet.StorageTargetFingerprint != "" {
			return manifestError(field, "generic-kernel facet shape is invalid")
		}
		if err := validateGenericTypeArguments(
			facet.GenericTypeArguments,
			field+".genericTypeArguments",
		); err != nil {
			return err
		}
		if err := validateCallableParameters(
			facet.CallableParameters,
			field+".callableParameters",
		); err != nil {
			return err
		}
	case FacetReflectionTypeOperations:
		if len(facet.Capabilities) != 1 ||
			facet.Capabilities[0] != FacetCapabilityMetadata ||
			facet.Effect != EffectInvalid ||
			len(facet.CallableParameters) != 0 ||
			len(facet.GenericTypeArguments) != 0 ||
			facet.ResultExport == "" ||
			!sourcePath(facet.ResultImplementationOwner) ||
			!validDigest(facet.ResultTargetFingerprint) ||
			facet.StorageExport != "" || facet.RepresentationExport != "" ||
			facet.StorageImplementationOwner != "" ||
			facet.StorageTargetFingerprint != "" {
			return manifestError(field, "reflection-type facet shape is invalid")
		}
	}
	return nil
}

func validateModule(module ModuleDocument, field string) error {
	if !canonicalRelative(module.GoImportPath) {
		return manifestError(field+".goImportPath", "value is not canonical")
	}
	if !strings.HasPrefix(module.Specifier, PackageName+"/") ||
		!strings.HasSuffix(module.Specifier, ".js") {
		return manifestError(field+".specifier", "value is not a provider module")
	}
	if !sourcePath(module.SourcePath) {
		return manifestError(field+".sourcePath", "value is not a provider source path")
	}
	return nil
}

func documentDigest(document Document) (string, error) {
	document.ManifestDigest = ""
	payload, err := json.Marshal(document)
	if err != nil {
		return "", manifestError("", err.Error())
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func encodeDocument(document Document) ([]byte, error) {
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, manifestError("", err.Error())
	}
	return append(payload, '\n'), nil
}

func manifestError(field string, reason string) error {
	return &ManifestError{Field: field, Reason: reason}
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' &&
			character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}

func canonicalRelative(value string) bool {
	return value != "" && value != "." && !path.IsAbs(value) &&
		path.Clean(value) == value && !strings.HasPrefix(value, "../")
}

func sourcePath(value string) bool {
	return canonicalRelative(value) && strings.HasPrefix(value, "src/") &&
		strings.HasSuffix(value, ".ts")
}
