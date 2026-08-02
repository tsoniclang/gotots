package certify

import (
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
	representationSeeds []providerRepresentationSeed,
	callableProfileSeeds []providerCallableProfileSeed,
	statefulProfileSeeds []providerStatefulProfileSeed,
	providerInterfaceSeeds []providerInterfaceSeed,
	modules []gostdlib.ModuleDocument,
	genericOperations map[string][]gostdlib.GenericOperationDocument,
	effectMarker tsgo.ProjectExport,
	selectedToolchain toolchain,
) ([]gostdlib.FacetModuleDocument, error) {
	interfaceTargets, err := providerInterfaceTargets(
		config,
		project,
		representationSeeds,
		modules,
	)
	if err != nil {
		return nil, err
	}
	bindingDocuments := make(map[string]gostdlib.BindingDocument)
	for _, module := range modules {
		for _, binding := range module.Bindings {
			bindingDocuments[binding.Identity] = binding
		}
	}
	bySpecifier := make(map[string][]facetSeed)
	for _, seed := range seeds {
		bySpecifier[seed.Specifier] = append(bySpecifier[seed.Specifier], seed)
	}
	representationsBySpecifier := make(
		map[string][]providerRepresentationSeed,
	)
	for _, seed := range representationSeeds {
		representationsBySpecifier[seed.Specifier] = append(
			representationsBySpecifier[seed.Specifier],
			seed,
		)
	}
	profilesBySpecifier := make(map[string][]providerCallableProfileSeed)
	for _, seed := range callableProfileSeeds {
		profilesBySpecifier[seed.Specifier] = append(
			profilesBySpecifier[seed.Specifier],
			seed,
		)
	}
	statefulProfilesBySpecifier := make(map[string][]providerStatefulProfileSeed)
	for _, seed := range statefulProfileSeeds {
		statefulProfilesBySpecifier[seed.Specifier] = append(
			statefulProfilesBySpecifier[seed.Specifier],
			seed,
		)
	}
	providerInterfacesBySpecifier := make(map[string][]providerInterfaceSeed)
	for _, seed := range providerInterfaceSeeds {
		providerInterfacesBySpecifier[seed.Specifier] = append(
			providerInterfacesBySpecifier[seed.Specifier],
			seed,
		)
	}
	specifiers := make([]string, 0, len(bySpecifier))
	for specifier := range bySpecifier {
		specifiers = append(specifiers, specifier)
	}
	for specifier := range representationsBySpecifier {
		if _, selected := bySpecifier[specifier]; !selected {
			specifiers = append(specifiers, specifier)
		}
	}
	for specifier := range profilesBySpecifier {
		if _, selected := bySpecifier[specifier]; selected {
			continue
		}
		if _, selected := representationsBySpecifier[specifier]; !selected {
			specifiers = append(specifiers, specifier)
		}
	}
	for specifier := range statefulProfilesBySpecifier {
		if _, selected := bySpecifier[specifier]; selected {
			continue
		}
		if _, selected := representationsBySpecifier[specifier]; selected {
			continue
		}
		if _, selected := profilesBySpecifier[specifier]; !selected {
			specifiers = append(specifiers, specifier)
		}
	}
	for specifier := range providerInterfacesBySpecifier {
		if _, selected := bySpecifier[specifier]; selected {
			continue
		}
		if _, selected := representationsBySpecifier[specifier]; selected {
			continue
		}
		if _, selected := profilesBySpecifier[specifier]; !selected {
			if _, selected := statefulProfilesBySpecifier[specifier]; !selected {
				specifiers = append(specifiers, specifier)
			}
		}
	}
	sort.Strings(specifiers)
	result := make([]gostdlib.FacetModuleDocument, 0, len(specifiers))
	for _, specifier := range specifiers {
		selected := bySpecifier[specifier]
		selectedRepresentations := representationsBySpecifier[specifier]
		selectedProfiles := profilesBySpecifier[specifier]
		selectedStatefulProfiles := statefulProfilesBySpecifier[specifier]
		selectedProviderInterfaces := providerInterfacesBySpecifier[specifier]
		sourcePath := ""
		if len(selected) != 0 {
			sourcePath = selected[0].SourcePath
		} else if len(selectedRepresentations) != 0 {
			sourcePath = selectedRepresentations[0].SourcePath
		} else if len(selectedProfiles) != 0 {
			sourcePath = selectedProfiles[0].SourcePath
		} else if len(selectedStatefulProfiles) != 0 {
			sourcePath = selectedStatefulProfiles[0].SourcePath
		} else if len(selectedProviderInterfaces) != 0 {
			sourcePath = selectedProviderInterfaces[0].SourcePath
		}
		for _, seed := range selected {
			if seed.SourcePath != sourcePath {
				return nil, certifyError(
					"build facets",
					specifier,
					"one facet module has multiple source files",
				)
			}
		}
		for _, seed := range selectedRepresentations {
			if seed.SourcePath != sourcePath {
				return nil, certifyError(
					"build representations",
					specifier,
					"one facet module has multiple source files",
				)
			}
		}
		for _, seed := range selectedProfiles {
			if seed.SourcePath != sourcePath {
				return nil, certifyError(
					"build provider callable profiles",
					specifier,
					"one profile module has multiple source files",
				)
			}
		}
		for _, seed := range selectedStatefulProfiles {
			if seed.SourcePath != sourcePath {
				return nil, certifyError(
					"build provider stateful profiles",
					specifier,
					"one profile module has multiple source files",
				)
			}
		}
		for _, seed := range selectedProviderInterfaces {
			if seed.SourcePath != sourcePath {
				return nil, certifyError(
					"build provider interfaces",
					specifier,
					"one provider-interface module has multiple source files",
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
		providerInterfaceDocuments := make(
			[]gostdlib.ProviderInterfaceBindingDocument,
			0,
			len(selectedProviderInterfaces),
		)
		for _, seed := range selectedProviderInterfaces {
			target, ok := byName[seed.Export]
			if !ok {
				return nil, certifyError(
					"build provider interface",
					seed.Export,
					"target export is absent",
				)
			}
			selected, err := buildLanguageProviderInterfaceBinding(
				seed,
				target,
				project,
				effectMarker,
			)
			if err != nil {
				return nil, err
			}
			providerInterfaceDocuments = append(
				providerInterfaceDocuments,
				selected,
			)
			owned[selected.Export] = struct{}{}
		}
		representationDocuments := make(
			[]gostdlib.ProviderRepresentationDocument,
			0,
			len(selectedRepresentations),
		)
		representations := make(
			map[string]gostdlib.ProviderRepresentationDocument,
			len(selectedRepresentations),
		)
		for _, seed := range selectedRepresentations {
			target, ok := byName[seed.Export]
			if !ok {
				return nil, certifyError(
					"build representation",
					seed.Export,
					"target export is absent",
				)
			}
			representation, err := buildProviderRepresentation(
				source,
				seed,
				target,
				interfaceTargets,
				project,
				effectMarker,
			)
			if err != nil {
				return nil, err
			}
			representationDocuments = append(
				representationDocuments,
				representation,
			)
			representations[representation.Export] = representation
			owned[representation.Export] = struct{}{}
		}
		profileDocuments := make(
			[]gostdlib.ProviderCallableProfileDocument,
			0,
			len(selectedProfiles),
		)
		statefulProfileDocuments := make(
			[]gostdlib.ProviderStatefulProfileDocument,
			0,
			len(selectedStatefulProfiles),
		)
		for _, seed := range selectedStatefulProfiles {
			built, err := buildProviderStatefulProfile(
				selectedToolchain,
				source,
				seed,
				byName,
				bindingDocuments,
				project,
				effectMarker,
			)
			if err != nil {
				return nil, err
			}
			statefulProfileDocuments = append(
				statefulProfileDocuments,
				built.profile,
			)
			owned[built.profile.Export] = struct{}{}
			for _, selected := range built.profile.Interfaces {
				owned[selected.Export] = struct{}{}
			}
		}
		for _, seed := range selectedProfiles {
			built, err := buildProviderCallableProfile(
				selectedToolchain,
				source,
				seed,
				byName,
				project,
				effectMarker,
			)
			if err != nil {
				return nil, err
			}
			profileDocuments = append(profileDocuments, built.profile)
			owned[built.profile.Export] = struct{}{}
			for _, selected := range built.profile.Interfaces {
				owned[selected.Export] = struct{}{}
			}
		}
		facets := make([]gostdlib.FacetDocument, 0, len(selected))
		for _, seed := range selected {
			facet, err := buildFacet(
				project,
				source,
				seed,
				byName,
				representations,
				genericOperations,
			)
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
		if err := verifyRepresentationReferences(facets, representations); err != nil {
			return nil, err
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
		sort.Slice(profileDocuments, func(left, right int) bool {
			leftKey := profileDocuments[left].SourceIdentity + "\x00" +
				profileDocuments[left].ProfileKey
			rightKey := profileDocuments[right].SourceIdentity + "\x00" +
				profileDocuments[right].ProfileKey
			return leftKey < rightKey
		})
		sort.Slice(statefulProfileDocuments, func(left, right int) bool {
			leftKey := statefulProfileDocuments[left].SourceIdentity + "\x00" +
				statefulProfileDocuments[left].ProfileKey
			rightKey := statefulProfileDocuments[right].SourceIdentity + "\x00" +
				statefulProfileDocuments[right].ProfileKey
			return leftKey < rightKey
		})
		result = append(result, gostdlib.FacetModuleDocument{
			Specifier:          specifier,
			SourcePath:         sourcePath,
			Representations:    representationDocuments,
			ProviderInterfaces: providerInterfaceDocuments,
			CallableProfiles:   profileDocuments,
			StatefulProfiles:   statefulProfileDocuments,
			Facets:             facets,
		})
	}
	return result, nil
}
