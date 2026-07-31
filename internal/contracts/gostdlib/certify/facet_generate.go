package certify

import (
	"fmt"
	"go/types"
	"path/filepath"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildFacetModules(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	source goSurface,
	seeds []facetSeed,
) ([]gostdlib.FacetModuleDocument, error) {
	bySpecifier := make(map[string][]facetSeed)
	for _, seed := range seeds {
		bySpecifier[seed.Specifier] = append(bySpecifier[seed.Specifier], seed)
	}
	specifiers := make([]string, 0, len(bySpecifier))
	for specifier := range bySpecifier {
		specifiers = append(specifiers, specifier)
	}
	sort.Strings(specifiers)
	result := make([]gostdlib.FacetModuleDocument, 0, len(specifiers))
	for _, specifier := range specifiers {
		selected := bySpecifier[specifier]
		sourcePath := selected[0].SourcePath
		for _, seed := range selected {
			if seed.SourcePath != sourcePath {
				return nil, certifyError(
					"build facets",
					specifier,
					"one facet module has multiple source files",
				)
			}
		}
		targets, err := project.Exports(filepath.Join(
			config.providerRoot,
			filepath.FromSlash(sourcePath),
		))
		if err != nil {
			return nil, err
		}
		byName := make(map[string]tsgo.ProjectExport, len(targets))
		for _, target := range targets {
			byName[target.Name()] = target
		}
		owned := make(map[string]struct{})
		facets := make([]gostdlib.FacetDocument, 0, len(selected))
		for _, seed := range selected {
			facet, err := buildFacet(source, seed, byName)
			if err != nil {
				return nil, err
			}
			for _, name := range []string{facet.Export, facet.StorageExport} {
				if name != "" {
					owned[name] = struct{}{}
				}
			}
			facets = append(facets, facet)
		}
		for name := range byName {
			if _, ok := owned[name]; !ok {
				return nil, certifyError(
					"build facets",
					specifier+"#"+name,
					"compiler-facet export has no seed owner",
				)
			}
		}
		sort.Slice(facets, func(left, right int) bool {
			leftKey := facets[left].SourceIdentity + "\x00" +
				string(facets[left].Kind) + "\x00" + facets[left].Export
			rightKey := facets[right].SourceIdentity + "\x00" +
				string(facets[right].Kind) + "\x00" + facets[right].Export
			return leftKey < rightKey
		})
		result = append(result, gostdlib.FacetModuleDocument{
			Specifier:  specifier,
			SourcePath: sourcePath,
			Facets:     facets,
		})
	}
	return result, nil
}

func buildFacet(
	source goSurface,
	seed facetSeed,
	targets map[string]tsgo.ProjectExport,
) (gostdlib.FacetDocument, error) {
	evidence, ok := source.objects[seed.SourceIdentity]
	if !ok {
		return gostdlib.FacetDocument{}, certifyError(
			"build facet",
			seed.SourceIdentity,
			"selected-GOROOT declaration is absent",
		)
	}
	switch seed.Kind {
	case gostdlib.FacetNamedStructOperations:
		if _, ok := evidence.object.(*types.TypeName); !ok {
			return gostdlib.FacetDocument{}, certifyError(
				"build facet",
				seed.SourceIdentity,
				"named-struct facet does not own a type",
			)
		}
	case gostdlib.FacetRecoveryCallable,
		gostdlib.FacetGenericCallableProfile:
		if _, ok := evidence.object.(*types.Func); !ok {
			return gostdlib.FacetDocument{}, certifyError(
				"build facet",
				seed.SourceIdentity,
				"callable facet does not own a function or method",
			)
		}
	}
	target, ok := targets[seed.Export]
	if !ok {
		return gostdlib.FacetDocument{}, certifyError(
			"build facet",
			seed.Export,
			"target export is absent",
		)
	}
	if err := validateFacetTarget(seed, target); err != nil {
		return gostdlib.FacetDocument{}, err
	}
	owner, err := singleImplementationOwner(seed.Export, target.ImplementationOwners())
	if err != nil {
		return gostdlib.FacetDocument{}, err
	}
	document := gostdlib.FacetDocument{
		Kind:                seed.Kind,
		SourceIdentity:      seed.SourceIdentity,
		Capabilities:        append([]gostdlib.FacetCapability(nil), seed.Capabilities...),
		ProfileKey:          seed.ProfileKey,
		Export:              seed.Export,
		StorageExport:       seed.StorageExport,
		Effect:              seed.Effect,
		ImplementationOwner: owner,
		TargetFingerprint:   target.Fingerprint(),
	}
	if seed.StorageExport == "" {
		return document, nil
	}
	storage, ok := targets[seed.StorageExport]
	if !ok {
		return gostdlib.FacetDocument{}, certifyError(
			"build facet",
			seed.StorageExport,
			"storage target export is absent",
		)
	}
	document.StorageImplementationOwner, err = singleImplementationOwner(
		seed.StorageExport,
		storage.ImplementationOwners(),
	)
	if err != nil {
		return gostdlib.FacetDocument{}, err
	}
	document.StorageTargetFingerprint = storage.Fingerprint()
	return document, nil
}

func validateFacetTarget(seed facetSeed, target tsgo.ProjectExport) error {
	if seed.Kind != gostdlib.FacetNamedStructOperations {
		return nil
	}
	for _, capability := range seed.Capabilities {
		members, err := facetCapabilityMembers(capability)
		if err != nil {
			return err
		}
		for _, member := range members {
			if _, ok := target.ValueMember(member); !ok {
				return certifyError(
					"build facet",
					seed.Export+"."+member,
					"named-struct capability member is absent",
				)
			}
		}
	}
	return nil
}

func facetCapabilityMembers(
	capability gostdlib.FacetCapability,
) ([]string, error) {
	switch capability {
	case gostdlib.FacetCapabilityMake:
		return []string{"$make"}, nil
	case gostdlib.FacetCapabilityZero:
		return []string{"$zero"}, nil
	case gostdlib.FacetCapabilityCopy:
		return []string{"$copy"}, nil
	case gostdlib.FacetCapabilityEqual:
		return []string{"$equal"}, nil
	case gostdlib.FacetCapabilityHash:
		return []string{"$hash"}, nil
	case gostdlib.FacetCapabilityConvert:
		return []string{"$convert"}, nil
	case gostdlib.FacetCapabilityStorage:
		return []string{"$storageOf", "$fromStorage"}, nil
	case gostdlib.FacetCapabilityAssign:
		return []string{"$assign"}, nil
	default:
		return nil, certifyError(
			"build facet",
			string(capability),
			"named-struct capability is invalid",
		)
	}
}

func singleImplementationOwner(name string, owners []string) (string, error) {
	if len(owners) != 1 {
		return "", certifyError(
			"build facet",
			name,
			fmt.Sprintf("target has %d implementation owners, want one", len(owners)),
		)
	}
	return owners[0], nil
}
