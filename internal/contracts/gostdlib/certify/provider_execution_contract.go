package certify

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type providerInvocationTransportSeed struct {
	SourceIdentity         string                                     `json:"sourceIdentity"`
	Specifier              string                                     `json:"specifier"`
	SourcePath             string                                     `json:"sourcePath"`
	Export                 string                                     `json:"export"`
	Member                 string                                     `json:"member"`
	InputParameters        []int                                      `json:"inputParameters,omitempty"`
	ResultOriginParameters []int                                      `json:"resultOriginParameters,omitempty"`
	State                  *gostdlib.InvocationTransportStateDocument `json:"state,omitempty"`
}

func buildProviderInvocationTransports(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	providerPackage packageDocument,
	seeds []providerInvocationTransportSeed,
	modules []gostdlib.ModuleDocument,
	facetModules []gostdlib.FacetModuleDocument,
) ([]gostdlib.InvocationTransportDocument, error) {
	sourceOwners := invocationTransportSourceOwners(modules, facetModules)
	exportsByPath := make(map[string]map[string]tsgo.ProjectExport)
	result := make([]gostdlib.InvocationTransportDocument, 0, len(seeds))
	for index, seed := range seeds {
		field := fmt.Sprintf("providerInvocationTransports[%d]", index)
		owner, ok := sourceOwners[invocationTransportSourceKey{
			identity:  seed.SourceIdentity,
			specifier: seed.Specifier,
			export:    seed.Export,
		}]
		if !ok || owner.sourcePath != seed.SourcePath {
			return nil, certifyError(
				"build provider invocation transport",
				field,
				"source identity does not exact-join its provider target",
			)
		}
		targets, ok := exportsByPath[seed.SourcePath]
		if !ok {
			exports, err := project.Exports(filepath.Join(
				config.providerRoot,
				filepath.FromSlash(seed.SourcePath),
			))
			if err != nil {
				return nil, err
			}
			targets = make(map[string]tsgo.ProjectExport, len(exports))
			for _, target := range exports {
				targets[target.Name()] = target
			}
			exportsByPath[seed.SourcePath] = targets
		}
		target, ok := targets[seed.Export]
		if !ok {
			return nil, certifyError(
				"build provider invocation transport",
				seed.Specifier+"#"+seed.Export,
				"target export is absent",
			)
		}
		member, ok := target.ValueMember(seed.Member)
		if !ok || !member.Visible() {
			return nil, certifyError(
				"build provider invocation transport",
				seed.Export+"."+seed.Member,
				"public static member is absent",
			)
		}
		parameterCount, err := project.CallableParameterCount(member)
		if err != nil {
			return nil, err
		}
		declarationPath, err := providerDeclarationPath(
			providerPackage,
			seed.Specifier,
		)
		if err != nil {
			return nil, err
		}
		document := gostdlib.InvocationTransportDocument{
			SourceIdentity:         seed.SourceIdentity,
			Specifier:              seed.Specifier,
			SourcePath:             seed.SourcePath,
			DeclarationPath:        declarationPath,
			Export:                 seed.Export,
			Member:                 seed.Member,
			TargetType:             member.TypeString(),
			TargetFingerprint:      member.Fingerprint(),
			InputParameters:        slices.Clone(seed.InputParameters),
			ResultOriginParameters: slices.Clone(seed.ResultOriginParameters),
			State:                  cloneInvocationTransportState(seed.State),
		}
		if err := gostdlib.ValidateInvocationTransportIndexes(
			document,
			parameterCount,
			field,
		); err != nil {
			return nil, certifyError(
				"build provider invocation transport",
				seed.Export+"."+seed.Member,
				err.Error(),
			)
		}
		if owner.targetFingerprint != "" &&
			owner.targetFingerprint != member.Fingerprint() {
			return nil, certifyError(
				"build provider invocation transport",
				seed.Export+"."+seed.Member,
				"binding and invocation target fingerprints disagree",
			)
		}
		result = append(result, document)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].Key() < result[right].Key()
	})
	for index := 1; index < len(result); index++ {
		if result[index-1].Key() == result[index].Key() {
			return nil, certifyError(
				"build provider invocation transports",
				result[index].Specifier,
				"target member is duplicated",
			)
		}
	}
	return result, nil
}

func buildInvocationTransportContract(
	config resolvedConfig,
	transports []gostdlib.InvocationTransportDocument,
) (*gostdlib.InvocationTransportContractDocument, error) {
	if len(transports) == 0 {
		return nil, nil
	}
	root, err := filepath.Rel(filepath.Dir(config.manifestPath), config.providerRoot)
	if err != nil {
		return nil, certifyError(
			"build provider invocation transport",
			config.manifestPath,
			err.Error(),
		)
	}
	return &gostdlib.InvocationTransportContractDocument{
		SchemaVersion:   gostdlib.InvocationTransportSchemaVersion,
		DeclarationRoot: filepath.ToSlash(root),
		Transports:      transports,
	}, nil
}

func providerDeclarationPath(
	document packageDocument,
	specifier string,
) (string, error) {
	subpath, ok := providerSubpath(specifier)
	if !ok {
		return "", certifyError(
			"build provider invocation transport",
			specifier,
			"provider subpath is invalid",
		)
	}
	encoded, ok := document.Exports[subpath]
	if !ok {
		return "", certifyError(
			"build provider invocation transport",
			specifier,
			"package export is absent",
		)
	}
	var selected packageExport
	if err := json.Unmarshal(encoded, &selected); err != nil {
		return "", certifyError(
			"build provider invocation transport",
			specifier,
			err.Error(),
		)
	}
	return strings.TrimPrefix(selected.Types, "./"), nil
}

type invocationTransportSourceKey struct {
	identity  string
	specifier string
	export    string
}

type invocationTransportSourceOwner struct {
	sourcePath        string
	targetFingerprint string
}

func invocationTransportSourceOwners(
	modules []gostdlib.ModuleDocument,
	facetModules []gostdlib.FacetModuleDocument,
) map[invocationTransportSourceKey]invocationTransportSourceOwner {
	result := make(map[invocationTransportSourceKey]invocationTransportSourceOwner)
	for _, module := range modules {
		for _, binding := range module.Bindings {
			result[invocationTransportSourceKey{
				identity:  binding.Identity,
				specifier: module.Specifier,
				export:    binding.Export,
			}] = invocationTransportSourceOwner{
				sourcePath:        module.SourcePath,
				targetFingerprint: binding.TargetFingerprint,
			}
		}
	}
	for _, module := range facetModules {
		for _, facet := range module.Facets {
			result[invocationTransportSourceKey{
				identity:  facet.SourceIdentity,
				specifier: module.Specifier,
				export:    facet.Export,
			}] = invocationTransportSourceOwner{sourcePath: module.SourcePath}
		}
	}
	return result
}

func cloneProviderInvocationTransportSeeds(
	source []providerInvocationTransportSeed,
) []providerInvocationTransportSeed {
	result := make([]providerInvocationTransportSeed, len(source))
	for index, seed := range source {
		result[index] = seed
		result[index].InputParameters = slices.Clone(seed.InputParameters)
		result[index].ResultOriginParameters = slices.Clone(
			seed.ResultOriginParameters,
		)
		result[index].State = cloneInvocationTransportState(seed.State)
	}
	return result
}

func cloneInvocationTransportState(
	source *gostdlib.InvocationTransportStateDocument,
) *gostdlib.InvocationTransportStateDocument {
	if source == nil {
		return nil
	}
	result := *source
	result.WriteParameters = slices.Clone(source.WriteParameters)
	if source.CarrierParameter != nil {
		carrier := *source.CarrierParameter
		result.CarrierParameter = &carrier
	}
	return &result
}

func verifyStatefulExecutionProfilePairs(
	modules []gostdlib.FacetModuleDocument,
) error {
	profiles := make(map[string][]gostdlib.ProviderStatefulProfileDocument)
	for _, module := range modules {
		for _, profile := range module.StatefulProfiles {
			profiles[profile.SourceIdentity] = append(
				profiles[profile.SourceIdentity],
				profile,
			)
		}
	}
	for identity, selected := range profiles {
		for _, profile := range selected {
			if !statefulProfileMaySuspend(profile) {
				continue
			}
			matched := false
			for _, candidate := range selected {
				if statefulProfileMaySuspend(candidate) ||
					!sameStatefulExecutionShape(profile, candidate) {
					continue
				}
				matched = true
				break
			}
			if !matched {
				return certifyError(
					"verify provider stateful execution profiles",
					identity,
					"suspending profile has no exact synchronous sibling",
				)
			}
		}
	}
	return nil
}

func statefulProfileMaySuspend(
	profile gostdlib.ProviderStatefulProfileDocument,
) bool {
	for _, selected := range profile.Interfaces {
		for _, method := range selected.ProviderInterface.Methods {
			if method.Effect.MaySuspend() {
				return true
			}
		}
	}
	for _, method := range profile.Methods {
		if method.Effect.MaySuspend() {
			return true
		}
	}
	return false
}

func sameStatefulExecutionShape(
	left gostdlib.ProviderStatefulProfileDocument,
	right gostdlib.ProviderStatefulProfileDocument,
) bool {
	if left.SourceIdentity != right.SourceIdentity ||
		!slices.Equal(left.TypeArguments, right.TypeArguments) ||
		!slices.Equal(left.Operations, right.Operations) ||
		len(left.Interfaces) != len(right.Interfaces) ||
		len(left.Fields) != len(right.Fields) ||
		len(left.Methods) != len(right.Methods) {
		return false
	}
	for index := range left.Interfaces {
		leftInterface := left.Interfaces[index]
		rightInterface := right.Interfaces[index]
		if leftInterface.SourceIdentity != rightInterface.SourceIdentity ||
			len(leftInterface.ProviderInterface.Methods) !=
				len(rightInterface.ProviderInterface.Methods) {
			return false
		}
		for methodIndex := range leftInterface.ProviderInterface.Methods {
			leftMethod := leftInterface.ProviderInterface.Methods[methodIndex]
			rightMethod := rightInterface.ProviderInterface.Methods[methodIndex]
			if leftMethod.SourceIdentity != rightMethod.SourceIdentity ||
				leftMethod.Kind != rightMethod.Kind ||
				leftMethod.SourceSignature != rightMethod.SourceSignature ||
				leftMethod.ContractSignature != rightMethod.ContractSignature {
				return false
			}
		}
	}
	for index := range left.Fields {
		leftField := left.Fields[index]
		rightField := right.Fields[index]
		if leftField.Member != rightField.Member ||
			leftField.Ordinal != rightField.Ordinal ||
			leftField.Embedded != rightField.Embedded ||
			leftField.SourceSignature != rightField.SourceSignature {
			return false
		}
	}
	for index := range left.Methods {
		leftMethod := left.Methods[index]
		rightMethod := right.Methods[index]
		if leftMethod.SourceIdentity != rightMethod.SourceIdentity ||
			leftMethod.Member != rightMethod.Member ||
			leftMethod.SourceSignature != rightMethod.SourceSignature {
			return false
		}
	}
	return true
}
