package gostdlib

import (
	"fmt"
	"slices"
	"strings"
)

func validateFacetModule(
	module FacetModuleDocument,
	field string,
	lookups map[facetLookup]struct{},
	callableProfileLookups map[string]struct{},
	statefulProfileLookups map[string]struct{},
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
		boundaryEffect, err := providerProfileBoundaryEffect(
			profile.Interfaces,
			profile.CallableParameters,
			profileField,
		)
		if err != nil {
			return err
		}
		if err := recordProviderBoundaryProfile(
			callableProfileLookups,
			profile.SourceIdentity,
			boundaryEffect,
			profileField,
		); err != nil {
			return err
		}
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
		boundaryEffect, err := providerProfileBoundaryEffect(
			profile.Interfaces,
			nil,
			profileField,
		)
		if err != nil {
			return err
		}
		if err := recordProviderBoundaryProfile(
			statefulProfileLookups,
			profile.SourceIdentity,
			boundaryEffect,
			profileField,
		); err != nil {
			return err
		}
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
		capabilities := make([]string, 0, len(facet.Capabilities))
		for _, capability := range facet.Capabilities {
			capabilities = append(capabilities, string(capability))
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

func recordProviderBoundaryProfile(
	lookups map[string]struct{},
	sourceIdentity string,
	effect EffectKind,
	field string,
) error {
	key := sourceIdentity + "\x00" + string(effect)
	if _, duplicate := lookups[key]; duplicate {
		return manifestError(
			field,
			"source identity has multiple profiles for one boundary effect",
		)
	}
	lookups[key] = struct{}{}
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
		slices.Equal(left.CallableParameters, right.CallableParameters) &&
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
		slices.Equal(left.Operations, right.Operations) &&
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
