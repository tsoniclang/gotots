package gostdlib

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"slices"
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

func validateFacetModule(
	module FacetModuleDocument,
	field string,
	lookups map[facetLookup]struct{},
	callableProfileLookups map[providerCallableProfileLookup]struct{},
	statefulProfileLookups map[providerStatefulProfileLookup]struct{},
	providerInterfaceLookups map[string]struct{},
) error {
	if !strings.HasPrefix(
		module.Specifier,
		PackageName+"/internal/facets/",
	) || !strings.HasSuffix(module.Specifier, ".js") {
		return manifestError(field+".specifier", "value is not a compiler-facet module")
	}
	if !sourcePath(module.SourcePath) ||
		!strings.HasPrefix(module.SourcePath, "src/internal/facets/") {
		return manifestError(field+".sourcePath", "value is not a compiler-facet source")
	}
	if len(module.Facets) == 0 && len(module.CallableProfiles) == 0 &&
		len(module.StatefulProfiles) == 0 &&
		len(module.ProviderInterfaces) == 0 {
		return manifestError(
			field,
			"facet, provider-interface, and callable-profile sets are empty",
		)
	}
	owners := make(map[string]struct{})
	representations := make(
		map[string]ProviderRepresentationDocument,
		len(module.Representations),
	)
	previousRepresentation := ""
	for index, representation := range module.Representations {
		representationField := fmt.Sprintf("%s.representations[%d]", field, index)
		if err := validateProviderRepresentation(
			representation,
			representationField,
		); err != nil {
			return err
		}
		if previousRepresentation != "" &&
			representation.Export <= previousRepresentation {
			return manifestError(
				field+".representations",
				"representations are not strictly ordered",
			)
		}
		previousRepresentation = representation.Export
		if _, duplicate := owners[representation.Export]; duplicate {
			return manifestError(
				representationField+".export",
				"target owner is duplicated",
			)
		}
		owners[representation.Export] = struct{}{}
		representations[representation.Export] = representation
	}
	previousProviderInterface := ""
	for index, selected := range module.ProviderInterfaces {
		selectedField := fmt.Sprintf("%s.providerInterfaces[%d]", field, index)
		if err := validateProviderInterfaceBinding(
			selected,
			selectedField,
		); err != nil {
			return err
		}
		if selected.SourceIdentity <= previousProviderInterface {
			return manifestError(
				field+".providerInterfaces",
				"values are not strictly ordered",
			)
		}
		previousProviderInterface = selected.SourceIdentity
		if _, duplicate := providerInterfaceLookups[selected.SourceIdentity]; duplicate {
			return manifestError(
				selectedField+".sourceIdentity",
				"value is duplicated",
			)
		}
		if _, duplicate := owners[selected.Export]; duplicate {
			return manifestError(selectedField+".export", "target owner is duplicated")
		}
		providerInterfaceLookups[selected.SourceIdentity] = struct{}{}
		owners[selected.Export] = struct{}{}
	}
	profileInterfaceTargets := make(
		map[string]ProviderCallableProfileInterfaceDocument,
	)
	callableProfileTargets := make(map[string]ProviderCallableProfileDocument)
	statefulProfileTargets := make(map[string]ProviderStatefulProfileDocument)
	previousProfile := ""
	for index, profile := range module.CallableProfiles {
		profileField := fmt.Sprintf("%s.callableProfiles[%d]", field, index)
		if err := validateProviderCallableProfile(profile, profileField); err != nil {
			return err
		}
		if err := recordProviderProfileInterfaceTargets(
			profile.Interfaces,
			profileField,
			profileInterfaceTargets,
			owners,
		); err != nil {
			return err
		}
		key := profile.SourceIdentity + "\x00" + profile.ProfileKey
		if previousProfile != "" && key <= previousProfile {
			return manifestError(
				field+".callableProfiles",
				"profiles are not strictly ordered",
			)
		}
		previousProfile = key
		lookup := providerCallableProfileLookup{
			sourceIdentity: profile.SourceIdentity,
			profileKey:     profile.ProfileKey,
		}
		if _, duplicate := callableProfileLookups[lookup]; duplicate {
			return manifestError(profileField, "profile identity is duplicated")
		}
		callableProfileLookups[lookup] = struct{}{}
		if err := recordProviderCallableProfileTarget(
			profile,
			profileField,
			callableProfileTargets,
			owners,
		); err != nil {
			return err
		}
	}
	previousStatefulProfile := ""
	for index, profile := range module.StatefulProfiles {
		profileField := fmt.Sprintf("%s.statefulProfiles[%d]", field, index)
		if err := validateProviderStatefulProfile(profile, profileField); err != nil {
			return err
		}
		if err := recordProviderProfileInterfaceTargets(
			profile.Interfaces,
			profileField,
			profileInterfaceTargets,
			owners,
		); err != nil {
			return err
		}
		key := profile.SourceIdentity + "\x00" + profile.ProfileKey
		if previousStatefulProfile != "" && key <= previousStatefulProfile {
			return manifestError(
				field+".statefulProfiles",
				"profiles are not strictly ordered",
			)
		}
		previousStatefulProfile = key
		lookup := providerStatefulProfileLookup{
			sourceIdentity: profile.SourceIdentity,
			profileKey:     profile.ProfileKey,
		}
		if _, duplicate := statefulProfileLookups[lookup]; duplicate {
			return manifestError(profileField, "profile identity is duplicated")
		}
		statefulProfileLookups[lookup] = struct{}{}
		if err := recordProviderStatefulProfileTarget(
			profile,
			profileField,
			statefulProfileTargets,
			owners,
		); err != nil {
			return err
		}
	}
	previous := ""
	referencedRepresentations := make(map[string]struct{})
	for index, facet := range module.Facets {
		facetField := fmt.Sprintf("%s.facets[%d]", field, index)
		if err := validateFacet(facet, facetField); err != nil {
			return err
		}
		key := facet.SourceIdentity + "\x00" + string(facet.Kind) + "\x00" + facet.Export
		if previous != "" && key <= previous {
			return manifestError(field+".facets", "facets are not strictly ordered")
		}
		previous = key
		if facet.RepresentationExport != "" {
			if _, ok := representations[facet.RepresentationExport]; !ok {
				return manifestError(
					facetField+".representationExport",
					"representation is absent from the facet module",
				)
			}
			referencedRepresentations[facet.RepresentationExport] = struct{}{}
		}
		for _, target := range []string{facet.Export, facet.StorageExport} {
			if target == "" {
				continue
			}
			if _, duplicate := owners[target]; duplicate {
				return manifestError(facetField+".export", "target owner is duplicated")
			}
			owners[target] = struct{}{}
		}
		capabilities := make([]string, 0, len(facet.Capabilities)+1)
		for _, capability := range facet.Capabilities {
			capabilities = append(capabilities, string(capability))
		}
		if facet.ProfileKey != "" {
			capabilities = append(capabilities, facet.ProfileKey)
		}
		for _, capability := range capabilities {
			lookup := facetLookup{
				sourceIdentity: facet.SourceIdentity,
				kind:           facet.Kind,
				capability:     capability,
			}
			if _, duplicate := lookups[lookup]; duplicate {
				return manifestError(facetField, "capability owner is duplicated")
			}
			lookups[lookup] = struct{}{}
		}
	}
	for export := range representations {
		if _, referenced := referencedRepresentations[export]; !referenced {
			return manifestError(
				field+".representations",
				"representation has no facet reference",
			)
		}
	}
	return nil
}

func recordProviderProfileInterfaceTargets(
	interfaces []ProviderCallableProfileInterfaceDocument,
	field string,
	targets map[string]ProviderCallableProfileInterfaceDocument,
	owners map[string]struct{},
) error {
	for index, selected := range interfaces {
		selectedField := fmt.Sprintf("%s.interfaces[%d]", field, index)
		prior, exists := targets[selected.Export]
		if exists {
			if !sameProviderCallableProfileInterface(prior, selected) {
				return manifestError(
					selectedField+".export",
					"shared target interface evidence disagrees",
				)
			}
			continue
		}
		if _, duplicate := owners[selected.Export]; duplicate {
			return manifestError(selectedField+".export", "target owner is duplicated")
		}
		targets[selected.Export] = selected
		owners[selected.Export] = struct{}{}
	}
	return nil
}

func recordProviderCallableProfileTarget(
	profile ProviderCallableProfileDocument,
	field string,
	targets map[string]ProviderCallableProfileDocument,
	owners map[string]struct{},
) error {
	prior, exists := targets[profile.Export]
	if exists {
		if !sameProviderCallableProfileTarget(prior, profile) {
			return manifestError(
				field+".export",
				"shared callable target evidence disagrees",
			)
		}
		return nil
	}
	if _, duplicate := owners[profile.Export]; duplicate {
		return manifestError(field+".export", "target owner is duplicated")
	}
	targets[profile.Export] = profile
	owners[profile.Export] = struct{}{}
	return nil
}

func sameProviderCallableProfileTarget(
	left ProviderCallableProfileDocument,
	right ProviderCallableProfileDocument,
) bool {
	return left.SourceIdentity == right.SourceIdentity &&
		left.Export == right.Export &&
		left.Required == right.Required &&
		left.Receiver == right.Receiver &&
		left.Effect == right.Effect &&
		left.ImplementationOwner == right.ImplementationOwner &&
		left.TargetFingerprint == right.TargetFingerprint &&
		slices.Equal(left.CanonicalParameters, right.CanonicalParameters) &&
		slices.Equal(left.CanonicalResults, right.CanonicalResults) &&
		slices.Equal(left.CanonicalValues, right.CanonicalValues) &&
		slices.Equal(left.CanonicalTypeArguments, right.CanonicalTypeArguments) &&
		slices.Equal(left.GuardInterfaces, right.GuardInterfaces) &&
		slices.Equal(left.ContractInterfaces, right.ContractInterfaces) &&
		slices.Equal(left.FromProviderInterfaces, right.FromProviderInterfaces) &&
		slices.Equal(
			left.ImplementedResultInterfaces,
			right.ImplementedResultInterfaces,
		)
}

func recordProviderStatefulProfileTarget(
	profile ProviderStatefulProfileDocument,
	field string,
	targets map[string]ProviderStatefulProfileDocument,
	owners map[string]struct{},
) error {
	prior, exists := targets[profile.Export]
	if exists {
		if !sameProviderStatefulProfileTarget(prior, profile) {
			return manifestError(
				field+".export",
				"shared stateful target evidence disagrees",
			)
		}
		return nil
	}
	if _, duplicate := owners[profile.Export]; duplicate {
		return manifestError(field+".export", "target owner is duplicated")
	}
	targets[profile.Export] = profile
	owners[profile.Export] = struct{}{}
	return nil
}

func sameProviderStatefulProfileTarget(
	left ProviderStatefulProfileDocument,
	right ProviderStatefulProfileDocument,
) bool {
	return left.SourceIdentity == right.SourceIdentity &&
		left.Export == right.Export &&
		left.ImplementationOwner == right.ImplementationOwner &&
		left.TargetFingerprint == right.TargetFingerprint &&
		slices.Equal(left.TypeArguments, right.TypeArguments) &&
		slices.Equal(left.Methods, right.Methods)
}

func validateProviderRepresentation(
	representation ProviderRepresentationDocument,
	field string,
) error {
	switch {
	case representation.Export == "":
		return manifestError(field+".export", "value is empty")
	case len(representation.SourceTypes) == 0:
		return manifestError(field+".sourceTypes", "set is empty")
	case len(representation.SourceInterfaces) == 0:
		return manifestError(field+".sourceInterfaces", "set is empty")
	case len(representation.Methods) == 0:
		return manifestError(field+".methods", "set is empty")
	case !sourcePath(representation.ImplementationOwner):
		return manifestError(
			field+".implementationOwner",
			"value is not a provider source path",
		)
	case !validDigest(representation.TargetFingerprint):
		return manifestError(
			field+".targetFingerprint",
			"value is not a sha256 digest",
		)
	}
	for index, identity := range representation.SourceTypes {
		if identity == "" || index != 0 && identity <= representation.SourceTypes[index-1] {
			return manifestError(
				field+".sourceTypes",
				"values are empty, duplicated, or not strictly ordered",
			)
		}
	}
	for index, identity := range representation.SourceInterfaces {
		if identity == "" || index != 0 && identity <= representation.SourceInterfaces[index-1] {
			return manifestError(
				field+".sourceInterfaces",
				"values are empty, duplicated, or not strictly ordered",
			)
		}
	}
	for index, method := range representation.Methods {
		methodField := fmt.Sprintf("%s.methods[%d]", field, index)
		if err := validateProviderRepresentationMethod(method, methodField); err != nil {
			return err
		}
		if index != 0 && method.SourceIdentity <=
			representation.Methods[index-1].SourceIdentity {
			return manifestError(
				field+".methods",
				"methods are not strictly ordered",
			)
		}
	}
	return nil
}

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
		if len(facet.Capabilities) == 0 || facet.ProfileKey != "" ||
			facet.Effect != EffectInvalid {
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
			facet.ProfileKey != "" || facet.Effect != EffectInvalid ||
			facet.StorageExport != "" || facet.RepresentationExport != "" ||
			facet.StorageImplementationOwner != "" ||
			facet.StorageTargetFingerprint != "" {
			return manifestError(field, "defined-value facet shape is invalid")
		}
	case FacetRecoveryCallable:
		if len(facet.Capabilities) != 1 ||
			facet.Capabilities[0] != FacetCapabilityRecovery ||
			facet.ProfileKey != "" || !facet.Effect.Valid() ||
			facet.StorageExport != "" ||
			facet.RepresentationExport != "" ||
			facet.StorageImplementationOwner != "" ||
			facet.StorageTargetFingerprint != "" {
			return manifestError(field, "recovery facet shape is invalid")
		}
	case FacetGenericCallableProfile:
		if len(facet.Capabilities) != 0 ||
			!validProfileKey(facet.ProfileKey) ||
			!facet.Effect.Valid() || facet.StorageExport != "" ||
			facet.RepresentationExport != "" ||
			facet.StorageImplementationOwner != "" ||
			facet.StorageTargetFingerprint != "" {
			return manifestError(field, "generic-callable facet shape is invalid")
		}
	}
	return nil
}

func validProfileKey(value string) bool {
	if value == "" || strings.ContainsAny(value, "\x00\r\n\t ") {
		return false
	}
	for _, part := range strings.Split(value, "|") {
		key, selected, ok := strings.Cut(part, "=")
		if !ok || len(key) != sha256.Size*2 || selected != "cooperative" ||
			!validDigest(key) {
			return false
		}
	}
	return true
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
