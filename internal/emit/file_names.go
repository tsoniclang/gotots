package emit

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type fileNames struct {
	owner            *nameOwner
	sourceFile       *ast.File
	packageScope     *types.Scope
	factory          tsgo.Factory
	targetPath       string
	require          func(types.Object) error
	temporaries      map[api.TemporaryKind]uint64
	importNames      map[string]struct{}
	importAliases    map[types.Object]string
	companionAliases map[api.DeclarationRequirement]string
	primitives       map[api.PrimitiveAlias]string
}

func (n *nameOwner) ForFile(
	sourceFile *ast.File,
	packageScope *types.Scope,
	factory tsgo.Factory,
	targetPath string,
	require func(types.Object) error,
) api.Names {
	return &fileNames{
		owner:            n,
		sourceFile:       sourceFile,
		packageScope:     packageScope,
		factory:          factory,
		targetPath:       targetPath,
		require:          require,
		temporaries:      make(map[api.TemporaryKind]uint64),
		importNames:      make(map[string]struct{}),
		importAliases:    make(map[types.Object]string),
		companionAliases: make(map[api.DeclarationRequirement]string),
		primitives:       make(map[api.PrimitiveAlias]string),
	}
}

func (n *fileNames) Declare(object types.Object) (string, error) {
	if object != nil && object.Parent() == n.packageScope {
		binding, ok := n.owner.byObject[object]
		if !ok {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "package declaration was not reserved",
			}
		}
		return binding.name, nil
	}
	return n.owner.declare(object, targetBinding{})
}

func (n *fileNames) Parameter(parameter *types.Var, index int) (string, error) {
	switch {
	case parameter == nil:
		return "", &api.NameError{Reason: "parameter object is nil"}
	case index < 0:
		return "", &api.NameError{Reason: "parameter index is negative"}
	case parameter.Name() != "":
		return n.Declare(parameter)
	}
	for suffix := uint64(0); ; suffix++ {
		candidate := "__gotots_parameter_" + strconv.Itoa(index)
		if suffix != 0 {
			candidate += "_" + strconv.FormatUint(suffix, 10)
		}
		if _, reserved := n.owner.sourceNameBases[candidate]; !reserved {
			return candidate, nil
		}
	}
}

func (n *fileNames) Reference(object types.Object) (api.NameReference, error) {
	return n.reference(object, api.ImportPhaseValue)
}

func (n *fileNames) TypeReference(
	object types.Object,
) (api.NameReference, error) {
	return n.reference(object, api.ImportPhaseType)
}

func (n *fileNames) reference(
	object types.Object,
	phase api.ImportPhase,
) (api.NameReference, error) {
	if object == nil {
		return api.NameReference{}, &api.NameError{Reason: "reference object is nil"}
	}
	if variable, ok := object.(*types.Var); ok &&
		!variable.IsField() &&
		variable.Pkg() != nil &&
		variable.Parent() == variable.Pkg().Scope() {
		return api.NameReference{}, &api.NameError{
			Name:   variable.Name(),
			Reason: "package variable requires package-state reference",
		}
	}
	binding, ok := n.owner.byObject[object]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[object]
	}
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name:   object.Name(),
			Reason: "object has no emitted declaration",
		}
	}
	if binding.sourceFile != nil && n.require != nil {
		if err := n.require(object); err != nil {
			return api.NameReference{}, err
		}
	}
	if binding.sourceFile != nil && binding.sourceFile != n.sourceFile {
		referencePath := binding.sourcePath
		if object.Pkg() != nil && object.Pkg().Scope() != n.packageScope {
			referencePath = n.owner.registry.assemblyPathByPackage[object.Pkg()]
			if referencePath == "" {
				return api.NameReference{}, &api.NameError{
					Name:   object.Name(),
					Reason: "cross-package declaration has no assembly path",
				}
			}
		}
		modulePath, err := output.ModuleSpecifier(n.targetPath, referencePath)
		if err != nil {
			return api.NameReference{}, err
		}
		localName := binding.name
		if object.Parent() != n.packageScope {
			localName, err = n.importName(object, binding.name)
			if err != nil {
				return api.NameReference{}, err
			}
		}
		request, err := api.NewImportRequest(
			n.factory,
			phase,
			modulePath,
			binding.name,
			localName,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(localName, request)
	}
	return api.NewNameReference(binding.name)
}

func (n *fileNames) PackageVariable(
	variable *types.Var,
) (api.PackageVariableReference, error) {
	if variable == nil {
		return api.PackageVariableReference{},
			&api.NameError{Reason: "package variable is nil"}
	}
	binding, ok := n.owner.registry.packageVariables[variable]
	if !ok {
		return api.PackageVariableReference{}, &api.NameError{
			Name:   variable.Name(),
			Reason: "package variable has no state ownership",
		}
	}
	if n.require != nil {
		if err := n.require(variable); err != nil {
			return api.PackageVariableReference{}, err
		}
	}
	targetPath := binding.statePath
	stateName := "$state"
	if variable.Parent() != n.packageScope {
		qualifier, err := n.packageImportQualifier(variable.Pkg())
		if err != nil {
			return api.PackageVariableReference{}, err
		}
		targetPath = binding.assemblyPath
		stateName = "$state__" + qualifier
	}
	if n.targetPath == targetPath {
		return api.NewPackageVariableReference(
			stateName,
			binding.fieldName,
		)
	}
	modulePath, err := output.ModuleSpecifier(n.targetPath, targetPath)
	if err != nil {
		return api.PackageVariableReference{}, err
	}
	request, err := api.NewImportRequest(
		n.factory,
		api.ImportPhaseValue,
		modulePath,
		"$state",
		stateName,
	)
	if err != nil {
		return api.PackageVariableReference{}, err
	}
	return api.NewPackageVariableReference(
		stateName,
		binding.fieldName,
		request,
	)
}

func (n *fileNames) Companion(
	typeName *types.TypeName,
	operation api.CompanionOperation,
) (api.NameReference, error) {
	if typeName == nil {
		return api.NameReference{}, &api.NameError{
			Reason: "companion type is nil",
		}
	}
	binding, ok := n.owner.byObject[typeName]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[typeName]
	}
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name:   typeName.Name(),
			Reason: "companion type has no emitted declaration",
		}
	}
	if binding.sourceFile != nil && n.require != nil {
		if err := n.require(typeName); err != nil {
			return api.NameReference{}, err
		}
	}
	companion, err := api.NewNamedStructCompanionRequirement(typeName, operation)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewCompanionRequest(typeName, operation)
	if err != nil {
		return api.NameReference{}, err
	}
	exportedName, err := api.CompanionExportName(binding.name, operation)
	if err != nil {
		return api.NameReference{}, err
	}
	if binding.sourceFile == nil || binding.sourceFile == n.sourceFile {
		return api.NewNameReference(exportedName, request)
	}
	referencePath := binding.sourcePath
	if typeName.Pkg().Scope() != n.packageScope {
		referencePath = n.owner.registry.assemblyPathByPackage[typeName.Pkg()]
		if referencePath == "" {
			return api.NameReference{}, &api.NameError{
				Name:   typeName.Name(),
				Reason: "cross-package companion has no assembly path",
			}
		}
	}
	modulePath, err := output.ModuleSpecifier(n.targetPath, referencePath)
	if err != nil {
		return api.NameReference{}, err
	}
	localName := n.companionAliases[companion]
	if localName == "" {
		qualifier, err := n.packageImportQualifier(typeName.Pkg())
		if err != nil {
			return api.NameReference{}, err
		}
		localName = n.allocateImportName(exportedName, qualifier)
		n.companionAliases[companion] = localName
	}
	importRequest, err := api.NewImportRequest(
		n.factory,
		api.ImportPhaseValue,
		modulePath,
		exportedName,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(localName, request, importRequest)
}

func (n *fileNames) Member(field *types.Var) (string, error) {
	if field == nil {
		return "", &api.NameError{Reason: "field object is nil"}
	}
	name := n.owner.memberNameByObject[field]
	if name == "" {
		return "", &api.NameError{
			Name:   field.Name(),
			Reason: "field object was not indexed from a named struct",
		}
	}
	return name, nil
}

func (n *fileNames) importName(
	object types.Object,
	preferred string,
) (string, error) {
	if existing := n.importAliases[object]; existing != "" {
		return existing, nil
	}
	qualifier, err := n.packageImportQualifier(object.Pkg())
	if err != nil {
		return "", err
	}
	candidate := n.allocateImportName(preferred, qualifier)
	n.importAliases[object] = candidate
	return candidate, nil
}

func (n *fileNames) packageImportQualifier(
	sourcePackage *types.Package,
) (string, error) {
	if sourcePackage == nil {
		return "", &api.NameError{Reason: "import package is nil"}
	}
	qualifier := n.owner.registry.importQualifierByPackage[sourcePackage]
	if qualifier == "" {
		return "", &api.NameError{
			Name:   sourcePackage.Path(),
			Reason: "import package has no preallocated qualifier",
		}
	}
	return qualifier, nil
}

func (n *fileNames) allocateImportName(preferred string, qualifier string) string {
	base := preferred + "__from_" + qualifier
	candidate := base
	for suffix := uint64(1); n.packageScope.Lookup(candidate) != nil ||
		n.hasImportName(candidate) ||
		n.owner.hasSourceName(candidate); suffix++ {
		candidate = base + "_" + strconv.FormatUint(suffix, 10)
	}
	n.importNames[candidate] = struct{}{}
	return candidate
}

func (n *fileNames) hasImportName(name string) bool {
	_, exists := n.importNames[name]
	return exists
}

func (n *nameOwner) hasSourceName(name string) bool {
	_, exists := n.sourceNameBases[name]
	return exists
}

func (n *fileNames) Primitive(alias api.PrimitiveAlias) (api.NameReference, error) {
	if existing := n.primitives[alias]; existing != "" {
		modulePath, err := output.ModuleSpecifier(
			n.targetPath,
			output.ScalarSupportPath,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		request, err := api.NewPrimitiveAliasRequest(
			n.factory,
			modulePath,
			alias,
			existing,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(existing, request)
	}
	exportedName, err := api.PrimitiveAliasName(alias)
	if err != nil {
		return api.NameReference{}, err
	}
	localName := exportedName
	if n.packageScope.Lookup(localName) != nil ||
		n.owner.hasSourceName(localName) ||
		n.hasImportName(localName) {
		base := exportedName + "__from_gotots_support"
		localName = base
		for suffix := uint64(1); n.packageScope.Lookup(localName) != nil ||
			n.owner.hasSourceName(localName) ||
			n.hasImportName(localName); suffix++ {
			localName = base + "_" + strconv.FormatUint(suffix, 10)
		}
	}
	n.importNames[localName] = struct{}{}
	n.primitives[alias] = localName
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		output.ScalarSupportPath,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewPrimitiveAliasRequest(
		n.factory,
		modulePath,
		alias,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(localName, request)
}

func (n *fileNames) Temporary(kind api.TemporaryKind) (string, error) {
	prefix, err := api.TemporaryPrefix(kind)
	if err != nil {
		return "", err
	}
	for {
		index := n.temporaries[kind]
		n.temporaries[kind] = index + 1
		candidate := prefix + strconv.FormatUint(index, 10)
		if _, reserved := n.owner.sourceNameBases[candidate]; reserved {
			continue
		}
		return candidate, nil
	}
}

func (n *fileNames) ModuleExport(object types.Object) (bool, error) {
	if object == nil {
		return false, &api.NameError{Reason: "declaration object is nil"}
	}
	binding, ok := n.owner.byObject[object]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[object]
	}
	if !ok {
		return false, &api.NameError{
			Name:   object.Name(),
			Reason: "object has no emitted declaration",
		}
	}
	return binding.moduleExport, nil
}
