package naming

import (
	"go/ast"
	"go/types"
	"strconv"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type File struct {
	owner           *Owner
	sourceFile      *ast.File
	packageScope    *types.Scope
	factory         tsgo.Factory
	targetPath      string
	observer        EnvironmentObserver
	temporaries     map[api.TemporaryKind]uint64
	importNames     map[string]struct{}
	importAliases   map[types.Object]string
	derivedImports  map[string]string
	projections     map[constantProjectionImport]string
	primitives      map[api.PrimitiveAlias]string
	tsonicCore      map[tsoniccore.Symbol]string
	runtime         map[api.RuntimeSymbol]string
	providerImports map[string]providerImport
	artifactOwner   api.ArtifactOwner
	artifactSource  ast.Node
	artifactFile    *ast.File
	artifactPath    string
}

type TemporarySnapshot map[api.TemporaryKind]uint64

func (n *File) sourceReferencePath(
	object types.Object,
	binding targetBinding,
) (string, bool, error) {
	if object == nil || binding.sourcePath == "" {
		return "", false, &api.NameError{
			Name:   objectName(object),
			Reason: "source reference identity is invalid",
		}
	}
	if n.packageScope == nil {
		return binding.sourcePath, object.Pkg() != nil, nil
	}
	crossPackage := n.packageScope != nil &&
		object.Pkg() != nil &&
		object.Pkg().Scope() != n.packageScope
	if !crossPackage {
		return binding.sourcePath, false, nil
	}
	if !object.Exported() {
		return binding.sourcePath, true, nil
	}
	referencePath := n.owner.registry.assemblyPathByPackage[object.Pkg()]
	if referencePath == "" {
		return "", false, &api.NameError{
			Name:   object.Name(),
			Reason: "cross-package declaration has no assembly path",
		}
	}
	return referencePath, true, nil
}

func (n *Owner) ForFile(
	sourceFile *ast.File,
	packageScope *types.Scope,
	factory tsgo.Factory,
	targetPath string,
	observer EnvironmentObserver,
) (api.Names, error) {
	if observer == nil {
		return nil, &api.NameError{
			Reason: "file name owner requires a non-optional environment observer",
		}
	}
	return &File{
		owner:           n,
		sourceFile:      sourceFile,
		packageScope:    packageScope,
		factory:         factory,
		targetPath:      targetPath,
		observer:        observer,
		temporaries:     make(map[api.TemporaryKind]uint64),
		importNames:     make(map[string]struct{}),
		importAliases:   make(map[types.Object]string),
		derivedImports:  make(map[string]string),
		projections:     make(map[constantProjectionImport]string),
		primitives:      make(map[api.PrimitiveAlias]string),
		tsonicCore:      make(map[tsoniccore.Symbol]string),
		runtime:         make(map[api.RuntimeSymbol]string),
		providerImports: make(map[string]providerImport),
	}, nil
}

func (n *File) Declare(object types.Object) (string, error) {
	if n.owner.registry != nil {
		if binding, ok := n.owner.registry.byObject[object]; ok &&
			binding.scheduled() {
			return binding.name, nil
		}
	}
	if object != nil && object.Parent() == n.packageScope {
		binding, ok := n.owner.byObject[object]
		if !ok && n.owner.registry != nil {
			binding, ok = n.owner.registry.byObject[object]
		}
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

func (n *File) Parameter(parameter *types.Var, index int) (string, error) {
	switch {
	case parameter == nil:
		return "", &api.NameError{Reason: "parameter object is nil"}
	case index < 0:
		return "", &api.NameError{Reason: "parameter index is negative"}
	case parameter.Name() != "" && parameter.Name() != "_":
		return n.Declare(parameter)
	default:
		return "$" + strconv.Itoa(index), nil
	}
}

func (n *File) Result(result *types.Var, index int) (string, error) {
	switch {
	case result == nil:
		return "", &api.NameError{Reason: "result object is nil"}
	case index < 0:
		return "", &api.NameError{Reason: "result index is negative"}
	case result.Name() != "" && result.Name() != "_":
		return n.Declare(result)
	default:
		return "$result" + strconv.Itoa(index), nil
	}
}

func (n *File) Reference(object types.Object) (api.NameReference, error) {
	facet, err := valueReferenceFacet(object)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.reference(object, api.ImportPhaseValue, facet)
}

func (n *File) TypeReference(
	object types.Object,
) (api.NameReference, error) {
	if _, ok := object.(*types.TypeName); !ok {
		return api.NameReference{}, &api.NameError{
			Name:   objectName(object),
			Reason: "type reference object is not a type declaration",
		}
	}
	return n.reference(
		object,
		api.ImportPhaseType,
		api.ArtifactFacetInstanceTypeSurface,
	)
}

func (n *File) reference(
	object types.Object,
	phase api.ImportPhase,
	facet api.ArtifactFacet,
) (api.NameReference, error) {
	if object == nil {
		return api.NameReference{}, &api.NameError{Reason: "reference object is nil"}
	}
	if function, ok := object.(*types.Func); ok {
		object = function.Origin()
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
	if typeName, ok := object.(*types.TypeName); ok {
		reference, profiled, err := n.providerStatefulRepresentation(
			typeName,
			phase,
			facet,
		)
		if err != nil || profiled {
			return reference, err
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
	if binding.kind == targetBindingMissingProvider {
		contract, err := environmentcontract.Describe(object)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NameReference{}, &api.NameError{
			Name:   contract.Identity(),
			Reason: "selected standard-library declaration has no provider binding",
		}
	}
	if binding.kind == targetBindingProvider {
		if binding.providerMember != "" {
			return api.NameReference{}, &api.NameError{
				Name:   object.Name(),
				Reason: "provider member requires method-target selection",
			}
		}
		if err := n.requireUse(object, referenceDemand(object, phase)); err != nil {
			return api.NameReference{}, err
		}
		qualifier, request, err := n.providerImport(
			binding.providerModule,
			phase,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewProviderQualifiedNameReference(
			qualifier,
			binding.providerExport,
			request,
		)
	}
	if binding.scheduled() {
		if err := n.requireUse(object, referenceDemand(object, phase)); err != nil {
			return api.NameReference{}, err
		}
	}
	var requests []api.RootRequest
	if binding.sourceOwned() && n.artifactOwner.Valid() {
		request, err := api.NewArtifactDependencyRequest(object, facet)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, request)
	}
	if binding.scheduled() && binding.sourcePath != n.targetPath {
		referencePath, crossPackage, err := n.sourceReferencePath(
			object,
			binding,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		modulePath, err := output.ModuleSpecifier(n.targetPath, referencePath)
		if err != nil {
			return api.NameReference{}, err
		}
		localName := binding.name
		if crossPackage {
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
		requests = append(requests, request)
		return api.NewNameReference(localName, requests...)
	}
	return api.NewNameReference(binding.name, requests...)
}

func (n *File) PackageVariable(
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
	target := n.owner.registry.byObject[variable]
	if target.kind == targetBindingMissingProvider {
		return api.PackageVariableReference{}, &api.NameError{
			Name:   variable.Name(),
			Reason: "selected standard-library variable has no provider binding",
		}
	}
	if target.kind == targetBindingProvider {
		if target.providerAccess != gostdlib.AccessStateMember {
			return api.PackageVariableReference{}, &api.NameError{
				Name:   variable.Name(),
				Reason: "provider variable has invalid package-state access",
			}
		}
		if err := n.requireUse(
			variable,
			environmentcontract.UseDemandState,
		); err != nil {
			return api.PackageVariableReference{}, err
		}
		qualifier, request, err := n.providerImport(
			target.providerModule,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.PackageVariableReference{}, err
		}
		return api.NewProviderQualifiedPackageVariableReference(
			qualifier,
			target.providerExport,
			target.providerMember,
			request,
		)
	}
	if err := n.requireUse(variable, environmentcontract.UseDemandState); err != nil {
		return api.PackageVariableReference{}, err
	}
	var requests []api.RootRequest
	if target.sourceOwned() && n.artifactOwner.Valid() {
		request, err := api.NewArtifactDependencyRequest(
			variable,
			api.ArtifactFacetValueSurface,
		)
		if err != nil {
			return api.PackageVariableReference{}, err
		}
		requests = append(requests, request)
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
			requests...,
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
	requests = append(requests, request)
	return api.NewPackageVariableReference(
		stateName,
		binding.fieldName,
		requests...,
	)
}

func (n *File) BeginArtifact(
	owner api.ArtifactOwner,
	source ast.Node,
	sourceFile *ast.File,
	sourcePath string,
) (func(), error) {
	if !owner.Valid() {
		return nil, &api.NameError{Reason: "artifact owner is invalid"}
	}
	if n.artifactOwner.Valid() {
		return nil, &api.NameError{
			Name:   owner.Name(),
			Reason: "artifact emission is already active",
		}
	}
	sourceOwner, sourceOwned := owner.Source()
	_, initializer, initializerOwned := owner.PackageInitializer()
	generatedOwner, generatedOwned := owner.Generated()
	switch {
	case sourceOwned:
		if source == nil ||
			sourceFile == nil ||
			sourcePath == "" ||
			sourceOwner.Pos() < source.Pos() ||
			sourceOwner.Pos() > source.End() ||
			source.Pos() < sourceFile.Pos() ||
			source.End() > sourceFile.End() {
			return nil, &api.NameError{
				Name:   owner.Name(),
				Reason: "source artifact has no exact declaration anchor",
			}
		}
	case initializerOwned:
		if source == nil ||
			sourceFile == nil ||
			sourcePath == "" ||
			initializer.Rhs.Pos() < source.Pos() ||
			initializer.Rhs.End() > source.End() ||
			source.Pos() < sourceFile.Pos() ||
			source.End() > sourceFile.End() {
			return nil, &api.NameError{
				Name:   owner.Name(),
				Reason: "package initializer artifact has no exact declaration anchor",
			}
		}
	case generatedOwned:
		if source != nil || sourceFile != nil || sourcePath != "" ||
			generatedOwner.Placement() !=
				api.GeneratedArtifactPlacementCompilation {
			return nil, &api.NameError{
				Name:   owner.Name(),
				Reason: "generated artifact received a source declaration",
			}
		}
	default:
		return nil, &api.NameError{Reason: "artifact owner is invalid"}
	}
	n.artifactOwner = owner
	n.artifactSource = source
	n.artifactFile = sourceFile
	n.artifactPath = sourcePath
	return func() {
		n.artifactOwner = api.ArtifactOwner{}
		n.artifactSource = nil
		n.artifactFile = nil
		n.artifactPath = ""
	}, nil
}

func (n *File) SnapshotTemporaries() TemporarySnapshot {
	snapshot := make(TemporarySnapshot, len(n.temporaries))
	for kind, value := range n.temporaries {
		snapshot[kind] = value
	}
	return snapshot
}

func (n *File) RestoreTemporaries(snapshot TemporarySnapshot) {
	n.temporaries = make(map[api.TemporaryKind]uint64, len(snapshot))
	for kind, value := range snapshot {
		n.temporaries[kind] = value
	}
}

func valueReferenceFacet(object types.Object) (api.ArtifactFacet, error) {
	switch object.(type) {
	case *types.Func, *types.Builtin:
		return api.ArtifactFacetCallableSignature, nil
	case *types.TypeName:
		return api.ArtifactFacetConstructorSurface, nil
	case *types.Const, *types.Var:
		return api.ArtifactFacetValueSurface, nil
	default:
		return api.ArtifactFacetInvalid, &api.NameError{
			Name:   objectName(object),
			Reason: "value reference has no observable artifact facet",
		}
	}
}

func (n *File) Member(field *types.Var) (string, error) {
	if field == nil {
		return "", &api.NameError{Reason: "field object is nil"}
	}
	name := n.owner.memberNameByObject[field]
	if name == "" {
		if !field.IsField() || field.Name() == "_" {
			return "", &api.NameError{
				Name:   field.Name(),
				Reason: "field object has no target member identity",
			}
		}
		name = portableIdentifier(field.Name())
	}
	return name, nil
}

func (n *File) importName(
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

func (n *File) packageImportQualifier(
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

func (n *File) allocateImportName(preferred string, qualifier string) string {
	base := preferred + "__from_" + qualifier
	candidate := base
	for suffix := uint64(1); n.sourceNameExists(candidate) ||
		n.hasImportName(candidate); suffix++ {
		candidate = base + "_" + strconv.FormatUint(suffix, 10)
	}
	n.importNames[candidate] = struct{}{}
	return candidate
}

func (n *File) sourceNameExists(name string) bool {
	return (n.packageScope != nil && n.packageScope.Lookup(name) != nil) ||
		n.owner.hasSourceName(name)
}

func (n *File) hasImportName(name string) bool {
	_, exists := n.importNames[name]
	return exists
}

func (n *Owner) hasSourceName(name string) bool {
	_, exists := n.sourceNameBases[name]
	return exists
}

func (n *File) Temporary(kind api.TemporaryKind) (string, error) {
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
