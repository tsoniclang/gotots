package emit

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const packageInitializeName = "$initialize"

type packageInitializationScheduler struct {
	queue   []*load.Package
	pending map[*load.Package]struct{}
	emitted map[*load.Package]struct{}
}

type packageStorage struct {
	owner             api.ArtifactOwner
	variable          *types.Var
	source            ast.Node
	field             tsgo.PropertyDeclaration
	zeroStatements    []tsgo.Statement
	statePlacement    *targetplacement.Owner
	assemblyPlacement *targetplacement.Owner
	reconstructions   uint64
}

type packageStorageRevision struct {
	field             tsgo.PropertyDeclaration
	zeroStatements    []tsgo.Statement
	statePlacement    *targetplacement.Owner
	assemblyPlacement *targetplacement.Owner
	dependencies      []api.ArtifactDependency
	contract          artifactstate.Contract
}

type packageInitializationArtifact struct {
	owner           api.ArtifactOwner
	initializer     *types.Initializer
	site            declarationSite
	statements      []tsgo.Statement
	placement       *targetplacement.Owner
	temporaryStart  emitnaming.TemporarySnapshot
	reconstructions uint64
}

type packageInitFunction struct {
	function *types.Func
	name     string
}

type packageTargetBuilder struct {
	sourcePackage      *load.Package
	statePath          string
	assemblyPath       string
	emitter            *emitter
	stateContext       api.Context
	assemblyContext    api.Context
	statePlacement     *targetplacement.Owner
	assemblyPlacement  *targetplacement.Owner
	storage            []packageStorage
	storageByObject    map[*types.Var]int
	initialization     []packageInitializationArtifact
	initializerByOwner map[api.ArtifactOwner]int
	initFunctions      []packageInitFunction
}

func newPackageInitializationScheduler() *packageInitializationScheduler {
	return &packageInitializationScheduler{
		pending: make(map[*load.Package]struct{}),
		emitted: make(map[*load.Package]struct{}),
	}
}

func (s *packageInitializationScheduler) enqueue(sourcePackage *load.Package) {
	if _, done := s.emitted[sourcePackage]; done {
		return
	}
	if _, queued := s.pending[sourcePackage]; queued {
		return
	}
	s.pending[sourcePackage] = struct{}{}
	s.queue = append(s.queue, sourcePackage)
}

func (s *packageInitializationScheduler) next() (*load.Package, bool) {
	if len(s.queue) == 0 {
		return nil, false
	}
	sourcePackage := s.queue[0]
	s.queue = s.queue[1:]
	delete(s.pending, sourcePackage)
	s.emitted[sourcePackage] = struct{}{}
	return sourcePackage, true
}

func (s *packageInitializationScheduler) hasPending() bool {
	return len(s.queue) != 0 || len(s.pending) != 0
}

func (s *programSession) requirePackage(sourcePackage *load.Package) error {
	if sourcePackage == nil || sourcePackage.Types() == nil {
		return &ScheduleError{Reason: "required package is nil"}
	}
	if s.packageBuilders[sourcePackage] != nil {
		return nil
	}
	emitter := s.emitters[sourcePackage]
	if emitter == nil {
		return &ScheduleError{
			Object: sourcePackage.Path(),
			Reason: "required package has no source emitter",
		}
	}
	statePath, err := targetoutput.PackageStatePath(sourcePackage)
	if err != nil {
		return err
	}
	assemblyPath, err := targetoutput.PackageAssemblyPath(sourcePackage)
	if err != nil {
		return err
	}
	stateContext, err := emitter.targetContext(nil, statePath)
	if err != nil {
		return err
	}
	assemblyContext, err := emitter.targetContext(nil, assemblyPath)
	if err != nil {
		return err
	}
	builder := &packageTargetBuilder{
		sourcePackage:      sourcePackage,
		statePath:          statePath,
		assemblyPath:       assemblyPath,
		emitter:            emitter,
		stateContext:       stateContext,
		assemblyContext:    assemblyContext,
		statePlacement:     targetplacement.New(),
		assemblyPlacement:  targetplacement.New(),
		storageByObject:    make(map[*types.Var]int),
		initializerByOwner: make(map[api.ArtifactOwner]int),
	}
	s.packageBuilders[sourcePackage] = builder

	imports := slices.Clone(sourcePackage.Types().Imports())
	sort.Slice(imports, func(left, right int) bool {
		return imports[left].Path() < imports[right].Path()
	})
	for _, imported := range imports {
		dependency := s.source.PackageForTypes(imported)
		if dependency == nil && s.goRuntime.Owns(imported) {
			continue
		}
		if dependency == nil &&
			s.source.EnvironmentForTypes(imported) != nil {
			continue
		}
		if dependency == nil || dependency.ModulePath() == "" {
			return &ScheduleError{
				Object: imported.Path(),
				Reason: "package dependency has no source-available assembly",
			}
		}
		if err := s.requirePackage(dependency); err != nil {
			return err
		}
	}

	scope := sourcePackage.Types().Scope()
	names := scope.Names()
	sort.Strings(names)
	for _, name := range names {
		variable, ok := scope.Lookup(name).(*types.Var)
		if !ok {
			continue
		}
		if _, exists := s.sites[variable]; !exists {
			return &ScheduleError{
				Object: variable.Name(),
				Reason: "package variable has no indexed declaration",
			}
		}
		s.scheduler.enqueue(variable)
	}
	s.packageInitializations.enqueue(sourcePackage)
	return nil
}

func (s *programSession) emitPackageStorage(
	variable *types.Var,
	site declarationSite,
) error {
	builder := s.packageBuilders[site.source]
	if builder == nil {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "package variable owner was not reached",
		}
	}
	if _, duplicate := builder.storageByObject[variable]; duplicate {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "package variable storage was emitted more than once",
		}
	}
	source, err := packageVariableSyntax(site, variable)
	if err != nil {
		return err
	}
	owner := api.MustSourceArtifactOwner(variable)
	revision, err := s.buildPackageStorageRevision(
		builder,
		owner,
		site,
		source,
		variable,
	)
	if err != nil {
		return err
	}
	if err := s.artifacts.Commit(
		owner,
		revision.contract,
		revision.dependencies,
	); err != nil {
		return err
	}
	builder.storageByObject[variable] = len(builder.storage)
	builder.storage = append(builder.storage, packageStorage{
		owner:             owner,
		variable:          variable,
		source:            source,
		field:             revision.field,
		zeroStatements:    revision.zeroStatements,
		statePlacement:    revision.statePlacement,
		assemblyPlacement: revision.assemblyPlacement,
	})
	return nil
}

func (s *programSession) buildPackageStorageRevision(
	builder *packageTargetBuilder,
	owner api.ArtifactOwner,
	site declarationSite,
	source ast.Node,
	variable *types.Var,
) (packageStorageRevision, error) {
	stateNames, ok := builder.stateContext.Names().(*emitnaming.File)
	if !ok {
		return packageStorageRevision{}, &ScheduleError{
			Object: owner.Name(),
			Reason: "package storage state has no concrete name owner",
		}
	}
	finishState, err := stateNames.BeginArtifact(
		owner,
		source,
		site.sourceFile.Syntax(),
		site.outputPath,
	)
	if err != nil {
		return packageStorageRevision{}, err
	}
	defer finishState()
	assemblyNames, ok := builder.assemblyContext.Names().(*emitnaming.File)
	if !ok {
		return packageStorageRevision{}, &ScheduleError{
			Object: owner.Name(),
			Reason: "package storage assembly has no concrete name owner",
		}
	}
	finishAssembly, err := assemblyNames.BeginArtifact(
		owner,
		source,
		site.sourceFile.Syntax(),
		site.outputPath,
	)
	if err != nil {
		return packageStorageRevision{}, err
	}
	defer finishAssembly()
	emission, err := packagevariable.EmitStorage(
		builder.stateContext.WithArtifactOwner(owner),
		builder.assemblyContext.WithArtifactOwner(owner),
		builder.emitter,
		source,
		variable,
	)
	if err != nil {
		return packageStorageRevision{}, err
	}
	statePlacement, stateDependencies, err := s.consumeArtifactRequests(
		owner,
		emission.StateRequests(),
	)
	if err != nil {
		return packageStorageRevision{}, err
	}
	if err := statePlacement.RequireTypeOnly(); err != nil {
		return packageStorageRevision{}, err
	}
	assemblyPlacement, assemblyDependencies, err :=
		s.consumeArtifactRequests(
			owner,
			emission.AssemblyRequests(),
		)
	if err != nil {
		return packageStorageRevision{}, err
	}
	contract, err := packageStorageContract(emission.Field())
	if err != nil {
		return packageStorageRevision{}, err
	}
	return packageStorageRevision{
		field:             emission.Field(),
		zeroStatements:    emission.ZeroStatements(),
		statePlacement:    statePlacement,
		assemblyPlacement: assemblyPlacement,
		dependencies: append(
			stateDependencies,
			assemblyDependencies...,
		),
		contract: contract,
	}, nil
}

func packageStorageContract(
	field tsgo.PropertyDeclaration,
) (artifactstate.Contract, error) {
	return artifactstate.ProjectFacet(
		api.ArtifactFacetValueSurface,
		field,
	)
}

func (s *programSession) reconstructPackageStorage(
	owner api.ArtifactOwner,
	variable *types.Var,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package storage reconstructed after target files were sealed",
		}
	}
	site, ok := s.sites[variable]
	if !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package storage lost its source declaration",
		}
	}
	builder := s.packageBuilders[site.source]
	index, found := builder.storageByObject[variable]
	if builder == nil || !found || index >= len(builder.storage) {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package storage was not emitted first",
		}
	}
	storage := &builder.storage[index]
	if storage.owner != owner ||
		storage.variable != variable ||
		storage.source == nil {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package storage identity changed",
		}
	}
	revision, err := s.buildPackageStorageRevision(
		builder,
		owner,
		site,
		storage.source,
		variable,
	)
	if err != nil {
		return err
	}
	if err := s.artifacts.Commit(
		owner,
		revision.contract,
		revision.dependencies,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	storage.field = revision.field
	storage.zeroStatements = revision.zeroStatements
	storage.statePlacement = revision.statePlacement
	storage.assemblyPlacement = revision.assemblyPlacement
	storage.reconstructions++
	return nil
}

func packageVariableSyntax(
	site declarationSite,
	variable *types.Var,
) (ast.Node, error) {
	declaration, ok := site.declaration.(*ast.GenDecl)
	if !ok {
		return nil, &ScheduleError{
			Object: variable.Name(),
			Reason: "package variable declaration is not general",
		}
	}
	for _, sourceSpec := range declaration.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range spec.Names {
			if site.source.TypesInfo().Defs[name] == variable {
				return name, nil
			}
		}
	}
	return nil, &ScheduleError{
		Object: variable.Name(),
		Reason: "package variable syntax is absent from its declaration",
	}
}

func (s *programSession) emitPackageInitialization(
	sourcePackage *load.Package,
) error {
	builder := s.packageBuilders[sourcePackage]
	if builder == nil {
		return &ScheduleError{
			Object: sourcePackage.Path(),
			Reason: "package initialization has no assembly owner",
		}
	}
	expectedStorage := 0
	scope := sourcePackage.Types().Scope()
	for _, name := range scope.Names() {
		variable, ok := scope.Lookup(name).(*types.Var)
		if !ok {
			continue
		}
		expectedStorage++
		if _, emitted := builder.storageByObject[variable]; !emitted {
			return &ScheduleError{
				Object: variable.Name(),
				Reason: "package variable has no state storage",
			}
		}
	}
	if len(builder.storageByObject) != expectedStorage {
		return &ScheduleError{
			Object: sourcePackage.Path(),
			Reason: "package initialization ran before complete state storage",
		}
	}
	for _, initializer := range sourcePackage.TypesInfo().InitOrder {
		if err := s.emitPackageInitializer(builder, initializer); err != nil {
			return err
		}
	}
	return s.emitPackageInitFunctions(builder)
}

func (s *programSession) emitPackageInitFunctions(
	packageBuilder *packageTargetBuilder,
) error {
	for _, sourceFile := range packageBuilder.sourcePackage.Files() {
		for _, declaration := range sourceFile.Syntax().Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !isPackageInitDeclaration(function) {
				continue
			}
			object, ok := packageBuilder.sourcePackage.TypesInfo().
				Defs[function.Name].(*types.Func)
			if !ok {
				return &ScheduleError{
					Object: "init",
					Reason: "package init has no function identity",
				}
			}
			if err := s.require(object); err != nil {
				return err
			}
			binding, ok := s.registry.Target(object)
			if !ok || binding.Name == "" || binding.SourcePath == "" {
				return &ScheduleError{
					Object: "init",
					Reason: "package init has no target artifact binding",
				}
			}
			modulePath, err := targetoutput.ModuleSpecifier(
				packageBuilder.assemblyPath,
				binding.SourcePath,
			)
			if err != nil {
				return err
			}
			request, err := api.NewImportRequest(
				s.factory,
				api.ImportPhaseValue,
				modulePath,
				binding.Name,
				binding.Name,
			)
			if err != nil {
				return err
			}
			if err := packageBuilder.assemblyPlacement.Apply(
				[]api.RootRequest{request},
			); err != nil {
				return err
			}
			packageBuilder.initFunctions = append(
				packageBuilder.initFunctions,
				packageInitFunction{
					function: object.Origin(),
					name:     binding.Name,
				},
			)
		}
	}
	return nil
}
