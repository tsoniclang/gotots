package gostdlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

type ManifestError struct {
	Field  string
	Reason string
}

func (e *ManifestError) Error() string {
	if e.Field == "" {
		return "validate gostdlib manifest: " + e.Reason
	}
	return fmt.Sprintf(
		"validate gostdlib manifest field %q: %s",
		e.Field,
		e.Reason,
	)
}

func Seal(source Document) ([]byte, error) {
	document := cloneDocument(source)
	if document.ManifestDigest != "" {
		return nil, manifestError("manifestDigest", "must be empty before sealing")
	}
	if err := validateDocument(document, false); err != nil {
		return nil, err
	}
	digest, err := documentDigest(document)
	if err != nil {
		return nil, err
	}
	document.ManifestDigest = digest
	return encodeDocument(document)
}

func Parse(payload []byte) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Manifest{}, manifestError("", err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return Manifest{}, manifestError("", err.Error())
	}
	if err := validateDocument(document, true); err != nil {
		return Manifest{}, err
	}
	want := document.ManifestDigest
	document.ManifestDigest = ""
	got, err := documentDigest(document)
	if err != nil {
		return Manifest{}, err
	}
	if got != want {
		return Manifest{}, manifestError(
			"manifestDigest",
			"content digest does not match payload",
		)
	}
	document.ManifestDigest = want
	canonical, err := encodeDocument(document)
	if err != nil {
		return Manifest{}, err
	}
	bindings := make(map[string]Binding)
	for _, module := range document.Modules {
		for _, binding := range module.Bindings {
			bindings[binding.Identity] = newBinding(module, binding)
		}
	}
	representations := make(
		map[providerRepresentationLookup]ProviderRepresentation,
	)
	callableProfiles := make(
		map[providerCallableProfileLookup]ProviderCallableProfile,
	)
	statefulProfiles := make(
		map[providerStatefulProfileLookup]ProviderStatefulProfile,
	)
	providerInterfaces := make(map[string]ProviderInterfaceBinding)
	for _, module := range document.FacetModules {
		for _, representation := range module.Representations {
			representations[providerRepresentationLookup{
				module: module.Specifier,
				export: representation.Export,
			}] = newProviderRepresentation(module, representation)
		}
		for _, profile := range module.CallableProfiles {
			callableProfiles[providerCallableProfileLookup{
				sourceIdentity: profile.SourceIdentity,
				profileKey:     profile.ProfileKey,
			}] = newProviderCallableProfile(module, profile)
		}
		for _, profile := range module.StatefulProfiles {
			statefulProfiles[providerStatefulProfileLookup{
				sourceIdentity: profile.SourceIdentity,
				profileKey:     profile.ProfileKey,
			}] = newProviderStatefulProfile(module, profile)
		}
		for _, selected := range module.ProviderInterfaces {
			providerInterfaces[selected.SourceIdentity] =
				newProviderInterfaceBinding(module, selected)
		}
	}
	facets := make(map[facetLookup]Facet)
	for _, module := range document.FacetModules {
		for _, facet := range module.Facets {
			capabilities := make([]string, 0, len(facet.Capabilities)+1)
			for _, capability := range facet.Capabilities {
				capabilities = append(capabilities, string(capability))
			}
			if facet.ProfileKey != "" {
				capabilities = append(capabilities, facet.ProfileKey)
			}
			for _, capability := range capabilities {
				facets[facetLookup{
					sourceIdentity: facet.SourceIdentity,
					kind:           facet.Kind,
					capability:     capability,
				}] = newFacet(module, facet)
			}
		}
	}
	return Manifest{
		document:           cloneDocument(document),
		payload:            canonical,
		bindings:           bindings,
		facets:             facets,
		representations:    representations,
		providerInterfaces: providerInterfaces,
		callableProfiles:   callableProfiles,
		statefulProfiles:   statefulProfiles,
	}, nil
}

func Encode(manifest Manifest) ([]byte, error) {
	if manifest.document.ManifestDigest == "" || len(manifest.payload) == 0 {
		return nil, manifestError("", "manifest is invalid")
	}
	return bytes.Clone(manifest.payload), nil
}

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
	case !strings.HasPrefix(document.GoVersion, "go"):
		return manifestError("goVersion", "value is invalid")
	case !strings.HasPrefix(document.MinimumGoVersion, "go"):
		return manifestError("minimumGoVersion", "value is invalid")
	case !strings.HasPrefix(document.MaximumGoVersion, "go"):
		return manifestError("maximumGoVersion", "value is invalid")
	case !validDigest(document.RuntimeDigest):
		return manifestError("runtimeDigest", "value is not a sha256 digest")
	case !validDigest(document.ProviderDigest):
		return manifestError("providerDigest", "value is not a sha256 digest")
	case sealed && !validDigest(document.ManifestDigest):
		return manifestError("manifestDigest", "value is not a sha256 digest")
	case !sealed && document.ManifestDigest != "":
		return manifestError("manifestDigest", "must be empty before sealing")
	case len(document.Modules) == 0:
		return manifestError("modules", "set is empty")
	}
	modulePaths := make(map[string]struct{}, len(document.Modules))
	specifiers := make(map[string]struct{}, len(document.Modules))
	identities := make(map[string]struct{})
	previousModule := ""
	for index, module := range document.Modules {
		field := fmt.Sprintf("modules[%d]", index)
		if err := validateModule(module, field); err != nil {
			return err
		}
		if previousModule != "" && module.GoImportPath <= previousModule {
			return manifestError("modules", "modules are not strictly ordered")
		}
		previousModule = module.GoImportPath
		if _, duplicate := modulePaths[module.GoImportPath]; duplicate {
			return manifestError(field+".goImportPath", "value is duplicated")
		}
		modulePaths[module.GoImportPath] = struct{}{}
		if _, duplicate := specifiers[module.Specifier]; duplicate {
			return manifestError(field+".specifier", "value is duplicated")
		}
		specifiers[module.Specifier] = struct{}{}
		previousBinding := ""
		for bindingIndex, binding := range module.Bindings {
			bindingField := fmt.Sprintf("%s.bindings[%d]", field, bindingIndex)
			if err := validateBinding(binding, bindingField); err != nil {
				return err
			}
			if previousBinding != "" && binding.Identity <= previousBinding {
				return manifestError(field+".bindings", "bindings are not strictly ordered")
			}
			previousBinding = binding.Identity
			if _, duplicate := identities[binding.Identity]; duplicate {
				return manifestError(bindingField+".identity", "value is duplicated")
			}
			identities[binding.Identity] = struct{}{}
		}
	}
	previousFacetModule := ""
	facetLookups := make(map[facetLookup]struct{})
	callableProfileLookups := make(map[providerCallableProfileLookup]struct{})
	statefulProfileLookups := make(map[providerStatefulProfileLookup]struct{})
	providerInterfaceLookups := make(map[string]struct{})
	for index, module := range document.FacetModules {
		field := fmt.Sprintf("facetModules[%d]", index)
		if err := validateFacetModule(
			module,
			field,
			facetLookups,
			callableProfileLookups,
			statefulProfileLookups,
			providerInterfaceLookups,
		); err != nil {
			return err
		}
		if previousFacetModule != "" && module.Specifier <= previousFacetModule {
			return manifestError("facetModules", "modules are not strictly ordered")
		}
		previousFacetModule = module.Specifier
		if _, duplicate := specifiers[module.Specifier]; duplicate {
			return manifestError(field+".specifier", "value is duplicated")
		}
		specifiers[module.Specifier] = struct{}{}
	}
	return nil
}
