package naming

import (
	"go/types"
	"slices"
	"sort"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

type interfaceContractDemand struct {
	source *types.Interface
	target interfaceContractSelection
}

type interfaceDemandRequestKey struct {
	kind       uint8
	sourceKey  string
	targetKey  string
	adapterKey string
}

type interfaceContractSelection struct {
	sourceType  types.Type
	contract    *types.Interface
	contractKey string
	surfaceKey  string
}

type interfaceReflectionDemand struct {
	source         *types.Interface
	reflectionType *types.TypeName
}

type reflectionInterfaceExposure struct {
	source   interfaceContractSelection
	adapters map[string]struct{}
}

func (r *Registry) IndexCompilationTargets(
	sourcePackages []*load.Package,
	environmentPackages []*load.Package,
	certificate *certify.Certificate,
	externalModules []string,
) error {
	if r == nil {
		return &api.NameError{Reason: "declaration registry is nil"}
	}
	packages := make(
		[]*types.Package,
		0,
		len(sourcePackages)+len(environmentPackages),
	)
	typeInformation := make(
		[]*types.Info,
		0,
		len(sourcePackages)+len(environmentPackages),
	)
	for _, sourcePackage := range sourcePackages {
		if sourcePackage == nil || sourcePackage.Types() == nil {
			return &api.NameError{Reason: "source package identity is nil"}
		}
		typesPackage := sourcePackage.Types()
		if _, duplicate := r.assemblyPathByPackage[typesPackage]; duplicate {
			return &api.NameError{
				Name:   typesPackage.Path(),
				Reason: "source package identity is duplicated",
			}
		}
		assemblyPath, err := output.PackageAssemblyPath(sourcePackage)
		if err != nil {
			return err
		}
		r.assemblyPathByPackage[typesPackage] = assemblyPath
		packages = append(packages, typesPackage)
		typeInformation = append(typeInformation, sourcePackage.TypesInfo())
	}
	var provider standardLibraryProvider
	if certificate != nil {
		provider = certificate
	}
	r.provider = provider
	providerModules := slices.Clone(externalModules)
	if provider != nil {
		providerModules = append(providerModules, provider.ProviderModules()...)
	}
	if len(providerModules) != 0 {
		sort.Strings(providerModules)
		providerModules = slices.Compact(providerModules)
		if err := r.indexProviderImportNames(providerModules); err != nil {
			return err
		}
	}
	for _, environmentPackage := range environmentPackages {
		if environmentPackage == nil ||
			!environmentPackage.Kind().EnvironmentContract() ||
			environmentPackage.Types() == nil {
			return &api.NameError{
				Reason: "environment package identity is invalid",
			}
		}
		typesPackage := environmentPackage.Types()
		if _, duplicate := r.assemblyPathByPackage[typesPackage]; duplicate {
			return &api.NameError{
				Name:   typesPackage.Path(),
				Reason: "environment package identity is duplicated",
			}
		}
		contractPath, err := output.EnvironmentContractPath(
			environmentPackage,
		)
		if err != nil {
			return err
		}
		targetPath := contractPath
		if provider != nil &&
			environmentPackage.Kind() == load.PackageStandardLibraryContract {
			targetPath, err = output.StandardLibraryConstantProjectionPath(
				environmentPackage,
			)
			if err != nil {
				return err
			}
		}
		r.assemblyPathByPackage[typesPackage] = targetPath
		if err := r.indexEnvironmentPackage(
			environmentPackage,
			targetPath,
			provider,
		); err != nil {
			return err
		}
		packages = append(packages, typesPackage)
		typeInformation = append(typeInformation, environmentPackage.TypesInfo())
	}
	if provider != nil {
		if err := r.indexProviderInterfaceCapabilities(); err != nil {
			return err
		}
	}
	return r.indexPackageQualifiers(packages, typeInformation...)
}

func (r *Registry) indexEnvironmentPackage(
	sourcePackage *load.Package,
	contractPath string,
	provider standardLibraryProvider,
) error {
	scope := sourcePackage.Types().Scope()
	names := scope.Names()
	sort.Strings(names)
	used := make(map[string]struct{}, len(names)+1)
	used[api.TargetGlobalAnchorName] = struct{}{}
	for _, sourceName := range names {
		object := scope.Lookup(sourceName)
		if object == nil || object.Name() == "_" {
			continue
		}
		name := allocatePackageName(portableIdentifier(object.Name()), used)
		binding := targetBinding{
			name:         name,
			sourcePath:   contractPath,
			moduleExport: true,
			kind:         targetBindingEnvironment,
		}
		binding, err := selectProviderBinding(
			sourcePackage,
			object,
			binding,
			provider,
		)
		if err != nil {
			return err
		}
		if err := r.reserve(object, binding); err != nil {
			return err
		}
		if provider != nil && (binding.kind == targetBindingProvider ||
			binding.kind == targetBindingMissingProvider) {
			contract, contractErr := environmentcontract.Describe(object)
			if contractErr != nil {
				return contractErr
			}
			if existing := r.providerObjectByIdentity[contract.Identity()]; existing != nil && existing != object {
				return &api.NameError{
					Name:   contract.Identity(),
					Reason: "provider source identity is duplicated",
				}
			}
			r.providerObjectByIdentity[contract.Identity()] = object
		}
		if variable, ok := object.(*types.Var); ok {
			r.packageVariables[variable] = packageVariableBinding{
				fieldName:    name,
				statePath:    contractPath,
				assemblyPath: contractPath,
			}
		}
		typeName, ok := object.(*types.TypeName)
		if !ok || typeName.IsAlias() {
			continue
		}
		named, ok := types.Unalias(typeName.Type()).(*types.Named)
		if !ok {
			continue
		}
		if structure, ok := named.Underlying().(*types.Struct); ok {
			r.indexEnvironmentFields(structure)
		}
		for index := range named.NumMethods() {
			method := named.Method(index).Origin()
			signature, ok := method.Type().(*types.Signature)
			if !ok || signature.Recv() == nil {
				continue
			}
			base := portableIdentifier(typeName.Name()) + "_" +
				portableIdentifier(method.Name())
			methodName := allocatePackageName(base, used)
			methodBinding := targetBinding{
				name:         methodName,
				sourcePath:   contractPath,
				moduleExport: true,
				kind:         targetBindingEnvironment,
			}
			methodBinding, err = selectProviderBinding(
				sourcePackage,
				method,
				methodBinding,
				provider,
			)
			if err != nil {
				return err
			}
			if methodBinding.kind == targetBindingMissingProvider &&
				binding.providerRepresentation {
				methodBinding, err = selectProviderRepresentationMethod(
					method,
					methodBinding,
					binding,
					provider,
				)
				if err != nil {
					return err
				}
			}
			if err := r.reserve(method, methodBinding); err != nil {
				return err
			}
			if provider != nil && (methodBinding.kind == targetBindingProvider ||
				methodBinding.kind == targetBindingMissingProvider) {
				contract, contractErr := environmentcontract.Describe(method)
				if contractErr != nil {
					return contractErr
				}
				if existing := r.providerObjectByIdentity[contract.Identity()]; existing != nil && existing != method {
					return &api.NameError{
						Name:   contract.Identity(),
						Reason: "provider source identity is duplicated",
					}
				}
				r.providerObjectByIdentity[contract.Identity()] = method
			}
		}
	}
	return nil
}

func (r *Registry) recordReflectionInterfaceAdapter(
	source interfaceContractSelection,
	binding interfaceAdapterBinding,
) error {
	if r == nil || !source.valid() || binding.owner == nil || binding.key == "" {
		return &api.NameError{
			Reason: "reflection interface exposure is invalid",
		}
	}
	canonical, ok := r.interfaceAdapters[binding.key]
	if !ok || canonical.owner != binding.owner || canonical.name != binding.name ||
		canonical.reflectionName != binding.reflectionName {
		return &api.NameError{
			Reason: "reflection interface exposure has no canonical adapter",
		}
	}
	sourceType, ok := binding.owner.InterfaceAdapterType()
	if !ok || !types.Implements(sourceType, source.contract) {
		return &api.NameError{
			Reason: "reflection interface exposure does not implement its source contract",
		}
	}
	selected, err := r.internInterfaceContract(source)
	if err != nil {
		return err
	}
	key := selected.demandKey()
	exposure := r.reflectionInterfaceExposures[key]
	if exposure.adapters == nil {
		exposure = reflectionInterfaceExposure{
			source:   selected,
			adapters: make(map[string]struct{}),
		}
	} else if !sameInterfaceContractSelection(exposure.source, selected) {
		return &api.NameError{
			Reason: "reflection interface exposure key joined non-identical contracts",
		}
	}
	if _, exists := exposure.adapters[binding.key]; exists {
		return nil
	}
	exposure.adapters[binding.key] = struct{}{}
	r.reflectionInterfaceExposures[key] = exposure
	r.reflectionInterfaceDirty = true
	return nil
}

func (r *Registry) FlushReflectionInterfaceDemands() (
	[]api.RootRequest,
	error,
) {
	if r == nil {
		return nil, &api.NameError{
			Reason: "reflection interface demand owner is nil",
		}
	}
	if !r.reflectionInterfaceDirty {
		return nil, nil
	}
	r.reflectionInterfaceDirty = false
	exposureKeys := make([]string, 0, len(r.reflectionInterfaceExposures))
	for key := range r.reflectionInterfaceExposures {
		exposureKeys = append(exposureKeys, key)
	}
	sort.Strings(exposureKeys)
	var requests []api.RootRequest
	for _, exposureKey := range exposureKeys {
		exposure := r.reflectionInterfaceExposures[exposureKey]
		if !exposure.source.valid() || len(exposure.adapters) == 0 {
			return nil, &api.NameError{
				Reason: "reflection interface exposure state is invalid",
			}
		}
		adapterKeys := make([]string, 0, len(exposure.adapters))
		for adapterKey := range exposure.adapters {
			adapterKeys = append(adapterKeys, adapterKey)
		}
		sort.Strings(adapterKeys)
		for _, adapterKey := range adapterKeys {
			binding, ok := r.interfaceAdapters[adapterKey]
			if !ok || binding.owner == nil {
				return nil, &api.NameError{
					Reason: "reflection interface exposure has no adapter owner",
				}
			}
			reached, err := r.reflectionInterfaceReachableContracts(
				binding,
				exposure.source,
			)
			if err != nil {
				return nil, err
			}
			settled := r.reflectionInterfaceContracts[adapterKey]
			if settled == nil {
				settled = make(map[string]struct{})
				r.reflectionInterfaceContracts[adapterKey] = settled
			}
			keys := make([]string, 0, len(reached))
			for key := range reached {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if _, exists := settled[key]; exists {
					continue
				}
				target := reached[key]
				request, err := api.NewInterfaceAdapterContractRequest(
					binding.owner,
					target.sourceType,
					target.contract,
					key,
				)
				if err != nil {
					return nil, err
				}
				settled[key] = struct{}{}
				requests = append(requests, request)
			}
		}
	}
	return api.CombineRequests(requests), nil
}

func (r *Registry) reflectionInterfaceReachableContracts(
	binding interfaceAdapterBinding,
	source interfaceContractSelection,
) (map[string]interfaceContractSelection, error) {
	sourceType, ok := binding.owner.InterfaceAdapterType()
	if !ok {
		return nil, &api.NameError{
			Reason: "reflection interface exposure has no concrete source type",
		}
	}
	pending := []interfaceContractSelection{source}
	visited := make(map[string]struct{})
	selected := make(map[string]interfaceContractSelection)
	for len(pending) != 0 {
		next := pending[0]
		pending = pending[1:]
		nextKey := next.demandKey()
		if _, duplicate := visited[nextKey]; duplicate {
			continue
		}
		visited[nextKey] = struct{}{}
		targets := r.interfaceContractDemands[next.contractKey]
		targetKeys := make([]string, 0, len(targets))
		for targetKey := range targets {
			targetKeys = append(targetKeys, targetKey)
		}
		sort.Strings(targetKeys)
		for _, targetKey := range targetKeys {
			demand := targets[targetKey]
			if !types.Identical(demand.source, next.contract) {
				return nil, &api.NameError{
					Reason: "reflection interface demand source identity drifted",
				}
			}
			target := demand.target
			if !types.Implements(sourceType, target.contract) {
				continue
			}
			pending = append(pending, target)
			if target.contract.NumMethods() != 0 {
				selected[target.demandKey()] = target
			}
		}
	}
	return selected, nil
}

func (r *Registry) indexEnvironmentFields(structure *types.Struct) {
	used := map[string]struct{}{
		"constructor": {},
		typescriptclass.PromiseAssimilationMember: {},
	}
	for index := range structure.NumFields() {
		field := structure.Field(index)
		name := allocatePackageName(portableIdentifier(field.Name()), used)
		r.memberNameByObject[field] = name
	}
}

func allocatePackageName(
	base string,
	used map[string]struct{},
) string {
	candidate := base
	for suffix := uint64(1); ; suffix++ {
		if _, duplicate := used[candidate]; !duplicate {
			used[candidate] = struct{}{}
			return candidate
		}
		candidate = base + "__declaration_" +
			strconv.FormatUint(suffix, 10)
	}
}

func (r *Registry) indexPackageQualifiers(
	sourcePackages []*types.Package,
	typeInformation ...*types.Info,
) error {
	if r == nil {
		return &api.NameError{Reason: "declaration registry is nil"}
	}
	packages := slices.Clone(sourcePackages)
	for _, sourcePackage := range packages {
		if sourcePackage == nil ||
			sourcePackage.Path() == "" ||
			sourcePackage.Name() == "" {
			return &api.NameError{Reason: "source package identity is nil"}
		}
	}
	sort.Slice(packages, func(left, right int) bool {
		return packages[left].Path() < packages[right].Path()
	})
	used := make(map[string]struct{}, len(packages))
	paths := make(map[string]struct{}, len(packages))
	for _, sourcePackage := range packages {
		if _, duplicate := paths[sourcePackage.Path()]; duplicate {
			return &api.NameError{
				Name:   sourcePackage.Path(),
				Reason: "source package path is duplicated",
			}
		}
		paths[sourcePackage.Path()] = struct{}{}
		base := portableIdentifier(sourcePackage.Name())
		qualifier := base
		for suffix := uint64(1); ; suffix++ {
			if _, duplicate := used[qualifier]; !duplicate {
				break
			}
			qualifier = base + "__package_" + strconv.FormatUint(suffix, 10)
		}
		used[qualifier] = struct{}{}
		r.importQualifierByPackage[sourcePackage] = qualifier
	}
	if err := r.indexPrivateMethodNames(packages, typeInformation); err != nil {
		return err
	}
	return nil
}
