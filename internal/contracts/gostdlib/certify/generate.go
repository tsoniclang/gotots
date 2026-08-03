package certify

import (
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
		ordered,
		seeds.facets,
		seeds.callableProfiles,
		seeds.statefulProfiles,
		seeds.providerInterfaces,
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
	effectMarker, err := loadCallableEffectMarker(resolved, project)
	if err != nil {
		client.Close()
		return nil, err
	}
	modules := make([]gostdlib.ModuleDocument, len(ordered))
	for index, seed := range ordered {
		module, buildErr := buildModule(
			resolved,
			selectedToolchain,
			project,
			source,
			seed,
			seeds.genericOperations,
			seeds.definedValueIdentities,
			effectMarker,
		)
		if buildErr != nil {
			client.Close()
			return nil, buildErr
		}
		modules[index] = module
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
		seeds.definedValueIdentities,
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
		modules,
		seeds.genericOperations,
		effectMarker,
		selectedToolchain,
	)
	if err != nil {
		client.Close()
		return nil, err
	}
	if err := client.Close(); err != nil {
		return nil, err
	}
	_, runtimeDigest, err := readRuntimeContract(resolved.runtimeContractPath)
	if err != nil {
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
	})
}

func buildModule(
	config resolvedConfig,
	selectedToolchain toolchain,
	project *tsgo.ProjectInspection,
	source goSurface,
	seed moduleSeed,
	genericOperations map[string][]gostdlib.GenericOperationDocument,
	definedValueIdentities map[string]struct{},
	effectMarker tsgo.ProjectExport,
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
			stateBindings, err := buildStateBindings(sourcePackage, target)
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
		if binding.Kind == gostdlib.BindingFunction {
			if err := verifyExportSourceCallableShape(
				project,
				evidence,
				target,
			); err != nil {
				return gostdlib.ModuleDocument{}, err
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
		_, identityValue := definedValueIdentities[binding.Identity]
		if binding.Kind == gostdlib.BindingFunction {
			binding.Effect, err = exportCallableEffect(
				project,
				target,
				effectMarker,
			)
			if err != nil {
				return gostdlib.ModuleDocument{}, err
			}
		} else if identityValue {
			binding.Effect, err = exportCallableValueFacetEffect(
				project,
				target,
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
		bindings[len(bindings)-1] = binding
		methodBindings, err := buildMethodBindings(
			project,
			target,
			sourcePackage.methodsByType[target.Name()],
			effectMarker,
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

func verifyGenericOperationBindings(
	source goSurface,
	modules []gostdlib.ModuleDocument,
	configured map[string][]gostdlib.GenericOperationDocument,
) error {
	bound := make(map[string]struct{}, len(configured))
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if len(binding.GenericOperations) != 0 {
				bound[binding.Identity] = struct{}{}
			}
		}
	}
	for identity, operations := range configured {
		evidence, ok := source.objects[identity]
		if !ok {
			return certifyError(
				"configure generic operations",
				identity,
				"selected-GOROOT declaration is absent",
			)
		}
		function, ok := evidence.object.(*types.Func)
		if !ok {
			return certifyError(
				"configure generic operations",
				identity,
				"operation-set owner is not a function",
			)
		}
		signature, _ := function.Type().(*types.Signature)
		typeParameterCount := 0
		callableParameterCount := 0
		if signature != nil {
			typeParameterCount = signature.RecvTypeParams().Len() +
				signature.TypeParams().Len()
			callableParameterCount = signature.Params().Len()
		}
		for _, operation := range operations {
			for _, reference := range append(
				append(
					[]gostdlib.ContractTypeDocument(nil),
					operation.Parameters...,
				),
				operation.Results...,
			) {
				if err := verifyContractTypeParameters(
					reference,
					typeParameterCount,
					callableParameterCount,
				); err != nil {
					return certifyError(
						"configure generic operations",
						identity,
						err.Error(),
					)
				}
			}
		}
		if _, ok := bound[identity]; !ok {
			return certifyError(
				"configure generic operations",
				identity,
				"provider export is absent",
			)
		}
	}
	return nil
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

func buildStateBindings(
	source *goPackageSurface,
	target tsgo.ProjectExport,
) ([]gostdlib.BindingDocument, error) {
	var result []gostdlib.BindingDocument
	for _, member := range target.ValueMembers() {
		if !member.Visible() {
			continue
		}
		evidence, ok := source.objectsByName[member.Name()]
		if !ok {
			return nil, certifyError(
				"build state",
				member.Name(),
				"state member has no selected-GOROOT declaration",
			)
		}
		if _, ok := evidence.object.(*types.Var); !ok {
			return nil, certifyError(
				"build state",
				member.Name(),
				"state member does not own a Go variable",
			)
		}
		binding, err := bindingDocument(
			evidence,
			"state",
			member.Name(),
			gostdlib.AccessStateMember,
			member.Fingerprint(),
			member.ImplementationOwners(),
		)
		if err != nil {
			return nil, err
		}
		result = append(result, binding)
	}
	if len(result) == 0 {
		return nil, certifyError("build state", target.Name(), "state has no members")
	}
	return result, nil
}

func buildMethodBindings(
	project *tsgo.ProjectInspection,
	target tsgo.ProjectExport,
	methods []goObject,
	effectMarker tsgo.ProjectExport,
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
		if err := verifyMethodSourceCallableShape(
			project,
			method,
			selected,
			access,
		); err != nil {
			return nil, err
		}
		binding.Effect, err = memberCallableEffect(
			project,
			selected,
			effectMarker,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, binding)
	}
	return result, nil
}

func selectMethodOwner(
	method goObject,
	static tsgo.ProjectMember,
	staticOK bool,
	instance tsgo.ProjectMember,
	instanceOK bool,
) (tsgo.ProjectMember, gostdlib.AccessKind, error) {
	signature, ok := method.object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return tsgo.ProjectMember{}, gostdlib.AccessInvalid, certifyError(
			"build methods",
			method.contract.Identity(),
			"Go method receiver evidence is absent",
		)
	}
	_, pointerReceiver := signature.Recv().Type().(*types.Pointer)
	if pointerReceiver {
		if !staticOK {
			return tsgo.ProjectMember{}, gostdlib.AccessInvalid, certifyError(
				"build methods",
				method.contract.Identity(),
				"pointer receiver has no static operation",
			)
		}
		return static, gostdlib.AccessStaticMethod, nil
	}
	if !instanceOK {
		return tsgo.ProjectMember{}, gostdlib.AccessInvalid, certifyError(
			"build methods",
			method.contract.Identity(),
			"value receiver has no instance operation",
		)
	}
	return instance, gostdlib.AccessInstanceMethod, nil
}

func addTargetOwner(
	owners map[string]struct{},
	binding gostdlib.BindingDocument,
) error {
	key := string(binding.Access) + "\x00" + binding.Export + "\x00" + binding.Member
	if _, duplicate := owners[key]; duplicate {
		return certifyError("build binding", key, "target owner is duplicated")
	}
	owners[key] = struct{}{}
	return nil
}
