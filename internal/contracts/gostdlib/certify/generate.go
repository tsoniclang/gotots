package certify

import (
	"bytes"
	"fmt"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

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
	facetSeeds, representationSeeds, definedValueIdentities, genericProjections,
		genericOperations, err :=
		readFacetSeeds(resolved.facetMapPath)
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
	if err := verifyPackageModules(providerPackage, ordered, facetSeeds); err != nil {
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
			project,
			source,
			seed,
			genericProjections,
			genericOperations,
			definedValueIdentities,
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
		genericOperations,
	); err != nil {
		client.Close()
		return nil, err
	}
	if err := verifyGenericCallableProjectionBindings(
		modules,
		genericProjections,
	); err != nil {
		client.Close()
		return nil, err
	}
	modules, err = applyDefinedValueRepresentations(
		source,
		modules,
		facetSeeds,
		definedValueIdentities,
	)
	if err != nil {
		client.Close()
		return nil, err
	}
	facetModules, err := buildFacetModules(
		resolved,
		project,
		source,
		facetSeeds,
		representationSeeds,
		modules,
		genericProjections,
		effectMarker,
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
		GOOS:             selectedToolchain.goos,
		GOARCH:           selectedToolchain.goarch,
		RuntimeDigest:    runtimeDigest,
		ProviderDigest:   integrity,
		Modules:          modules,
		FacetModules:     facetModules,
	})
}

func buildModule(
	config resolvedConfig,
	project *tsgo.ProjectInspection,
	source goSurface,
	seed moduleSeed,
	genericProjections map[string][]gostdlib.GenericTypeArgumentDocument,
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
		binding.GenericTypeArguments, err = certifiedGenericCallableProjection(
			project,
			evidence,
			target,
			genericProjections,
		)
		if err != nil {
			return gostdlib.ModuleDocument{}, err
		}
		_, identityValue := definedValueIdentities[binding.Identity]
		if binding.Kind == gostdlib.BindingFunction || identityValue {
			binding.Effect, err = exportCallableEffect(
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
		parameterCount := 0
		if signature != nil {
			parameterCount = signature.RecvTypeParams().Len() +
				signature.TypeParams().Len()
		}
		for _, operation := range operations {
			for _, reference := range append(
				append(
					[]gostdlib.GenericOperationTypeDocument(nil),
					operation.Parameters...,
				),
				operation.Results...,
			) {
				if err := verifyGenericOperationTypeParameters(
					reference,
					parameterCount,
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

func verifyGenericOperationTypeParameters(
	reference gostdlib.GenericOperationTypeDocument,
	parameterCount int,
) error {
	if reference.Kind == gostdlib.GenericOperationTypeParameter {
		if reference.TypeParameter == nil ||
			*reference.TypeParameter < 0 ||
			*reference.TypeParameter >= parameterCount {
			return fmt.Errorf(
				"type-parameter index is outside its Go declaration",
			)
		}
	}
	for _, child := range []*gostdlib.GenericOperationTypeDocument{
		reference.Key,
		reference.Element,
	} {
		if child == nil {
			continue
		}
		if err := verifyGenericOperationTypeParameters(
			*child,
			parameterCount,
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

func validateSeeds(source []moduleSeed) ([]moduleSeed, error) {
	if len(source) == 0 {
		return nil, certifyError("configure modules", "", "module set is empty")
	}
	result := append([]moduleSeed(nil), source...)
	specifiers := make(map[string]struct{}, len(result))
	sources := make(map[string]struct{}, len(result))
	for index, seed := range result {
		if seed.GoImportPath == "" || seed.GoImportPath == "." ||
			path.Clean(seed.GoImportPath) != seed.GoImportPath ||
			strings.HasPrefix(seed.GoImportPath, "../") ||
			strings.HasPrefix(seed.GoImportPath, "/") {
			return nil, certifyError(
				"configure modules",
				seed.GoImportPath,
				"Go import path is not canonical",
			)
		}
		if _, ok := providerSubpath(seed.Specifier); !ok ||
			path.Clean(seed.SourcePath) != seed.SourcePath ||
			!strings.HasPrefix(seed.SourcePath, "src/") ||
			!strings.HasSuffix(seed.SourcePath, ".ts") {
			return nil, certifyError("configure modules", seed.GoImportPath, "identity is incomplete")
		}
		if index != 0 && result[index-1].GoImportPath >= seed.GoImportPath {
			return nil, certifyError(
				"configure modules",
				seed.GoImportPath,
				"modules are not strictly ordered",
			)
		}
		if _, duplicate := specifiers[seed.Specifier]; duplicate {
			return nil, certifyError("configure modules", seed.Specifier, "specifier is duplicated")
		}
		if _, duplicate := sources[seed.SourcePath]; duplicate {
			return nil, certifyError("configure modules", seed.SourcePath, "source is duplicated")
		}
		specifiers[seed.Specifier] = struct{}{}
		sources[seed.SourcePath] = struct{}{}
	}
	return result, nil
}

func verifyPublicName(name string, targetType string) error {
	if name == "" || targetType == "" {
		return fmt.Errorf("public symbol identity is incomplete")
	}
	for _, forbidden := range []string{
		"$argument",
		"__from_",
		"$cooperative_",
		"$contract",
		"$state",
	} {
		if strings.Contains(name, forbidden) || strings.Contains(targetType, forbidden) {
			return fmt.Errorf("public symbol contains encoded ABI spelling %q", forbidden)
		}
	}
	return nil
}

func compareCanonical(left []byte, right []byte) error {
	if bytes.Equal(left, right) {
		return nil
	}
	return certifyError(
		"verify manifest",
		"canonical bytes",
		"checked manifest differs from independently regenerated evidence",
	)
}

func readManifest(path string) ([]byte, gostdlib.Manifest, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, gostdlib.Manifest{}, certifyError("read manifest", path, err.Error())
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		return nil, gostdlib.Manifest{}, err
	}
	canonical, err := gostdlib.Encode(manifest)
	if err != nil {
		return nil, gostdlib.Manifest{}, err
	}
	return canonical, manifest, nil
}
