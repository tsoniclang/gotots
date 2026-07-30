package naming

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

type targetBinding struct {
	name         string
	sourceFile   *ast.File
	sourcePath   string
	moduleExport bool
	kind         targetBindingKind
}

type targetBindingKind uint8

const (
	targetBindingLocal targetBindingKind = iota
	targetBindingSource
	targetBindingEnvironment
)

func (b targetBinding) scheduled() bool {
	return b.kind == targetBindingSource ||
		b.kind == targetBindingEnvironment
}

func (b targetBinding) sourceOwned() bool {
	return b.kind == targetBindingSource
}

type packageVariableBinding struct {
	fieldName    string
	statePath    string
	assemblyPath string
}

type anonymousStructBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type mapSpecializationBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type interfaceAdapterBinding struct {
	owner *api.GeneratedArtifact
	name  string
	key   string
}

type anonymousInterfaceBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type interfaceMethodTokenBinding struct {
	owner  *api.GeneratedArtifact
	method *types.Func
	name   string
}

type interfaceDynamicTypeTokenBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type interfaceContractDemand struct {
	source *types.Interface
	target *types.Interface
}

type genericCapabilityBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type callableABIBinding struct {
	owner *api.GeneratedArtifact
	name  string
}

type Target struct {
	Name       string
	SourcePath string
}

type Registry struct {
	byObject                    map[types.Object]targetBinding
	memberNameByObject          map[*types.Var]string
	packageVariables            map[*types.Var]packageVariableBinding
	assemblyPathByPackage       map[*types.Package]string
	importQualifierByPackage    map[*types.Package]string
	anonymousStructs            map[string]anonymousStructBinding
	anonymousStructNames        map[string]string
	mapSpecializations          map[string]mapSpecializationBinding
	mapSpecializationNames      map[string]string
	interfaceAdapters           map[string]interfaceAdapterBinding
	interfaceAdapterNames       map[string]string
	anonymousInterfaces         map[string]anonymousInterfaceBinding
	anonymousInterfaceNames     map[string]string
	interfaceMethodTokens       map[string]interfaceMethodTokenBinding
	interfaceMethodNames        map[string]string
	interfaceDynamicTypes       map[string]interfaceDynamicTypeTokenBinding
	interfaceDynamicNames       map[string]string
	interfaceContracts          map[string]*types.Interface
	interfaceAdaptersByContract map[string]map[string]struct{}
	interfaceContractDemands    map[string]map[string]interfaceContractDemand
	genericCapabilities         map[string]genericCapabilityBinding
	genericCapabilityNames      map[string]string
	callableABIs                map[string]callableABIBinding
	callableABINames            map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		byObject:                    make(map[types.Object]targetBinding),
		memberNameByObject:          make(map[*types.Var]string),
		packageVariables:            make(map[*types.Var]packageVariableBinding),
		assemblyPathByPackage:       make(map[*types.Package]string),
		importQualifierByPackage:    make(map[*types.Package]string),
		anonymousStructs:            make(map[string]anonymousStructBinding),
		anonymousStructNames:        make(map[string]string),
		mapSpecializations:          make(map[string]mapSpecializationBinding),
		mapSpecializationNames:      make(map[string]string),
		interfaceAdapters:           make(map[string]interfaceAdapterBinding),
		interfaceAdapterNames:       make(map[string]string),
		anonymousInterfaces:         make(map[string]anonymousInterfaceBinding),
		anonymousInterfaceNames:     make(map[string]string),
		interfaceMethodTokens:       make(map[string]interfaceMethodTokenBinding),
		interfaceMethodNames:        make(map[string]string),
		interfaceDynamicTypes:       make(map[string]interfaceDynamicTypeTokenBinding),
		interfaceDynamicNames:       make(map[string]string),
		interfaceContracts:          make(map[string]*types.Interface),
		interfaceAdaptersByContract: make(map[string]map[string]struct{}),
		interfaceContractDemands:    make(map[string]map[string]interfaceContractDemand),
		genericCapabilities:         make(map[string]genericCapabilityBinding),
		genericCapabilityNames:      make(map[string]string),
		callableABIs:                make(map[string]callableABIBinding),
		callableABINames:            make(map[string]string),
	}
}

func (r *Registry) Target(object types.Object) (Target, bool) {
	if r == nil {
		return Target{}, false
	}
	binding, ok := r.byObject[object]
	if !ok {
		return Target{}, false
	}
	return Target{
		Name:       binding.name,
		SourcePath: binding.sourcePath,
	}, true
}

func (r *Registry) ImportQualifier(sourcePackage *types.Package) string {
	if r == nil {
		return ""
	}
	return r.importQualifierByPackage[sourcePackage]
}

func (r *Registry) GeneratedArtifact(
	kind api.GeneratedArtifactKind,
	artifactKey string,
) (*api.GeneratedArtifact, bool) {
	if r == nil || !kind.Valid() || artifactKey == "" {
		return nil, false
	}
	switch kind {
	case api.GeneratedArtifactAnonymousStruct:
		binding, ok := r.anonymousStructs[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactMapSpecialization:
		binding, ok := r.mapSpecializations[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactInterfaceAdapter:
		binding, ok := r.interfaceAdapters[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactAnonymousInterface:
		binding, ok := r.anonymousInterfaces[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactInterfaceMethodToken:
		binding, ok := r.interfaceMethodTokens[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		binding, ok := r.interfaceDynamicTypes[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactGenericCapability:
		binding, ok := r.genericCapabilities[artifactKey]
		return binding.owner, ok && binding.owner != nil
	case api.GeneratedArtifactCallableABI:
		binding, ok := r.callableABIs[artifactKey]
		return binding.owner, ok && binding.owner != nil
	default:
		return nil, false
	}
}

func (r *Registry) GeneratedArtifacts(
	kind api.GeneratedArtifactKind,
) []*api.GeneratedArtifact {
	if r == nil || !kind.Valid() {
		return nil
	}
	var artifacts []*api.GeneratedArtifact
	switch kind {
	case api.GeneratedArtifactAnonymousStruct:
		for _, binding := range r.anonymousStructs {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactMapSpecialization:
		for _, binding := range r.mapSpecializations {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactInterfaceAdapter:
		for _, binding := range r.interfaceAdapters {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactAnonymousInterface:
		for _, binding := range r.anonymousInterfaces {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactInterfaceMethodToken:
		for _, binding := range r.interfaceMethodTokens {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		for _, binding := range r.interfaceDynamicTypes {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactGenericCapability:
		for _, binding := range r.genericCapabilities {
			artifacts = append(artifacts, binding.owner)
		}
	case api.GeneratedArtifactCallableABI:
		for _, binding := range r.callableABIs {
			artifacts = append(artifacts, binding.owner)
		}
	}
	sort.Slice(artifacts, func(left, right int) bool {
		return artifacts[left].ArtifactKey() < artifacts[right].ArtifactKey()
	})
	return artifacts
}

func (r *Registry) reserve(
	object types.Object,
	binding targetBinding,
) error {
	if r == nil {
		return &api.NameError{Name: objectName(object), Reason: "declaration registry is nil"}
	}
	if existing, ok := r.byObject[object]; ok {
		if existing.sourceFile != binding.sourceFile ||
			existing.sourcePath != binding.sourcePath ||
			existing.name != binding.name ||
			existing.kind != binding.kind {
			return &api.NameError{
				Name:   objectName(object),
				Reason: "declaration has conflicting target ownership",
			}
		}
		return nil
	}
	r.byObject[object] = binding
	return nil
}

func (r *Registry) IndexCompilationTargets(
	sourcePackages []*load.Package,
	environmentPackages []*load.Package,
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
		r.assemblyPathByPackage[typesPackage] = contractPath
		if err := r.indexEnvironmentPackage(
			environmentPackage,
			contractPath,
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
) error {
	scope := sourcePackage.Types().Scope()
	names := scope.Names()
	sort.Strings(names)
	used := make(map[string]struct{}, len(names))
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
		if err := r.reserve(object, binding); err != nil {
			return err
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
			if err := r.reserve(method, targetBinding{
				name:         methodName,
				sourcePath:   contractPath,
				moduleExport: true,
				kind:         targetBindingEnvironment,
			}); err != nil {
				return err
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
