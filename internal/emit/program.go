package emit

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type TargetFile struct {
	outputPath  string
	packageName string
	sourceFile  tsgo.SourceFile
	kind        TargetFileKind
}

type TargetFileKind uint8

const (
	TargetFileInvalid TargetFileKind = iota
	TargetFileSource
	TargetFilePackageState
	TargetFilePackageAssembly
	TargetFileProgramInitialization
	TargetFileSupport
)

type ProgramEmission struct {
	files []TargetFile
}

type programSession struct {
	source                   *load.Program
	factory                  tsgo.Factory
	integer                  api.IntegerRepresentation
	registry                 *declarationRegistry
	scheduler                *scheduler
	requirements             *declarationRequirementScheduler
	artifacts                *artifactstate.Graph
	sites                    map[types.Object]declarationSite
	emitters                 map[*load.Package]*emitter
	builders                 map[string]*targetFileBuilder
	packageBuilders          map[*load.Package]*packageTargetBuilder
	packageInitializations   *packageInitializationScheduler
	packageInitializerOwners map[types.Object]struct{}
	sealed                   bool
}

type targetDeclaration struct {
	owner           api.ArtifactOwner
	name            string
	position        token.Pos
	statements      []tsgo.Statement
	placement       *placementOwner
	temporaryStart  temporarySnapshot
	reconstructions uint64
}

type targetFileBuilder struct {
	sourcePackage *load.Package
	sourceFile    load.File
	outputPath    string
	emitter       *emitter
	context       api.Context
	placement     *placementOwner
	declarations  []targetDeclaration
	byOwner       map[api.ArtifactOwner]struct{}
	indexByOwner  map[api.ArtifactOwner]int
}

func Compile(source *load.Program, roots []Root) (ProgramEmission, error) {
	return CompileWithOptions(source, roots, DefaultOptions())
}

func CompileWithOptions(
	source *load.Program,
	roots []Root,
	options Options,
) (ProgramEmission, error) {
	if err := options.validate(); err != nil {
		return ProgramEmission{}, err
	}
	if source == nil {
		return ProgramEmission{},
			&ScheduleError{Reason: "source program is nil"}
	}
	if len(roots) == 0 {
		return ProgramEmission{},
			&ScheduleError{Reason: "emission roots are empty"}
	}
	session, err := newProgramSession(source, options)
	if err != nil {
		return ProgramEmission{}, err
	}
	for _, root := range roots {
		if !root.valid() {
			return ProgramEmission{},
				&ScheduleError{Reason: "emission root is invalid"}
		}
	}
	orderedRoots := slices.Clone(roots)
	sort.Slice(orderedRoots, func(left, right int) bool {
		return compareRoots(orderedRoots[left], orderedRoots[right]) < 0
	})
	for _, root := range orderedRoots {
		if err := session.requireRoot(root); err != nil {
			return ProgramEmission{}, err
		}
	}
	for {
		if object, ok := session.scheduler.next(); ok {
			if err := session.emit(object); err != nil {
				return ProgramEmission{}, err
			}
			continue
		}
		if requirements, ok := session.requirements.nextBatch(); ok {
			if err := session.applyDeclarationRequirements(requirements); err != nil {
				return ProgramEmission{}, err
			}
			continue
		}
		if object, ok := session.artifacts.NextDirty(); ok {
			if err := session.reconstructScheduledArtifact(object); err != nil {
				return ProgramEmission{}, err
			}
			continue
		}
		if sourcePackage, ok := session.packageInitializations.next(); ok {
			if err := session.emitPackageInitialization(sourcePackage); err != nil {
				return ProgramEmission{}, err
			}
			continue
		}
		break
	}
	files, err := session.targetFiles()
	if err != nil {
		return ProgramEmission{}, err
	}
	if err := session.verifyRootObligations(orderedRoots, files); err != nil {
		return ProgramEmission{}, err
	}
	return ProgramEmission{files: files}, nil
}

func CompileFile(
	sourcePackage *load.Package,
	sourceFile *ast.File,
) (ProgramEmission, error) {
	if sourcePackage == nil || sourcePackage.Program() == nil {
		return ProgramEmission{}, &ScheduleError{Reason: "source package is nil"}
	}
	_, ok := sourcePackage.FileForSyntax(sourceFile)
	if !ok {
		return ProgramEmission{}, &ScheduleError{Reason: "source file is not package-owned"}
	}
	roots, err := fileRoots(sourcePackage, sourceFile)
	if err != nil {
		return ProgramEmission{}, err
	}
	return Compile(sourcePackage.Program(), roots)
}

func fileRoots(
	sourcePackage *load.Package,
	sourceFile *ast.File,
) ([]Root, error) {
	var roots []Root
	for _, declaration := range sourceFile.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if isPackageInitDeclaration(declaration) {
				continue
			}
			object := sourcePackage.TypesInfo().Defs[declaration.Name]
			if object == nil {
				return nil, &ScheduleError{
					Object: declaration.Name.Name,
					Reason: "function declaration has no object identity",
				}
			}
			root, err := newRoot(object, RootFileCoverage)
			if err != nil {
				return nil, err
			}
			roots = append(roots, root)
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						object := sourcePackage.TypesInfo().Defs[name]
						if object == nil || object.Name() == "_" {
							continue
						}
						root, err := newRoot(object, RootFileCoverage)
						if err != nil {
							return nil, err
						}
						roots = append(roots, root)
					}
				case *ast.TypeSpec:
					object := sourcePackage.TypesInfo().Defs[spec.Name]
					if object == nil {
						continue
					}
					root, err := newRoot(object, RootFileCoverage)
					if err != nil {
						return nil, err
					}
					roots = append(roots, root)
				}
			}
		}
	}
	return roots, nil
}

func compareObjects(left types.Object, right types.Object) int {
	leftPackage := ""
	if left != nil && left.Pkg() != nil {
		leftPackage = left.Pkg().Path()
	}
	rightPackage := ""
	if right != nil && right.Pkg() != nil {
		rightPackage = right.Pkg().Path()
	}
	switch {
	case leftPackage < rightPackage:
		return -1
	case leftPackage > rightPackage:
		return 1
	case left.Pos() < right.Pos():
		return -1
	case left.Pos() > right.Pos():
		return 1
	case left.Name() < right.Name():
		return -1
	case left.Name() > right.Name():
		return 1
	default:
		return 0
	}
}

func compareArtifactOwners(
	left api.ArtifactOwner,
	right api.ArtifactOwner,
) int {
	leftSource, leftIsSource := left.Source()
	rightSource, rightIsSource := right.Source()
	switch {
	case leftIsSource && rightIsSource:
		return compareObjects(leftSource, rightSource)
	case leftIsSource:
		return -1
	case rightIsSource:
		return 1
	}
	leftGenerated, leftOK := left.Generated()
	rightGenerated, rightOK := right.Generated()
	if !leftOK || !rightOK {
		switch {
		case leftOK:
			return 1
		case rightOK:
			return -1
		default:
			return 0
		}
	}
	switch {
	case leftGenerated.ArtifactKey() < rightGenerated.ArtifactKey():
		return -1
	case leftGenerated.ArtifactKey() > rightGenerated.ArtifactKey():
		return 1
	default:
		return 0
	}
}

func (e ProgramEmission) Files() []TargetFile {
	return slices.Clone(e.files)
}

func (f TargetFile) OutputPath() string {
	return f.outputPath
}

func (f TargetFile) PackageName() string {
	return f.packageName
}

func (f TargetFile) SourceFile() tsgo.SourceFile {
	return f.sourceFile
}

func (f TargetFile) Kind() TargetFileKind {
	return f.kind
}

func newProgramSession(
	source *load.Program,
	options Options,
) (*programSession, error) {
	sites, err := indexDeclarations(source)
	if err != nil {
		return nil, err
	}
	registry := newDeclarationRegistry()
	if err := registry.indexPackageTargets(source.Packages()); err != nil {
		return nil, err
	}
	session := &programSession{
		source:                   source,
		factory:                  tsgo.NewFactory(),
		integer:                  options.IntegerRepresentation,
		registry:                 registry,
		scheduler:                newScheduler(),
		requirements:             newDeclarationRequirementScheduler(),
		artifacts:                artifactstate.NewGraph(compareArtifactOwners),
		sites:                    sites,
		emitters:                 make(map[*load.Package]*emitter),
		builders:                 make(map[string]*targetFileBuilder),
		packageBuilders:          make(map[*load.Package]*packageTargetBuilder),
		packageInitializations:   newPackageInitializationScheduler(),
		packageInitializerOwners: make(map[types.Object]struct{}),
	}
	for _, sourcePackage := range source.Packages() {
		for _, initializer := range sourcePackage.TypesInfo().InitOrder {
			owner, ok := packageInitializerOwner(initializer)
			if !ok {
				return nil, &ScheduleError{
					Object: sourcePackage.Path(),
					Reason: "package initializer has no exact source owner",
				}
			}
			if _, indexed := sites[owner]; !indexed {
				return nil, &ScheduleError{
					Object: owner.Name(),
					Reason: "package initializer source owner is not indexed",
				}
			}
			session.packageInitializerOwners[owner] = struct{}{}
		}
	}
	for _, sourcePackage := range source.Packages() {
		session.emitters[sourcePackage] = newEmitter(
			sourcePackage,
			session.factory,
			session.registry,
			options.IntegerRepresentation,
			options.EvaluationOrder,
			session.require,
		)
	}
	orderedSites := make([]declarationSite, 0, len(sites))
	for _, site := range sites {
		orderedSites = append(orderedSites, site)
	}
	sort.Slice(orderedSites, func(left, right int) bool {
		return compareDeclarationSites(orderedSites[left], orderedSites[right]) < 0
	})
	initializers := make(map[*load.Package][]types.Object)
	for _, site := range orderedSites {
		function, ok := site.declaration.(*ast.FuncDecl)
		if ok && isPackageInitDeclaration(function) {
			initializers[site.source] = append(
				initializers[site.source],
				site.object,
			)
		}
	}
	for sourcePackage, objects := range initializers {
		emitter := session.emitters[sourcePackage]
		if emitter == nil {
			return nil, &ScheduleError{
				Object: "init",
				Reason: "package initializer has no emitter",
			}
		}
		if err := emitter.names.preallocatePackageInitializers(objects); err != nil {
			return nil, err
		}
	}
	for _, site := range orderedSites {
		emitter := session.emitters[site.source]
		if emitter == nil {
			return nil, &ScheduleError{
				Object: site.object.Name(),
				Reason: "declaration package has no emitter",
			}
		}
		if variable, ok := site.object.(*types.Var); ok {
			if variable.Name() == "_" {
				continue
			}
			assemblyPath, err := targetoutput.PackageAssemblyPath(site.source)
			if err != nil {
				return nil, err
			}
			if _, err := emitter.names.ReservePackageVariable(
				variable,
				site.outputPath,
				assemblyPath,
			); err != nil {
				return nil, err
			}
		} else {
			if _, err := emitter.names.Reserve(
				site.object,
				site.sourceFile.Syntax(),
				site.outputPath,
			); err != nil {
				return nil, err
			}
		}
	}
	return session, nil
}

func (s *programSession) require(object types.Object) error {
	if s.sealed {
		objectName := ""
		if object != nil {
			objectName = object.Name()
		}
		return &ScheduleError{
			Object: objectName,
			Reason: "declaration requested after target files were sealed",
		}
	}
	if object == nil {
		return &ScheduleError{Reason: "referenced object is nil"}
	}
	if _, ok := s.sites[object]; !ok {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "object has no supported source declaration",
		}
	}
	sourcePackage := s.source.PackageForTypes(object.Pkg())
	if sourcePackage == nil {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "object package has no source owner",
		}
	}
	if err := s.requirePackage(sourcePackage); err != nil {
		return err
	}
	s.scheduler.enqueue(object)
	return nil
}

func (s *programSession) emit(object types.Object) error {
	site, ok := s.sites[object]
	if !ok {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "scheduled object lost its declaration",
		}
	}
	if variable, ok := object.(*types.Var); ok {
		return s.emitPackageStorage(variable, site)
	}
	builder, err := s.builder(site)
	if err != nil {
		return err
	}
	owner := api.MustSourceArtifactOwner(object)
	if _, duplicate := builder.byOwner[owner]; duplicate {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "object was emitted more than once",
		}
	}
	revision, err := s.buildArtifactRevision(
		builder,
		site,
		object,
		nil,
		false,
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
	builder.byOwner[owner] = struct{}{}
	builder.indexByOwner[owner] = len(builder.declarations)
	builder.declarations = append(builder.declarations, targetDeclaration{
		owner:          owner,
		name:           object.Name(),
		position:       object.Pos(),
		statements:     revision.statements,
		placement:      revision.placement,
		temporaryStart: revision.temporaryStart,
	})
	return nil
}

func (s *programSession) applyRootRequests(
	placement *placementOwner,
	requests []api.RootRequest,
) error {
	if s.sealed {
		return &ScheduleError{Reason: "root request arrived after target files were sealed"}
	}
	imports := make([]api.RootRequest, 0, len(requests))
	for _, request := range requests {
		switch request.Kind() {
		case api.RootRequestImport:
			imports = append(imports, request)
		case api.RootRequestDeclarationRequirement:
			requirement, ok := request.DeclarationRequirement()
			if !ok {
				return &ScheduleError{Reason: "declaration requirement is invalid"}
			}
			if err := s.scheduleDeclarationRequirement(requirement); err != nil {
				return err
			}
		case api.RootRequestArtifactDependency:
			return &ScheduleError{
				Reason: "artifact dependency has no reconstructible target owner",
			}
		default:
			return &ScheduleError{Reason: "root request kind is invalid"}
		}
	}
	return placement.Apply(imports)
}

func (s *programSession) builder(site declarationSite) (*targetFileBuilder, error) {
	return s.builderForFile(
		site.source,
		site.sourceFile,
		site.outputPath,
		site.object.Name(),
	)
}

func (s *programSession) builderForFile(
	sourcePackage *load.Package,
	sourceFile load.File,
	outputPath string,
	objectName string,
) (*targetFileBuilder, error) {
	if existing := s.builders[outputPath]; existing != nil {
		return existing, nil
	}
	emitter := s.emitters[sourcePackage]
	if emitter == nil {
		return nil, &ScheduleError{
			Object: objectName,
			Reason: "source package has no emitter",
		}
	}
	context, err := emitter.fileContext(sourceFile.Syntax(), outputPath)
	if err != nil {
		return nil, err
	}
	builder := &targetFileBuilder{
		sourcePackage: sourcePackage,
		sourceFile:    sourceFile,
		outputPath:    outputPath,
		emitter:       emitter,
		context:       context,
		placement:     newPlacementOwner(),
		byOwner:       make(map[api.ArtifactOwner]struct{}),
		indexByOwner:  make(map[api.ArtifactOwner]int),
	}
	s.builders[outputPath] = builder
	return builder, nil
}
