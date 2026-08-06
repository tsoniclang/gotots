package externals

import (
	"crypto/sha256"
	"fmt"
	"path"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

func validateDocument(document Document, sealed bool) error {
	if document.BuildTags == nil {
		return manifestError("buildTags", "set is absent")
	}
	if _, err := environmentcontract.NewBuildProfileForToolchain(
		document.GoVersion,
		document.GOOS,
		document.GOARCH,
		document.CGOEnabled,
		document.BuildTags,
	); err != nil {
		return manifestError("buildProfile", err.Error())
	}
	switch {
	case document.SchemaVersion != SchemaVersion:
		return manifestError("schemaVersion", "unsupported schema")
	case document.PackageName != PackageName:
		return manifestError("packageName", "package identity is invalid")
	case document.PackageVersion == "":
		return manifestError("packageVersion", "value is empty")
	case document.Backend == "":
		return manifestError("backend", "value is empty")
	case document.ProviderIntegerRepresentation != "number" &&
		document.ProviderIntegerRepresentation != "bigint":
		return manifestError("providerIntegerRepresentation", "value is invalid")
	case !validDigest(document.StandardLibraryDigest):
		return manifestError("standardLibraryDigest", "value is not a sha256 digest")
	case !validDigest(document.ProviderDigest):
		return manifestError("providerDigest", "value is not a sha256 digest")
	case sealed && !validDigest(document.ManifestDigest):
		return manifestError("manifestDigest", "value is not a sha256 digest")
	case !sealed && document.ManifestDigest != "":
		return manifestError("manifestDigest", "must be empty before sealing")
	case len(document.Bindings) == 0:
		return manifestError("bindings", "set is empty")
	}
	previous := ""
	for index, binding := range document.Bindings {
		field := fmt.Sprintf("bindings[%d]", index)
		if err := validateBinding(binding, field); err != nil {
			return err
		}
		if previous != "" && binding.SourceIdentity <= previous {
			return manifestError("bindings", "values are not strictly ordered")
		}
		previous = binding.SourceIdentity
	}
	return nil
}

func validateBinding(binding BindingDocument, field string) error {
	switch {
	case binding.SourceIdentity == "":
		return manifestError(field+".sourceIdentity", "value is empty")
	case binding.SourceSignature == "":
		return manifestError(field+".sourceSignature", "value is empty")
	case !canonicalImportPath(binding.SourceModulePath):
		return manifestError(field+".sourceModulePath", "value is not canonical")
	case binding.SourceModuleVersion == "":
		return manifestError(field+".sourceModuleVersion", "value is empty")
	case binding.SourceLocation == "":
		return manifestError(field+".sourceLocation", "value is empty")
	case !binding.TargetKind.Valid():
		return manifestError(field+".targetKind", "value is invalid")
	}
	switch binding.TargetKind {
	case TargetModule:
		if !providerSpecifier(binding.ModuleSpecifier) ||
			!exportName(binding.Export) ||
			!providerSourcePath(binding.ImplementationOwner) ||
			!validDigest(binding.TargetFingerprint) {
			return manifestError(field, "module target evidence is incomplete")
		}
		if binding.TargetIdentity != "" || binding.TargetSignature != "" ||
			binding.TargetLocation != "" {
			return manifestError(field, "module target carries source-target evidence")
		}
	case TargetSource:
		if binding.TargetIdentity == "" || binding.TargetSignature == "" ||
			binding.TargetLocation == "" {
			return manifestError(field, "source target evidence is incomplete")
		}
		if binding.ModuleSpecifier != "" || binding.Export != "" ||
			binding.ImplementationOwner != "" || binding.TargetFingerprint != "" {
			return manifestError(field, "source target carries module-target evidence")
		}
	}
	return nil
}

func cloneDocument(source Document) Document {
	result := source
	result.BuildTags = append([]string(nil), source.BuildTags...)
	result.Bindings = append([]BindingDocument(nil), source.Bindings...)
	return result
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

func canonicalImportPath(value string) bool {
	return value != "" && value != "." && !path.IsAbs(value) &&
		path.Clean(value) == value && !strings.HasPrefix(value, "../")
}

func providerSpecifier(value string) bool {
	prefix := PackageName + "/"
	return strings.HasPrefix(value, prefix) &&
		strings.HasSuffix(value, ".js") &&
		!strings.Contains(strings.TrimPrefix(value, prefix), "..")
}

func providerSourcePath(value string) bool {
	return canonicalImportPath(value) && strings.HasPrefix(value, "src/") &&
		strings.HasSuffix(value, ".ts")
}

func exportName(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if character == '_' || character == '$' ||
			character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			index > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}
