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

const facetMapSchemaVersion = 16

type facetMapDocument struct {
	SchemaVersion            int                           `json:"schemaVersion"`
	Representations          []providerRepresentationSeed  `json:"representations,omitempty"`
	DefinedValueIdentities   []string                      `json:"definedValueIdentities,omitempty"`
	Facets                   []facetSeed                   `json:"facets"`
	ProviderCallableProfiles []providerCallableProfileSeed `json:"providerCallableProfiles,omitempty"`
	ProviderProtocols        []providerProtocolSeed        `json:"providerProtocols,omitempty"`
	ProviderStatefulProfiles []providerStatefulProfileSeed `json:"providerStatefulProfiles,omitempty"`
	ProviderInterfaces       []providerInterfaceSeed       `json:"providerInterfaces,omitempty"`
	GenericOperationSets     []genericOperationSetSeed     `json:"genericOperationSets"`
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
	Kind                 gostdlib.FacetKind                     `json:"kind"`
	SourceIdentity       string                                 `json:"sourceIdentity"`
	Capabilities         []gostdlib.FacetCapability             `json:"capabilities,omitempty"`
	GenericTypeArguments []gostdlib.GenericTypeArgumentDocument `json:"genericTypeArguments,omitempty"`
	Specifier            string                                 `json:"specifier"`
	SourcePath           string                                 `json:"sourcePath"`
	Export               string                                 `json:"export"`
	StorageExport        string                                 `json:"storageExport,omitempty"`
	RepresentationExport string                                 `json:"representationExport,omitempty"`
	Effect               gostdlib.EffectKind                    `json:"effect,omitempty"`
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
		seed.Operations = slices.Clone(seed.Operations)
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
		for operationIndex, capability := range seed.Operations {
			if !capability.NamedStructOperation() ||
				capability == gostdlib.FacetCapabilityRepresentation ||
				operationIndex != 0 && capability <= seed.Operations[operationIndex-1] {
				return nil, certifyError(
					"configure provider stateful profiles",
					key,
					"operations are invalid, duplicated, or unordered",
				)
			}
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
