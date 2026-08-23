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
		target, parameterCount, err := buildInvocationTransportTarget(
			config,
			project,
			providerPackage,
			exportsByPath,
			providerInvocationTargetSeed{
				specifier:  seed.Specifier,
				sourcePath: seed.SourcePath,
				access:     gostdlib.InvocationTransportAccessStaticMethod,
				export:     seed.Export,
				member:     seed.Member,
			},
		)
		if err != nil {
			return nil, err
		}
		document := gostdlib.InvocationTransportDocument{
			SourceIdentity:         seed.SourceIdentity,
			Target:                 target,
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
			owner.targetFingerprint != target.TargetFingerprint {
			return nil, certifyError(
				"build provider invocation transport",
				seed.Export+"."+seed.Member,
				"binding and invocation target fingerprints disagree",
			)
		}
		result = append(result, document)
	}
	automatic, err := buildSynchronousGenericInvocationTransports(
		config,
		project,
		providerPackage,
		exportsByPath,
		modules,
		facetModules,
	)
	if err != nil {
		return nil, err
	}
	result = append(result, automatic...)
	sort.Slice(result, func(left, right int) bool {
		return result[left].Key() < result[right].Key()
	})
	for index := 1; index < len(result); index++ {
		if result[index-1].Key() == result[index].Key() {
			return nil, certifyError(
				"build provider invocation transports",
				result[index].Target.Specifier,
				"target member is duplicated",
			)
		}
	}
	return result, nil
}

type providerInvocationTargetSeed struct {
	specifier  string
	sourcePath string
	access     gostdlib.InvocationTransportAccessKind
	export     string
	member     string
}

func buildInvocationTransportTarget(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	providerPackage packageDocument,
	exportsByPath map[string]map[string]tsgo.ProjectExport,
	seed providerInvocationTargetSeed,
) (gostdlib.InvocationTransportTargetDocument, int, error) {
	targets, ok := exportsByPath[seed.sourcePath]
	if !ok {
		exports, err := project.Exports(filepath.Join(
			config.providerRoot,
			filepath.FromSlash(seed.sourcePath),
		))
		if err != nil {
			return gostdlib.InvocationTransportTargetDocument{}, 0, err
		}
		targets = make(map[string]tsgo.ProjectExport, len(exports))
		for _, target := range exports {
			targets[target.Name()] = target
		}
		exportsByPath[seed.sourcePath] = targets
	}
	target, ok := targets[seed.export]
	if !ok {
		return gostdlib.InvocationTransportTargetDocument{}, 0, certifyError(
			"build provider invocation transport",
			seed.specifier+"#"+seed.export,
			"target export is absent",
		)
	}
	declarationPath, err := providerDeclarationPath(
		providerPackage,
		seed.specifier,
	)
	if err != nil {
		return gostdlib.InvocationTransportTargetDocument{}, 0, err
	}
	document := gostdlib.InvocationTransportTargetDocument{
		Specifier:       seed.specifier,
		SourcePath:      seed.sourcePath,
		DeclarationPath: declarationPath,
		Access:          seed.access,
		Export:          seed.export,
		Member:          seed.member,
	}
	switch seed.access {
	case gostdlib.InvocationTransportAccessExport:
		document.TargetType = target.TypeString()
		document.TargetFingerprint = target.Fingerprint()
		parameterCount, countErr := project.CallableParameterCount(target)
		return document, parameterCount, countErr
	case gostdlib.InvocationTransportAccessStaticMethod:
		member, found := target.ValueMember(seed.member)
		if !found || !member.Visible() {
			return gostdlib.InvocationTransportTargetDocument{}, 0, certifyError(
				"build provider invocation transport",
				seed.export+"."+seed.member,
				"public static member is absent",
			)
		}
		document.TargetType = member.TypeString()
		document.TargetFingerprint = member.Fingerprint()
		parameterCount, countErr := project.CallableParameterCount(member)
		return document, parameterCount, countErr
	default:
		return gostdlib.InvocationTransportTargetDocument{}, 0, certifyError(
			"build provider invocation transport",
			seed.specifier,
			"target access is invalid",
		)
	}
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

type genericKernelInvocationTarget struct {
	module gostdlib.FacetModuleDocument
	facet  gostdlib.FacetDocument
}

func buildSynchronousGenericInvocationTransports(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	providerPackage packageDocument,
	exportsByPath map[string]map[string]tsgo.ProjectExport,
	modules []gostdlib.ModuleDocument,
	facetModules []gostdlib.FacetModuleDocument,
) ([]gostdlib.InvocationTransportDocument, error) {
	bindings := make(map[string]gostdlib.BindingDocument)
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if _, exists := bindings[binding.Identity]; exists {
				return nil, certifyError(
					"build synchronous generic invocation transport",
					binding.Identity,
					"public binding is duplicated",
				)
			}
			bindings[binding.Identity] = binding
		}
	}
	canonical, synchronous, err := indexGenericKernelInvocationTargets(
		facetModules,
	)
	if err != nil {
		return nil, err
	}
	identities := make([]string, 0, len(synchronous))
	for identity := range synchronous {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	result := make([]gostdlib.InvocationTransportDocument, 0, len(identities))
	for _, identity := range identities {
		base, ok := canonical[identity]
		if !ok {
			return nil, certifyError(
				"build synchronous generic invocation transport",
				identity,
				"canonical kernel is absent",
			)
		}
		narrowed := synchronous[identity]
		binding, ok := bindings[identity]
		if !ok {
			return nil, certifyError(
				"build synchronous generic invocation transport",
				identity,
				"public binding is absent",
			)
		}
		if base.module.Specifier != narrowed.module.Specifier ||
			base.module.SourcePath != narrowed.module.SourcePath {
			return nil, certifyError(
				"build synchronous generic invocation transport",
				identity,
				"kernel pair changes provider module",
			)
		}
		parameters := genericKernelTargetCallableParameters(
			len(binding.GenericOperations),
			base.facet.CallableParameters,
		)
		if len(parameters) == 0 {
			return nil, certifyError(
				"build synchronous generic invocation transport",
				identity,
				"canonical kernel has no callable parameter",
			)
		}
		current, currentArity, err := buildInvocationTransportTarget(
			config,
			project,
			providerPackage,
			exportsByPath,
			providerInvocationTargetSeed{
				specifier:  base.module.Specifier,
				sourcePath: base.module.SourcePath,
				access:     gostdlib.InvocationTransportAccessExport,
				export:     base.facet.Export,
			},
		)
		if err != nil {
			return nil, err
		}
		replacement, replacementArity, err := buildInvocationTransportTarget(
			config,
			project,
			providerPackage,
			exportsByPath,
			providerInvocationTargetSeed{
				specifier:  narrowed.module.Specifier,
				sourcePath: narrowed.module.SourcePath,
				access:     gostdlib.InvocationTransportAccessExport,
				export:     narrowed.facet.Export,
			},
		)
		if err != nil {
			return nil, err
		}
		if current.TargetFingerprint != base.facet.TargetFingerprint ||
			replacement.TargetFingerprint != narrowed.facet.TargetFingerprint {
			return nil, certifyError(
				"build synchronous generic invocation transport",
				identity,
				"kernel target fingerprint changed after facet certification",
			)
		}
		document := gostdlib.InvocationTransportDocument{
			SourceIdentity:  identity,
			Target:          current,
			InputParameters: slices.Clone(parameters),
			Conditional: &gostdlib.InvocationTransportConditionalDocument{
				CallableParameters: slices.Clone(parameters),
				Replacement:        replacement,
			},
		}
		if err := gostdlib.ValidateInvocationTransportIndexes(
			document,
			currentArity,
			fmt.Sprintf("synchronousGenericInvocationTransport[%q]", identity),
		); err != nil {
			return nil, certifyError(
				"build synchronous generic invocation transport",
				identity,
				err.Error(),
			)
		}
		if err := gostdlib.ValidateInvocationTransportIndexes(
			document,
			replacementArity,
			fmt.Sprintf("synchronousGenericInvocationReplacement[%q]", identity),
		); err != nil {
			return nil, certifyError(
				"build synchronous generic invocation transport",
				identity,
				err.Error(),
			)
		}
		result = append(result, document)
	}
	return result, nil
}

func indexGenericKernelInvocationTargets(
	modules []gostdlib.FacetModuleDocument,
) (
	map[string]genericKernelInvocationTarget,
	map[string]genericKernelInvocationTarget,
	error,
) {
	canonical := make(map[string]genericKernelInvocationTarget)
	synchronous := make(map[string]genericKernelInvocationTarget)
	for _, module := range modules {
		for _, facet := range module.Facets {
			if facet.Kind != gostdlib.FacetGenericCallableKernel ||
				len(facet.Capabilities) != 1 {
				continue
			}
			var selected map[string]genericKernelInvocationTarget
			switch facet.Capabilities[0] {
			case gostdlib.FacetCapabilityKernel:
				selected = canonical
			case gostdlib.FacetCapabilitySynchronousKernel:
				selected = synchronous
			default:
				continue
			}
			if _, exists := selected[facet.SourceIdentity]; exists {
				return nil, nil, certifyError(
					"build synchronous generic invocation transport",
					facet.SourceIdentity,
					"kernel capability is duplicated",
				)
			}
			selected[facet.SourceIdentity] = genericKernelInvocationTarget{
				module: module,
				facet:  facet,
			}
		}
	}
	return canonical, synchronous, nil
}

func genericKernelTargetCallableParameters(
	capabilityParameters int,
	source []gostdlib.ProviderCallableParameterDocument,
) []int {
	result := make([]int, len(source))
	for index, parameter := range source {
		result[index] = capabilityParameters + parameter.Parameter
	}
	return result
}
