package certify

import (
	"errors"
	"fmt"
	"go/types"
	"path/filepath"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type moduleSeed struct {
	GoImportPath string `json:"goImportPath"`
	Specifier    string `json:"specifier"`
	SourcePath   string `json:"sourcePath"`
}

func Generate(config Config) ([]byte, error) {
	resolved, err := resolveConfig(config)
	if err != nil {
		return nil, err
	}
	runtimeRequirements, runtimeDigest, err := readRuntimeContract(
		resolved.runtimeContractPath,
	)
	if err != nil {
		return nil, err
	}
	ordered, err := readModuleSeeds(resolved.moduleMapPath)
	if err != nil {
		return nil, err
	}
	seeds, err := readFacetSeeds(resolved.facetMapPath)
	if err != nil {
		return nil, err
	}
	selectedToolchain, err := inspectToolchain(resolved)
	if err != nil {
		return nil, err
	}
	providerPackage, err := readProviderPackage(resolved)
	if err != nil {
		return nil, err
	}
	if err := verifyPackageModules(
		providerPackage,
		runtimeRequirements.ProviderScalarModule(),
		runtimeRequirements.ProviderPointerModule(),
		ordered,
		seeds.facets,
		seeds.callableProfiles,
		seeds.statefulProfiles,
		seeds.providerInterfaces,
		seeds.providerCapabilities,
	); err != nil {
		return nil, err
	}
	paths := make([]string, len(ordered))
	for index, seed := range ordered {
		paths[index] = seed.GoImportPath
	}
	source, err := loadGoSurface(resolved, selectedToolchain, paths)
	if err != nil {
		return nil, err
	}
	client, err := tsgo.StartClient(resolved.repositoryRoot, resolved.providerRoot)
	if err != nil {
		return nil, err
	}
	project, err := client.OpenProject(resolved.tsConfigPath)
	if err != nil {
		client.Close()
		return nil, err
	}
	if err := verifyProviderScalarContract(
		resolved,
		project,
		runtimeRequirements,
	); err != nil {
		client.Close()
		return nil, err
	}
	if err := verifyProviderPointerContract(
		resolved,
		project,
		runtimeRequirements,
	); err != nil {
		client.Close()
		return nil, err
	}
	scalarAliases := providerScalarAliasPaths(resolved, runtimeRequirements)
	effectMarker, err := loadCallableEffectMarker(resolved, project)
	if err != nil {
		client.Close()
		return nil, err
	}
	supportMarkers, err := loadProviderSupportMarkers(resolved, project)
	if err != nil {
		client.Close()
		return nil, err
	}
	modules := make([]gostdlib.ModuleDocument, len(ordered))
	behaviorEvidence := newImplementationEvidence()
	var scalarErrors []error
	for index, seed := range ordered {
		module, buildErr := buildModule(
			resolved,
			selectedToolchain,
			project,
			source,
			seed,
			seeds.genericOperations,
			effectMarker,
			scalarAliases,
			&scalarErrors,
			behaviorEvidence,
		)
		if buildErr != nil {
			client.Close()
			return nil, buildErr
		}
		modules[index] = module
	}
	if len(scalarErrors) != 0 {
		client.Close()
		return nil, errors.Join(scalarErrors...)
	}
	if err := verifyGenericOperationBindings(
		source,
		modules,
		seeds.genericOperations,
	); err != nil {
		client.Close()
		return nil, err
	}
	if err := verifySourceGenericCallableProjectionBindings(
		source,
		modules,
	); err != nil {
		client.Close()
		return nil, err
	}
	modules, err = applyDefinedValueRepresentations(
		source,
		modules,
		seeds.facets,
	)
	if err != nil {
		client.Close()
		return nil, err
	}
	facetModules, err := buildFacetModules(
		resolved,
		project,
		source,
		seeds.facets,
		seeds.representations,
		seeds.callableProfiles,
		seeds.statefulProfiles,
		seeds.providerInterfaces,
		seeds.providerCapabilities,
		modules,
		seeds.genericOperations,
		effectMarker,
		supportMarkers,
		selectedToolchain,
		scalarAliases,
		behaviorEvidence,
	)
	if err != nil {
		client.Close()
		return nil, err
	}
	if err := verifyProviderBoundaryCoverage(
		source,
		modules,
		facetModules,
	); err != nil {
		client.Close()
		return nil, err
	}
	implementations, err := certifyPrivateImplementations(
		resolved,
		project,
		behaviorEvidence.public,
		behaviorEvidence.worklist,
	)
	if err != nil {
		client.Close()
		return nil, err
	}
	if err := client.Close(); err != nil {
		return nil, err
	}
	integrity, err := providerDigest(resolved)
	if err != nil {
		return nil, err
	}
	return gostdlib.Seal(gostdlib.Document{
		SchemaVersion:    gostdlib.SchemaVersion,
		PackageName:      gostdlib.PackageName,
		PackageVersion:   providerPackage.Version,
		Backend:          resolved.backend,
		GoVersion:        selectedToolchain.version,
		MinimumGoVersion: resolved.minimumGoVersion,
		MaximumGoVersion: resolved.maximumGoVersion,
		GOOS:             selectedToolchain.profile.GOOS(),
		GOARCH:           selectedToolchain.profile.GOARCH(),
		CGOEnabled:       selectedToolchain.profile.CgoEnabled(),
		BuildTags:        selectedToolchain.profile.Tags(),
		RuntimeDigest:    runtimeDigest,
		ProviderDigest:   integrity,
		Modules:          modules,
		FacetModules:     facetModules,
		Implementations:  implementations,
	})
}

func buildModule(
	config resolvedConfig,
	selectedToolchain toolchain,
	project *tsgo.ProjectInspection,
	source goSurface,
	seed moduleSeed,
	genericOperations map[string][]gostdlib.GenericOperationDocument,
	effectMarker tsgo.ProjectExport,
	scalarAliases map[string]string,
	scalarErrors *[]error,
	behaviorEvidence *implementationEvidence,
) (gostdlib.ModuleDocument, error) {
	sourcePackage := source.packages[seed.GoImportPath]
	if sourcePackage == nil {
		return gostdlib.ModuleDocument{}, certifyError(
			"build module",
			seed.GoImportPath,
			"Go package is absent",
		)
	}
	sourcePath := filepath.Join(config.providerRoot, filepath.FromSlash(seed.SourcePath))
	exports, err := project.Exports(sourcePath)
	if err != nil {
		return gostdlib.ModuleDocument{}, err
	}
	var bindings []gostdlib.BindingDocument
	targetOwners := make(map[string]struct{})
	for _, target := range exports {
		if err := verifyPublicName(target.Name(), target.TypeString()); err != nil {
			return gostdlib.ModuleDocument{}, certifyError(
				"build module",
				seed.Specifier+"#"+target.Name(),
				err.Error(),
			)
		}
		if target.Name() == "state" {
			stateBindings, err := buildStateBindings(
				config,
				project,
				sourcePackage,
				target,
				behaviorEvidence,
			)
			if err != nil {
				return gostdlib.ModuleDocument{}, err
			}
			for _, binding := range stateBindings {
				if err := addTargetOwner(targetOwners, binding); err != nil {
					return gostdlib.ModuleDocument{}, err
				}
			}
			bindings = append(bindings, stateBindings...)
			continue
		}
		evidence, ok := sourcePackage.objectsByName[target.Name()]
		if !ok {
			return gostdlib.ModuleDocument{}, certifyError(
				"build module",
				seed.Specifier+"#"+target.Name(),
				"public export has no selected-GOROOT declaration",
			)
		}
		binding, err := bindingDocument(
			evidence,
			target.Name(),
			"",
			gostdlib.AccessExport,
			target.Fingerprint(),
			target.ImplementationOwners(),
		)
		if err != nil {
			return gostdlib.ModuleDocument{}, err
		}
		binding.GenericOperations = genericOperations[binding.Identity]
		if binding.Kind == gostdlib.BindingFunction ||
			binding.Kind == gostdlib.BindingVariable {
			sites, behavior, behaviorErr := certifyBindingBehavior(
				config,
				project,
				binding.Identity,
				target.Name(),
				target.DeclarationNodeHandles(),
				behaviorEvidence,
			)
			if behaviorErr != nil {
				return gostdlib.ModuleDocument{}, behaviorErr
			}
			binding.ImplementationSites = sites
			binding.Dependencies = behavior.dependencies
			binding.Disposition = behavior.disposition
		}
		if binding.Kind == gostdlib.BindingType {
			// A public class edge carries construction behavior only;
			// member behavior joins per method binding.
			sites, behavior, behaviorErr := certifyConstructionBehavior(
				config,
				project,
				binding.Identity,
				target.Name(),
				target.DeclarationNodeHandles(),
				behaviorEvidence,
			)
			if behaviorErr != nil {
				return gostdlib.ModuleDocument{}, behaviorErr
			}
			binding.ImplementationSites = sites
			binding.Dependencies = behavior.dependencies
			binding.Disposition = behavior.disposition
		}
		if binding.Kind == gostdlib.BindingFunction {
			if err := verifyExportSourceCallableShape(
				project,
				evidence,
				target,
			); err != nil {
				return gostdlib.ModuleDocument{}, err
			}
			if err := verifyExportSourceCallableScalars(
				project,
				evidence,
				target,
				scalarAliases,
			); err != nil {
				*scalarErrors = append(*scalarErrors, err)
			}
			binding.GenericTypeArguments, err =
				certifiedSourceGenericCallableProjection(
					project,
					evidence,
					target,
				)
			if err != nil {
				return gostdlib.ModuleDocument{}, err
			}
		}
		if binding.Kind == gostdlib.BindingFunction {
			binding.Effect, err = exportCallableEffect(
				project,
				target,
				effectMarker,
			)
			if err != nil {
				return gostdlib.ModuleDocument{}, err
			}
			binding.CallableParameters, err = exportCallableParameters(
				evidence,
				target,
				project,
				effectMarker,
			)
			if err != nil {
				return gostdlib.ModuleDocument{}, err
			}
		}
		if err := addTargetOwner(targetOwners, binding); err != nil {
			return gostdlib.ModuleDocument{}, err
		}
		bindings = append(bindings, binding)
		typeName, ok := evidence.object.(*types.TypeName)
		if !ok || typeName.IsAlias() {
			continue
		}
		binding.ProviderInterface, err = buildProviderInterface(
			selectedToolchain,
			sourcePackage,
			typeName,
			target,
			project,
			effectMarker,
		)
		if err != nil {
			return gostdlib.ModuleDocument{}, err
		}
		named, namedOK := types.Unalias(typeName.Type()).(*types.Named)
		if !namedOK {
			return gostdlib.ModuleDocument{}, certifyError(
				"build module",
				binding.Identity,
				"defined provider type has no named source type",
			)
		}
		binding.StructFields, err = buildProviderStructFields(
			selectedToolchain,
			source,
			typeName,
			named,
			target,
			project,
			scalarAliases,
		)
		if err != nil {
			return gostdlib.ModuleDocument{}, err
		}
		bindings[len(bindings)-1] = binding
		methodBindings, err := buildMethodBindings(
			config,
			project,
			target,
			sourcePackage.methodsByType[target.Name()],
			effectMarker,
			scalarAliases,
			scalarErrors,
			behaviorEvidence,
		)
		if err != nil {
			return gostdlib.ModuleDocument{}, err
		}
		for _, methodBinding := range methodBindings {
			if err := addTargetOwner(targetOwners, methodBinding); err != nil {
				return gostdlib.ModuleDocument{}, err
			}
		}
		bindings = append(bindings, methodBindings...)
	}
	sort.Slice(bindings, func(left, right int) bool {
		return bindings[left].Identity < bindings[right].Identity
	})
	return gostdlib.ModuleDocument{
		GoImportPath: seed.GoImportPath,
		Specifier:    seed.Specifier,
		SourcePath:   seed.SourcePath,
		Bindings:     bindings,
	}, nil
}

func verifyContractTypeParameters(
	reference gostdlib.ContractTypeDocument,
	typeParameterCount int,
	callableParameterCount int,
) error {
	if reference.Kind == gostdlib.ContractTypeParameter {
		if reference.TypeParameter == nil ||
			*reference.TypeParameter < 0 ||
			*reference.TypeParameter >= typeParameterCount {
			return fmt.Errorf(
				"type-parameter index is outside its Go declaration",
			)
		}
	}
	if reference.Kind == gostdlib.ContractTypeCallableParameter {
		if reference.CallableParameter == nil ||
			*reference.CallableParameter < 0 ||
			*reference.CallableParameter >= callableParameterCount {
			return fmt.Errorf(
				"callable-parameter index is outside its Go declaration",
			)
		}
	}
	for _, child := range []*gostdlib.ContractTypeDocument{
		reference.Key,
		reference.Element,
	} {
		if child == nil {
			continue
		}
		if err := verifyContractTypeParameters(
			*child,
			typeParameterCount,
			callableParameterCount,
		); err != nil {
			return err
		}
	}
	return nil
}

func buildMethodBindings(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	methods []goObject,
	effectMarker tsgo.ProjectExport,
	scalarAliases map[string]string,
	scalarErrors *[]error,
	behaviorEvidence *implementationEvidence,
) ([]gostdlib.BindingDocument, error) {
	var result []gostdlib.BindingDocument
	for _, method := range methods {
		name := method.object.Name()
		static, staticOK := target.ValueMember(name)
		instance, instanceOK := target.TypeMember(name)
		if staticOK && !static.Visible() {
			staticOK = false
		}
		if instanceOK && !instance.Visible() {
			instanceOK = false
		}
		if !staticOK && !instanceOK {
			continue
		}
		selected, access, err := selectMethodOwner(
			method,
			static,
			staticOK,
			instance,
			instanceOK,
		)
		if err != nil {
			return nil, err
		}
		binding, err := bindingDocument(
			method,
			target.Name(),
			name,
			access,
			selected.Fingerprint(),
			selected.ImplementationOwners(),
		)
		if err != nil {
			return nil, err
		}
		sites, behavior, behaviorErr := certifyBindingBehavior(
			config,
			project,
			binding.Identity,
			name,
			selected.DeclarationNodeHandles(),
			behaviorEvidence,
		)
		if behaviorErr != nil {
			return nil, behaviorErr
		}
		binding.ImplementationSites = sites
		binding.Dependencies = behavior.dependencies
		binding.Disposition = behavior.disposition
		if err := verifyMethodSourceCallableShape(
			project,
			method,
			selected,
			access,
		); err != nil {
			return nil, err
		}
		if err := verifyMethodSourceCallableScalars(
			project,
			method,
			selected,
			access,
			scalarAliases,
		); err != nil {
			*scalarErrors = append(*scalarErrors, err)
		}
		binding.Effect, err = memberCallableEffect(
			project,
			selected,
			effectMarker,
		)
		if err != nil {
			return nil, err
		}
		binding.CallableParameters, err = memberCallableParameters(
			method,
			selected,
			access,
			project,
			effectMarker,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, binding)
	}
	return result, nil
}
