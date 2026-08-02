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

const facetMapSchemaVersion = 14

type facetMapDocument struct {
	SchemaVersion              int                             `json:"schemaVersion"`
	Representations            []providerRepresentationSeed    `json:"representations,omitempty"`
	DefinedValueIdentities     []string                        `json:"definedValueIdentities,omitempty"`
	Facets                     []facetSeed                     `json:"facets"`
	ProviderCallableProfiles   []providerCallableProfileSeed   `json:"providerCallableProfiles,omitempty"`
	ProviderProtocols          []providerProtocolSeed          `json:"providerProtocols,omitempty"`
	ProviderStatefulProfiles   []providerStatefulProfileSeed   `json:"providerStatefulProfiles,omitempty"`
	ProviderInterfaces         []providerInterfaceSeed         `json:"providerInterfaces,omitempty"`
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
	SourceIdentity              string                                 `json:"sourceIdentity"`
	Specifier                   string                                 `json:"specifier"`
	SourcePath                  string                                 `json:"sourcePath"`
	Export                      string                                 `json:"export"`
	Required                    bool                                   `json:"required,omitempty"`
	Receiver                    bool                                   `json:"receiver,omitempty"`
	CanonicalParameters         []int                                  `json:"canonicalParameters"`
	CanonicalResults            []int                                  `json:"canonicalResults,omitempty"`
	CanonicalValues             []string                               `json:"canonicalValues,omitempty"`
	CanonicalTypeArguments      []string                               `json:"canonicalTypeArguments,omitempty"`
	GuardInterfaces             []string                               `json:"guardInterfaces,omitempty"`
	ContractInterfaces          []string                               `json:"contractInterfaces,omitempty"`
	FromProviderInterfaces      []string                               `json:"fromProviderInterfaces,omitempty"`
	ImplementedResultInterfaces []string                               `json:"implementedResultInterfaces,omitempty"`
	Interfaces                  []providerCallableProfileInterfaceSeed `json:"interfaces"`
	Protocols                   []providerCallableProtocolSeed         `json:"protocols,omitempty"`
}

type providerCallableProfileInterfaceSeed struct {
	SourceIdentity string `json:"sourceIdentity"`
	Export         string `json:"export"`
}

type providerCallableProtocolSeed struct {
	Protocol       string `json:"protocol"`
	Export         string `json:"export"`
	ValueParameter *int   `json:"valueParameter"`
	document       gostdlib.ProviderProtocolInterfaceDocument
}

type providerProtocolSeed struct {
	Name     string                                     `json:"name"`
	Protocol gostdlib.ProviderProtocolInterfaceDocument `json:"protocol"`
}

type providerStatefulProfileSeed struct {
	SourceIdentity string                                 `json:"sourceIdentity"`
	Specifier      string                                 `json:"specifier"`
	SourcePath     string                                 `json:"sourcePath"`
	Export         string                                 `json:"export"`
	Interfaces     []providerCallableProfileInterfaceSeed `json:"interfaces"`
	TypeArguments  []string                               `json:"typeArguments"`
}

type providerInterfaceSeed struct {
	SourceIdentity string `json:"sourceIdentity"`
	Specifier      string `json:"specifier"`
	SourcePath     string `json:"sourcePath"`
	Export         string `json:"export"`
}

type facetSeedSet struct {
	facets                 []facetSeed
	representations        []providerRepresentationSeed
	callableProfiles       []providerCallableProfileSeed
	statefulProfiles       []providerStatefulProfileSeed
	definedValueIdentities map[string]struct{}
	genericProjections     map[string][]gostdlib.GenericTypeArgumentDocument
	genericOperations      map[string][]gostdlib.GenericOperationDocument
	providerInterfaces     []providerInterfaceSeed
}

func readFacetSeeds(sourcePath string) (facetSeedSet, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return facetSeedSet{}, certifyError("read facet map", sourcePath, err.Error())
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document facetMapDocument
	if err := decoder.Decode(&document); err != nil {
		return facetSeedSet{}, certifyError("read facet map", sourcePath, err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return facetSeedSet{}, certifyError("read facet map", sourcePath, err.Error())
	}
	if document.SchemaVersion != facetMapSchemaVersion {
		return facetSeedSet{}, certifyError("read facet map", sourcePath, "schema is unsupported")
	}
	representations, representationIndex, err :=
		validateProviderRepresentationSeeds(document.Representations)
	if err != nil {
		return facetSeedSet{}, err
	}
	protocols, err := validateProviderProtocolSeeds(document.ProviderProtocols)
	if err != nil {
		return facetSeedSet{}, err
	}
	profiles, err := validateProviderCallableProfileSeeds(
		document.ProviderCallableProfiles,
		protocols,
	)
	if err != nil {
		return facetSeedSet{}, err
	}
	statefulProfiles, err := validateProviderStatefulProfileSeeds(
		document.ProviderStatefulProfiles,
	)
	if err != nil {
		return facetSeedSet{}, err
	}
	providerInterfaces, err := validateProviderInterfaceSeeds(
		document.ProviderInterfaces,
	)
	if err != nil {
		return facetSeedSet{}, err
	}
	identities, err := validateDefinedValueIdentities(document.DefinedValueIdentities)
	if err != nil {
		return facetSeedSet{}, err
	}
	facets, err := validateFacetSeeds(document.Facets, representationIndex)
	if err != nil {
		return facetSeedSet{}, err
	}
	projections, err := validateGenericCallableProjectionSeeds(
		document.GenericCallableProjections,
	)
	if err != nil {
		return facetSeedSet{}, err
	}
	operations, err := validateGenericOperationSetSeeds(
		document.GenericOperationSets,
	)
	if err != nil {
		return facetSeedSet{}, err
	}
	return facetSeedSet{
		facets:                 facets,
		representations:        representations,
		callableProfiles:       profiles,
		statefulProfiles:       statefulProfiles,
		definedValueIdentities: identities,
		genericProjections:     projections,
		genericOperations:      operations,
		providerInterfaces:     providerInterfaces,
	}, nil
}

func validateProviderStatefulProfileSeeds(
	source []providerStatefulProfileSeed,
) ([]providerStatefulProfileSeed, error) {
	result := append([]providerStatefulProfileSeed(nil), source...)
	sort.Slice(result, func(left, right int) bool {
		return providerStatefulProfileSeedKey(result[left]) <
			providerStatefulProfileSeedKey(result[right])
	})
	previous := ""
	for index := range result {
		seed := &result[index]
		seed.Interfaces = slices.Clone(seed.Interfaces)
		seed.TypeArguments = slices.Clone(seed.TypeArguments)
		key := providerStatefulProfileSeedKey(*seed)
		if key == "" || key == previous || seed.SourceIdentity == "" ||
			seed.Specifier == "" || seed.SourcePath == "" || seed.Export == "" ||
			len(seed.Interfaces) == 0 || len(seed.TypeArguments) == 0 {
			return nil, certifyError(
				"configure provider stateful profiles",
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
				"configure provider stateful profiles",
				key,
				"profile module is invalid",
			)
		}
		previousInterface := ""
		interfaceIdentities := make(map[string]struct{}, len(seed.Interfaces))
		seenExports := make(map[string]struct{}, len(seed.Interfaces))
		for _, selected := range seed.Interfaces {
			if selected.SourceIdentity == "" || selected.Export == "" ||
				selected.SourceIdentity <= previousInterface {
				return nil, certifyError(
					"configure provider stateful profiles",
					key,
					"profile interfaces are empty, duplicated, or unordered",
				)
			}
			previousInterface = selected.SourceIdentity
			interfaceIdentities[selected.SourceIdentity] = struct{}{}
			if _, duplicate := seenExports[selected.Export]; duplicate {
				return nil, certifyError(
					"configure provider stateful profiles",
					key,
					"profile interface export is duplicated",
				)
			}
			seenExports[selected.Export] = struct{}{}
		}
		seenTypeArguments := make(map[string]struct{}, len(seed.TypeArguments))
		for _, identity := range seed.TypeArguments {
			if _, ok := interfaceIdentities[identity]; !ok {
				return nil, certifyError(
					"configure provider stateful profiles",
					key,
					"type argument has no retained-interface owner",
				)
			}
			if _, duplicate := seenTypeArguments[identity]; duplicate {
				return nil, certifyError(
					"configure provider stateful profiles",
					key,
					"type argument is duplicated",
				)
			}
			seenTypeArguments[identity] = struct{}{}
		}
		if len(seenTypeArguments) != len(interfaceIdentities) {
			return nil, certifyError(
				"configure provider stateful profiles",
				key,
				"type arguments do not exact-join retained interfaces",
			)
		}
	}
	return result, nil
}

func providerStatefulProfileSeedKey(seed providerStatefulProfileSeed) string {
	return seed.SourceIdentity + "\x00" + seed.Specifier + "\x00" + seed.Export +
		providerProfileInterfaceSeedKey(seed.Interfaces)
}

func validateProviderInterfaceSeeds(
	source []providerInterfaceSeed,
) ([]providerInterfaceSeed, error) {
	result := append([]providerInterfaceSeed(nil), source...)
	sort.Slice(result, func(left, right int) bool {
		return result[left].SourceIdentity < result[right].SourceIdentity
	})
	previous := ""
	for _, seed := range result {
		if seed.SourceIdentity == "" || seed.SourceIdentity <= previous ||
			seed.Specifier == "" || seed.SourcePath == "" || seed.Export == "" {
			return nil, certifyError(
				"configure provider interfaces",
				seed.SourceIdentity,
				"provider-interface identity or target is incomplete or duplicated",
			)
		}
		previous = seed.SourceIdentity
		if subpath, ok := providerSubpath(seed.Specifier); !ok ||
			!strings.HasPrefix(subpath, "./internal/facets/") ||
			!strings.HasPrefix(seed.SourcePath, "src/internal/facets/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, certifyError(
				"configure provider interfaces",
				seed.SourceIdentity,
				"provider-interface module is invalid",
			)
		}
	}
	return result, nil
}

func validateProviderCallableProfileSeeds(
	source []providerCallableProfileSeed,
	protocols map[string]gostdlib.ProviderProtocolInterfaceDocument,
) ([]providerCallableProfileSeed, error) {
	result := make([]providerCallableProfileSeed, len(source))
	for index, selected := range source {
		result[index] = selected
		result[index].CanonicalParameters = slices.Clone(selected.CanonicalParameters)
		result[index].CanonicalResults = slices.Clone(selected.CanonicalResults)
		result[index].CanonicalValues = slices.Clone(selected.CanonicalValues)
		result[index].CanonicalTypeArguments = slices.Clone(selected.CanonicalTypeArguments)
		result[index].GuardInterfaces = slices.Clone(selected.GuardInterfaces)
		result[index].ContractInterfaces = slices.Clone(selected.ContractInterfaces)
		result[index].FromProviderInterfaces = slices.Clone(selected.FromProviderInterfaces)
		result[index].ImplementedResultInterfaces = slices.Clone(
			selected.ImplementedResultInterfaces,
		)
		result[index].Interfaces = slices.Clone(selected.Interfaces)
		result[index].Protocols = make(
			[]providerCallableProtocolSeed,
			len(selected.Protocols),
		)
		for protocolIndex, protocol := range selected.Protocols {
			canonical, ok := protocols[protocol.Protocol]
			if !ok || protocol.Export == "" {
				return nil, certifyError(
					"configure provider callable profiles",
					selected.SourceIdentity,
					"protocol reference or export is invalid",
				)
			}
			result[index].Protocols[protocolIndex] = providerCallableProtocolSeed{
				Protocol:       protocol.Protocol,
				Export:         protocol.Export,
				ValueParameter: cloneIndex(protocol.ValueParameter),
				document:       canonical,
			}
		}
	}
	sort.Slice(result, func(left, right int) bool {
		return providerCallableProfileSeedKey(result[left]) <
			providerCallableProfileSeedKey(result[right])
	})
	previous := ""
	for index := range result {
		seed := &result[index]
		key := providerCallableProfileSeedKey(*seed)
		if key == "" || key == previous || seed.SourceIdentity == "" ||
			seed.Specifier == "" || seed.SourcePath == "" || seed.Export == "" ||
			len(seed.CanonicalParameters) == 0 ||
			len(seed.Interfaces)+len(seed.Protocols) == 0 {
			return nil, certifyError(
				"configure provider callable profiles",
				key,
				"profile identity or shape is incomplete or duplicated",
			)
		}
		if seed.Required != (len(seed.Protocols) != 0) {
			return nil, certifyError(
				"configure provider callable profiles",
				key,
				"required selection and semantic protocols disagree",
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
		for valueIndex, identity := range seed.CanonicalValues {
			if identity == "" || valueIndex != 0 && identity <= seed.CanonicalValues[valueIndex-1] {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"canonical value identities are empty, duplicated, or unordered",
				)
			}
		}
		interfaceIdentities := make(
			map[string]struct{},
			len(seed.Interfaces)+len(seed.Protocols),
		)
		interfaceExports := make(
			map[string]struct{},
			len(seed.Interfaces)+len(seed.Protocols),
		)
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
		for _, protocol := range seed.Protocols {
			identity, err := gostdlib.BuildProviderProtocolInterfaceIdentity(
				protocol.document,
			)
			if err != nil || protocol.Export == "" ||
				protocol.ValueParameter == nil || *protocol.ValueParameter < 0 {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"protocol identity or export is invalid",
				)
			}
			if _, duplicate := interfaceIdentities[identity]; duplicate {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"protocol interface is duplicated",
				)
			}
			if _, duplicate := interfaceExports[protocol.Export]; duplicate {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"protocol export is duplicated",
				)
			}
			interfaceIdentities[identity] = struct{}{}
			interfaceExports[protocol.Export] = struct{}{}
			if _, found := slices.BinarySearch(
				seed.CanonicalParameters,
				*protocol.ValueParameter,
			); !found {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"protocol value parameter is not a canonical root",
				)
			}
			references, err := gostdlib.ProviderProtocolCallableParameters(
				protocol.document,
			)
			if err != nil {
				return nil, err
			}
			for _, parameter := range references {
				if _, found := slices.BinarySearch(seed.CanonicalParameters, parameter); !found {
					return nil, certifyError(
						"configure provider callable profiles",
						key,
						"protocol callable parameter is not a canonical root",
					)
				}
			}
		}
		seenTypeArguments := make(map[string]struct{}, len(seed.CanonicalTypeArguments))
		for _, identity := range seed.CanonicalTypeArguments {
			if _, ok := interfaceIdentities[identity]; !ok {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"canonical type argument has no profile-interface owner",
				)
			}
			if _, duplicate := seenTypeArguments[identity]; duplicate {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"canonical type argument is duplicated",
				)
			}
			seenTypeArguments[identity] = struct{}{}
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
		seenContracts := make(map[string]struct{}, len(seed.ContractInterfaces))
		for _, identity := range seed.ContractInterfaces {
			if _, ok := interfaceIdentities[identity]; !ok {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"contract has no profile-interface owner",
				)
			}
			if _, duplicate := seenContracts[identity]; duplicate {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"contract interface is duplicated",
				)
			}
			seenContracts[identity] = struct{}{}
		}
		previousBridge := ""
		for _, identity := range seed.FromProviderInterfaces {
			if identity <= previousBridge {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"from-provider interfaces are empty, duplicated, or unordered",
				)
			}
			previousBridge = identity
			if _, ok := interfaceIdentities[identity]; !ok {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"from-provider interface has no profile-interface owner",
				)
			}
		}
		previousImplemented := ""
		for _, identity := range seed.ImplementedResultInterfaces {
			if identity <= previousImplemented {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"implemented result interfaces are empty, duplicated, or unordered",
				)
			}
			previousImplemented = identity
			if _, ok := seenContracts[identity]; !ok {
				return nil, certifyError(
					"configure provider callable profiles",
					key,
					"implemented result interface is not a contract interface",
				)
			}
		}
	}
	return result, nil
}

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
