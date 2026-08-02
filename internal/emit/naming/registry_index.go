package naming

import (
	"go/types"
	"slices"
	"sort"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

func (r *Registry) IndexCompilationTargets(
	sourcePackages []*load.Package,
	environmentPackages []*load.Package,
	certificate *certify.Certificate,
) error {
	if r == nil {
		return &api.NameError{Reason: "declaration registry is nil"}
	}
	packages := make(
		[]*types.Package,
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
	}
	var provider standardLibraryProvider
	if certificate != nil {
		provider = certificate
	}
	r.provider = provider
	if provider != nil {
		if err := r.indexProviderImportNames(provider.ProviderModules()); err != nil {
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
	}
	return r.indexPackageQualifiers(packages)
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

func (r *Registry) indexEnvironmentFields(structure *types.Struct) {
	used := map[string]struct{}{"constructor": {}}
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
	return nil
}
