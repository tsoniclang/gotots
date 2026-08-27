package certify

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func cloneIndex(source *int) *int {
	if source == nil {
		return nil
	}
	result := *source
	return &result
}

func validateProviderProtocolSeeds(
	source []providerProtocolSeed,
) (map[string]gostdlib.ProviderProtocolInterfaceDocument, error) {
	result := make(
		map[string]gostdlib.ProviderProtocolInterfaceDocument,
		len(source),
	)
	identities := make(map[string]string, len(source))
	for _, selected := range source {
		if selected.Name == "" {
			return nil, certifyError(
				"configure provider protocols",
				"",
				"protocol name is empty",
			)
		}
		if _, duplicate := result[selected.Name]; duplicate {
			return nil, certifyError(
				"configure provider protocols",
				selected.Name,
				"protocol name is duplicated",
			)
		}
		canonical, err := gostdlib.CanonicalProviderProtocolInterface(
			selected.Protocol,
		)
		if err != nil {
			return nil, certifyError(
				"configure provider protocols",
				selected.Name,
				err.Error(),
			)
		}
		identity, err := gostdlib.BuildProviderProtocolInterfaceIdentity(canonical)
		if err != nil {
			return nil, err
		}
		if prior, duplicate := identities[identity]; duplicate {
			return nil, certifyError(
				"configure provider protocols",
				selected.Name,
				"method set duplicates protocol "+prior,
			)
		}
		identities[identity] = selected.Name
		result[selected.Name] = canonical
	}
	return result, nil
}

func validateSeedIndexes(source []int) error {
	previous := -1
	for _, selected := range source {
		if selected < 0 || selected <= previous {
			return fmt.Errorf("indexes are negative, duplicated, or unordered")
		}
		previous = selected
	}
	return nil
}

func validateProviderRepresentationSeeds(
	source []providerRepresentationSeed,
) (
	[]providerRepresentationSeed,
	map[string]providerRepresentationSeed,
	error,
) {
	result := append([]providerRepresentationSeed(nil), source...)
	for index := range result {
		result[index].SourceTypes = append([]string(nil), result[index].SourceTypes...)
		sort.Strings(result[index].SourceTypes)
		result[index].SourceInterfaces = append(
			[]string(nil),
			result[index].SourceInterfaces...,
		)
		sort.Strings(result[index].SourceInterfaces)
	}
	sort.Slice(result, func(left, right int) bool {
		return representationSeedKey(result[left]) < representationSeedKey(result[right])
	})
	index := make(map[string]providerRepresentationSeed, len(result))
	for _, seed := range result {
		key := representationSeedKey(seed)
		if seed.Specifier == "" || seed.SourcePath == "" || seed.Export == "" ||
			len(seed.SourceTypes) == 0 || len(seed.SourceInterfaces) == 0 {
			return nil, nil, certifyError(
				"configure representations",
				key,
				"representation identity is incomplete",
			)
		}
		if subpath, ok := providerSubpath(seed.Specifier); !ok ||
			!strings.HasPrefix(subpath, "./internal/facets/") ||
			!strings.HasPrefix(seed.SourcePath, "src/internal/facets/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, nil, certifyError(
				"configure representations",
				key,
				"representation module is invalid",
			)
		}
		for typeIndex, identity := range seed.SourceTypes {
			if identity == "" || typeIndex != 0 && identity == seed.SourceTypes[typeIndex-1] {
				return nil, nil, certifyError(
					"configure representations",
					key,
					"source types are empty or duplicated",
				)
			}
		}
		for interfaceIndex, identity := range seed.SourceInterfaces {
			if identity == "" || interfaceIndex != 0 &&
				identity == seed.SourceInterfaces[interfaceIndex-1] {
				return nil, nil, certifyError(
					"configure representations",
					key,
					"source interfaces are empty or duplicated",
				)
			}
		}
		if _, duplicate := index[key]; duplicate {
			return nil, nil, certifyError(
				"configure representations",
				key,
				"representation owner is duplicated",
			)
		}
		index[key] = seed
	}
	return result, index, nil
}

func validateGenericOperationSetSeeds(
	source []genericOperationSetSeed,
) (map[string][]gostdlib.GenericOperationDocument, error) {
	result := make(
		map[string][]gostdlib.GenericOperationDocument,
		len(source),
	)
	for _, seed := range source {
		if seed.SourceIdentity == "" || len(seed.Operations) == 0 {
			return nil, certifyError(
				"configure generic operations",
				seed.SourceIdentity,
				"operation-set identity is incomplete",
			)
		}
		if _, duplicate := result[seed.SourceIdentity]; duplicate {
			return nil, certifyError(
				"configure generic operations",
				seed.SourceIdentity,
				"operation-set owner is duplicated",
			)
		}
		operations, err := gostdlib.CanonicalGenericOperations(seed.Operations)
		if err != nil {
			return nil, certifyError(
				"configure generic operations",
				seed.SourceIdentity,
				err.Error(),
			)
		}
		result[seed.SourceIdentity] = operations
	}
	return result, nil
}

func validateFacetSeeds(
	source []facetSeed,
	representations map[string]providerRepresentationSeed,
) ([]facetSeed, error) {
	result := append([]facetSeed(nil), source...)
	for index := range result {
		result[index].Capabilities = append(
			[]gostdlib.FacetCapability(nil),
			result[index].Capabilities...,
		)
		result[index].GenericTypeArguments = slices.Clone(
			result[index].GenericTypeArguments,
		)
	}
	sort.Slice(result, func(left, right int) bool {
		return facetSeedKey(result[left]) < facetSeedKey(result[right])
	})
	lookups := make(map[string]struct{})
	targets := make(map[string]struct{})
	referencedRepresentations := make(map[string]struct{})
	for key := range representations {
		targets[key] = struct{}{}
	}
	previous := ""
	for index, seed := range result {
		key := facetSeedKey(seed)
		if key == "" || index != 0 && key == previous {
			return nil, certifyError("configure facets", key, "facet identity is invalid or duplicated")
		}
		previous = key
		if !seed.Kind.Valid() || seed.SourceIdentity == "" ||
			seed.Specifier == "" || seed.SourcePath == "" || seed.Export == "" {
			return nil, certifyError("configure facets", key, "facet identity is incomplete")
		}
		if err := validateGenericFacetSeedShape(seed, key); err != nil {
			return nil, err
		}
		if subpath, ok := providerSubpath(seed.Specifier); !ok ||
			!strings.HasPrefix(subpath, "./internal/facets/") ||
			!strings.HasPrefix(seed.SourcePath, "src/internal/facets/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, certifyError("configure facets", key, "facet module is invalid")
		}
		capabilities := make([]string, 0, len(seed.Capabilities))
		for _, capability := range seed.Capabilities {
			capabilities = append(capabilities, string(capability))
		}
		if len(capabilities) == 0 {
			return nil, certifyError("configure facets", key, "capability set is empty")
		}
		if seed.Kind == gostdlib.FacetReflectionTypeOperations &&
			(len(seed.Capabilities) != 1 ||
				seed.Capabilities[0] != gostdlib.FacetCapabilityMetadata ||
				seed.ResultExport == "" ||
				seed.StorageExport != "" ||
				seed.RepresentationExport != "" ||
				seed.Effect != gostdlib.EffectInvalid) {
			return nil, certifyError(
				"configure facets",
				key,
				"reflection-type facet shape is invalid",
			)
		}
		if seed.Kind == gostdlib.FacetDefinedValueOperations &&
			(len(seed.Capabilities) != 2 ||
				seed.Capabilities[0] != gostdlib.FacetCapabilityProject ||
				seed.Capabilities[1] != gostdlib.FacetCapabilityWrap ||
				seed.ResultExport != "" ||
				seed.StorageExport != "" ||
				seed.RepresentationExport != "" ||
				seed.Effect != gostdlib.EffectInvalid) {
			return nil, certifyError(
				"configure facets",
				key,
				"defined-value facet shape is invalid",
			)
		}
		representation := false
		for _, capability := range seed.Capabilities {
			representation = representation ||
				capability == gostdlib.FacetCapabilityRepresentation
		}
		if representation != (seed.RepresentationExport != "") {
			return nil, certifyError(
				"configure facets",
				key,
				"representation target shape is invalid",
			)
		}
		if seed.RepresentationExport != "" {
			representationKey := seed.Specifier + "\x00" + seed.RepresentationExport
			if _, ok := representations[representationKey]; !ok {
				return nil, certifyError(
					"configure facets",
					key,
					"representation is absent from the facet module",
				)
			}
			referencedRepresentations[representationKey] = struct{}{}
		}
		for _, capability := range capabilities {
			lookup := seed.SourceIdentity + "\x00" + string(seed.Kind) + "\x00" + capability
			if _, duplicate := lookups[lookup]; duplicate {
				return nil, certifyError("configure facets", lookup, "capability owner is duplicated")
			}
			lookups[lookup] = struct{}{}
		}
		if seed.Kind != gostdlib.FacetReflectionTypeOperations &&
			seed.ResultExport != "" {
			return nil, certifyError(
				"configure facets",
				key,
				"result export belongs only to a reflection-type facet",
			)
		}
		for _, export := range []string{
			seed.Export,
			seed.ResultExport,
			seed.StorageExport,
		} {
			if export == "" {
				continue
			}
			target := seed.Specifier + "\x00" + export
			if _, duplicate := targets[target]; duplicate {
				return nil, certifyError("configure facets", target, "target owner is duplicated")
			}
			targets[target] = struct{}{}
		}
	}
	for key := range representations {
		if _, referenced := referencedRepresentations[key]; !referenced {
			return nil, certifyError(
				"configure representations",
				key,
				"representation has no facet reference",
			)
		}
	}
	return result, nil
}

func validateGenericFacetSeedShape(seed facetSeed, key string) error {
	if seed.Kind == gostdlib.FacetGenericCallableKernel {
		if len(seed.Capabilities) != 1 ||
			seed.Capabilities[0] != gostdlib.FacetCapabilityKernel ||
			len(seed.GenericTypeArguments) == 0 ||
			seed.Effect != gostdlib.EffectInvalid || seed.ResultExport != "" ||
			seed.StorageExport != "" ||
			seed.RepresentationExport != "" {
			return certifyError(
				"configure facets",
				key,
				"generic-kernel facet shape is invalid",
			)
		}
		seen := make(map[gostdlib.GenericTypeArgumentDocument]struct{})
		for _, argument := range seed.GenericTypeArguments {
			if argument.TypeParameter < 0 || !argument.Facet.Valid() {
				return certifyError(
					"configure facets",
					key,
					"generic-kernel projection is invalid",
				)
			}
			if _, duplicate := seen[argument]; duplicate {
				return certifyError(
					"configure facets",
					key,
					"generic-kernel projection entry is duplicated",
				)
			}
			seen[argument] = struct{}{}
		}
		return nil
	}
	if len(seed.GenericTypeArguments) != 0 {
		return certifyError(
			"configure facets",
			key,
			"generic type arguments belong only to a generic-kernel facet",
		)
	}
	return nil
}

func facetSeedKey(seed facetSeed) string {
	return seed.Specifier + "\x00" + seed.SourceIdentity + "\x00" +
		string(seed.Kind) + "\x00" + seed.Export
}

func representationSeedKey(seed providerRepresentationSeed) string {
	return seed.Specifier + "\x00" + seed.Export
}

func providerCallableProfileSeedKey(seed providerCallableProfileSeed) string {
	return seed.Specifier + "\x00" + seed.SourceIdentity + "\x00" + seed.Export +
		providerProfileInterfaceSeedKey(seed.Interfaces) +
		providerCallableProtocolSeedKey(seed.Protocols)
}

func providerCallableProtocolSeedKey(
	protocols []providerCallableProtocolSeed,
) string {
	var result strings.Builder
	for _, selected := range protocols {
		identity, err := gostdlib.BuildProviderProtocolInterfaceIdentity(
			selected.document,
		)
		result.WriteByte(0)
		result.WriteString(selected.Protocol)
		result.WriteByte(0)
		if err == nil {
			result.WriteString(identity)
		}
		result.WriteByte(0)
		result.WriteString(selected.Export)
		result.WriteByte(0)
		if selected.ValueParameter != nil {
			result.WriteString(fmt.Sprintf("%d", *selected.ValueParameter))
		}
	}
	return result.String()
}

func providerProfileInterfaceSeedKey(
	interfaces []providerCallableProfileInterfaceSeed,
) string {
	var result strings.Builder
	for _, selected := range interfaces {
		result.WriteByte(0)
		result.WriteString(selected.SourceIdentity)
		result.WriteByte(0)
		result.WriteString(selected.Export)
	}
	return result.String()
}
