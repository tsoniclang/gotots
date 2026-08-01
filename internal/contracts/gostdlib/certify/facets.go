package certify

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

const facetMapSchemaVersion = 9

type facetMapDocument struct {
	SchemaVersion              int                             `json:"schemaVersion"`
	Representations            []providerRepresentationSeed    `json:"representations,omitempty"`
	DefinedValueIdentities     []string                        `json:"definedValueIdentities,omitempty"`
	Facets                     []facetSeed                     `json:"facets"`
	ProviderCallableProfiles   []providerCallableProfileSeed   `json:"providerCallableProfiles,omitempty"`
	GenericCallableProjections []genericCallableProjectionSeed `json:"genericCallableProjections"`
	GenericOperationSets       []genericOperationSetSeed       `json:"genericOperationSets"`
}

type genericCallableProjectionSeed struct {
	SourceIdentity string                                 `json:"sourceIdentity"`
	TypeArguments  []gostdlib.GenericTypeArgumentDocument `json:"typeArguments"`
}

type providerRepresentationSeed struct {
	Specifier        string   `json:"specifier"`
	SourcePath       string   `json:"sourcePath"`
	Export           string   `json:"export"`
	SourceTypes      []string `json:"sourceTypes"`
	SourceInterfaces []string `json:"sourceInterfaces"`
}

type genericOperationSetSeed struct {
	SourceIdentity string                              `json:"sourceIdentity"`
	Operations     []gostdlib.GenericOperationDocument `json:"operations"`
}

type facetSeed struct {
	Kind                 gostdlib.FacetKind         `json:"kind"`
	SourceIdentity       string                     `json:"sourceIdentity"`
	Capabilities         []gostdlib.FacetCapability `json:"capabilities,omitempty"`
	ProfileKey           string                     `json:"profileKey,omitempty"`
	Specifier            string                     `json:"specifier"`
	SourcePath           string                     `json:"sourcePath"`
	Export               string                     `json:"export"`
	StorageExport        string                     `json:"storageExport,omitempty"`
	RepresentationExport string                     `json:"representationExport,omitempty"`
	Effect               gostdlib.EffectKind        `json:"effect,omitempty"`
}

type providerCallableProfileSeed struct {
	SourceIdentity      string                                 `json:"sourceIdentity"`
	Specifier           string                                 `json:"specifier"`
	SourcePath          string                                 `json:"sourcePath"`
	Export              string                                 `json:"export"`
	Receiver            bool                                   `json:"receiver,omitempty"`
	CanonicalParameters []int                                  `json:"canonicalParameters"`
	CanonicalResults    []int                                  `json:"canonicalResults,omitempty"`
	GuardInterfaces     []string                               `json:"guardInterfaces,omitempty"`
	Interfaces          []providerCallableProfileInterfaceSeed `json:"interfaces"`
}

type providerCallableProfileInterfaceSeed struct {
	SourceIdentity string `json:"sourceIdentity"`
	Export         string `json:"export"`
}

func readFacetSeeds(
	sourcePath string,
) (
	[]facetSeed,
	[]providerRepresentationSeed,
	[]providerCallableProfileSeed,
	map[string]struct{},
	map[string][]gostdlib.GenericTypeArgumentDocument,
	map[string][]gostdlib.GenericOperationDocument,
	error,
) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, certifyError("read facet map", sourcePath, err.Error())
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document facetMapDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, nil, nil, nil, nil, certifyError("read facet map", sourcePath, err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, nil, nil, nil, nil, nil, certifyError("read facet map", sourcePath, err.Error())
	}
	if document.SchemaVersion != facetMapSchemaVersion {
		return nil, nil, nil, nil, nil, nil, certifyError("read facet map", sourcePath, "schema is unsupported")
	}
	representations, representationIndex, err :=
		validateProviderRepresentationSeeds(document.Representations)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	profiles, err := validateProviderCallableProfileSeeds(
		document.ProviderCallableProfiles,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	identities, err := validateDefinedValueIdentities(document.DefinedValueIdentities)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	facets, err := validateFacetSeeds(document.Facets, representationIndex)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	projections, err := validateGenericCallableProjectionSeeds(
		document.GenericCallableProjections,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	operations, err := validateGenericOperationSetSeeds(
		document.GenericOperationSets,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	return facets, representations, profiles, identities, projections, operations, nil
}

func validateProviderCallableProfileSeeds(
	source []providerCallableProfileSeed,
) ([]providerCallableProfileSeed, error) {
	result := append([]providerCallableProfileSeed(nil), source...)
	sort.Slice(result, func(left, right int) bool {
		return providerCallableProfileSeedKey(result[left]) <
			providerCallableProfileSeedKey(result[right])
	})
	previous := ""
	for index := range result {
		seed := &result[index]
		seed.CanonicalParameters = slices.Clone(seed.CanonicalParameters)
		seed.CanonicalResults = slices.Clone(seed.CanonicalResults)
		seed.GuardInterfaces = slices.Clone(seed.GuardInterfaces)
		seed.Interfaces = slices.Clone(seed.Interfaces)
		key := providerCallableProfileSeedKey(*seed)
		if key == "" || key == previous || seed.SourceIdentity == "" ||
			seed.Specifier == "" || seed.SourcePath == "" || seed.Export == "" ||
			len(seed.CanonicalParameters) == 0 || len(seed.Interfaces) == 0 {
			return nil, certifyError(
				"configure provider callable profiles",
				key,
				"profile identity or shape is incomplete or duplicated",
			)
		}
		previous = key
		if subpath, ok := providerSubpath(seed.Specifier); !ok ||
			!strings.HasPrefix(subpath, "./internal/facets/") ||
			!strings.HasPrefix(seed.SourcePath, "src/internal/facets/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, certifyError(
				"configure provider callable profiles",
				key,
				"profile module is invalid",
			)
		}
		if err := validateSeedIndexes(seed.CanonicalParameters); err != nil {
			return nil, certifyError(
				"configure provider callable profiles",
				key,
				"canonical parameter roots are invalid",
			)
		}
		if err := validateSeedIndexes(seed.CanonicalResults); err != nil {
			return nil, certifyError(
				"configure provider callable profiles",
				key,
				"canonical result roots are invalid",
			)
		}
		interfaceIdentities := make(map[string]struct{}, len(seed.Interfaces))
		interfaceExports := make(map[string]struct{}, len(seed.Interfaces))
		previousInterface := ""
		for _, selected := range seed.Interfaces {
			if selected.SourceIdentity == "" || selected.Export == "" ||
				selected.SourceIdentity <= previousInterface {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"profile interfaces are empty, duplicated, or unordered",
				)
			}
			previousInterface = selected.SourceIdentity
			if _, duplicate := interfaceExports[selected.Export]; duplicate {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"profile interface export is duplicated",
				)
			}
			interfaceExports[selected.Export] = struct{}{}
			interfaceIdentities[selected.SourceIdentity] = struct{}{}
		}
		seenGuards := make(map[string]struct{}, len(seed.GuardInterfaces))
		for _, identity := range seed.GuardInterfaces {
			if _, ok := interfaceIdentities[identity]; !ok {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"guard has no profile-interface owner",
				)
			}
			if _, duplicate := seenGuards[identity]; duplicate {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"guard interface is duplicated",
				)
			}
			seenGuards[identity] = struct{}{}
		}
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

func validateGenericCallableProjectionSeeds(
	source []genericCallableProjectionSeed,
) (map[string][]gostdlib.GenericTypeArgumentDocument, error) {
	result := make(
		map[string][]gostdlib.GenericTypeArgumentDocument,
		len(source),
	)
	previous := ""
	for _, seed := range source {
		if seed.SourceIdentity == "" ||
			previous >= seed.SourceIdentity ||
			len(seed.TypeArguments) == 0 {
			return nil, certifyError(
				"configure generic callable projections",
				seed.SourceIdentity,
				"projection identity is incomplete or unordered",
			)
		}
		previous = seed.SourceIdentity
		arguments := slices.Clone(seed.TypeArguments)
		seen := make(map[gostdlib.GenericTypeArgumentDocument]struct{})
		for _, argument := range arguments {
			if argument.TypeParameter < 0 || !argument.Facet.Valid() {
				return nil, certifyError(
					"configure generic callable projections",
					seed.SourceIdentity,
					"source parameter or representation facet is invalid",
				)
			}
			if _, duplicate := seen[argument]; duplicate {
				return nil, certifyError(
					"configure generic callable projections",
					seed.SourceIdentity,
					"target projection entry is duplicated",
				)
			}
			seen[argument] = struct{}{}
		}
		result[seed.SourceIdentity] = arguments
	}
	return result, nil
}

func validateDefinedValueIdentities(source []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(source))
	for index, identity := range source {
		if identity == "" {
			return nil, certifyError(
				"configure defined values",
				identity,
				"identity is empty",
			)
		}
		if _, duplicate := result[identity]; duplicate {
			return nil, certifyError(
				"configure defined values",
				identity,
				"identity owner is duplicated",
			)
		}
		if index != 0 && identity < source[index-1] {
			return nil, certifyError(
				"configure defined values",
				identity,
				"identities are not ordered",
			)
		}
		result[identity] = struct{}{}
	}
	return result, nil
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
		if subpath, ok := providerSubpath(seed.Specifier); !ok ||
			!strings.HasPrefix(subpath, "./internal/facets/") ||
			!strings.HasPrefix(seed.SourcePath, "src/internal/facets/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, certifyError("configure facets", key, "facet module is invalid")
		}
		capabilities := make([]string, 0, len(seed.Capabilities)+1)
		for _, capability := range seed.Capabilities {
			capabilities = append(capabilities, string(capability))
		}
		if seed.ProfileKey != "" {
			capabilities = append(capabilities, seed.ProfileKey)
		}
		if len(capabilities) == 0 {
			return nil, certifyError("configure facets", key, "capability set is empty")
		}
		if seed.Kind == gostdlib.FacetDefinedValueOperations &&
			(len(seed.Capabilities) != 2 ||
				seed.Capabilities[0] != gostdlib.FacetCapabilityProject ||
				seed.Capabilities[1] != gostdlib.FacetCapabilityWrap ||
				seed.ProfileKey != "" || seed.StorageExport != "" ||
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
		for _, export := range []string{seed.Export, seed.StorageExport} {
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

func facetSeedKey(seed facetSeed) string {
	return seed.Specifier + "\x00" + seed.SourceIdentity + "\x00" +
		string(seed.Kind) + "\x00" + seed.Export
}

func representationSeedKey(seed providerRepresentationSeed) string {
	return seed.Specifier + "\x00" + seed.Export
}

func providerCallableProfileSeedKey(seed providerCallableProfileSeed) string {
	return seed.Specifier + "\x00" + seed.SourceIdentity + "\x00" + seed.Export
}
