package emit

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	declarationindex "github.com/tsoniclang/gotots/internal/emit/declaration/index"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/emit/runtime/gocontract"
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
	TargetFileEnvironmentContract
	TargetFileStandardLibraryConstantProjection
)

type ProgramEmission struct {
	files                       []TargetFile
	environmentObligations      []EnvironmentObligation
	externalFunctionObligations []ExternalFunctionObligation
	runtimePackage              RuntimePackage
}

type RuntimePackage struct {
	assembled runtimeemission.Package
}

type declarationSite = declarationindex.Site

type programSession struct {
	source                  *load.Program
	factory                 tsgo.Factory
	integer                 api.IntegerRepresentation
	evaluationOrder         api.EvaluationOrder
	concurrency             api.ConcurrencySemantics
	registry                *emitnaming.Registry
	scheduler               *scheduler
	requirements            *declarationRequirementScheduler
	artifacts               *artifactstate.Graph
	sites                   map[types.Object]declarationSite
	emitters                map[*load.Package]*emitter
	builders                map[string]*targetFileBuilder
	packageBuilders         map[*load.Package]*packageTargetBuilder
	packageExports          *packageExportScheduler
	environmentBuilders     map[*load.Package]*environmentContractBuilder
	packageInitializations  *packageInitializationScheduler
	genericOperations       map[genericOperationIdentity]*api.GenericOperationContract
	genericProfiles         map[genericCallableProfileIdentity]*api.GenericCallableProfile
	classMembers            map[*types.Func]classMemberContribution
	goRuntime               *gocontract.Contract
	runtimePackage          RuntimePackage
	compareArtifactOwners   func(api.ArtifactOwner, api.ArtifactOwner) int
	requirementRemovalOwner api.ArtifactOwner
	standardLibrary         *certify.Certificate
	externalFunctions       map[*types.Func]ExternalFunctionObligation
	sealed                  bool
}

type targetDeclaration struct {
	owner             api.ArtifactOwner
	name              string
	position          token.Pos
	statements        []tsgo.Statement
	placement         *targetplacement.Owner
	eagerDependencies []api.ArtifactOwner
	temporaryStart    emitnaming.TemporarySnapshot
	reconstructions   uint64
}

type targetFileBuilder struct {
	sourcePackage *load.Package
	sourceFile    load.File
	outputPath    string
	emitter       *emitter
	context       api.Context
	placement     *targetplacement.Owner
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
		return compareRoots(
			orderedRoots[left],
			orderedRoots[right],
			session.compareArtifactOwners,
		) < 0
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
		if builders := session.packageExports.nextBatch(); len(builders) != 0 {
			for _, builder := range builders {
				if err := session.publishPackageExports(builder); err != nil {
					return ProgramEmission{}, err
				}
			}
			continue
		}
		if owner, requirements, removed, ok :=
			session.requirements.nextBatch(); ok {
			if err := session.applyDeclarationRequirements(
				owner,
				requirements,
				removed,
			); err != nil {
				return ProgramEmission{}, err
			}
			continue
		}
		if dirty := session.artifacts.DirtyBatch(); len(dirty) != 0 {
			for _, object := range dirty {
				if err := session.reconstructScheduledArtifact(object); err != nil {
					return ProgramEmission{}, err
				}
			}
			continue
		}
		if sourcePackage, ok := session.packageInitializations.next(); ok {
			if err := session.emitPackageInitialization(sourcePackage); err != nil {
				return ProgramEmission{}, err
			}
			continue
		}
		if session.requirements.finalizeRemovals() {
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
	obligations, err := session.environmentObligations()
	if err != nil {
		return ProgramEmission{}, err
	}
	return ProgramEmission{
		files:                       files,
		environmentObligations:      obligations,
		externalFunctionObligations: session.externalFunctionObligations(),
		runtimePackage:              session.runtimePackage,
	}, nil
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
			if declarationindex.IsPackageInitializer(declaration) {
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

func newProgramSession(
	source *load.Program,
	options Options,
) (*programSession, error) {
	sites, err := declarationindex.Program(source)
	if err != nil {
		return nil, err
	}
	registry := emitnaming.NewRegistry()
	if err := registry.IndexCompilationTargets(
		source.Packages(),
		source.EnvironmentPackages(),
		options.StandardLibrary,
	); err != nil {
		return nil, err
	}
	goRuntime, err := gocontract.Resolve(source)
	if err != nil {
		return nil, err
	}
	compareArtifactOwners := sourceArtifactOwnerOrder(sites)
	session := &programSession{
		source:          source,
		factory:         tsgo.NewFactory(),
		integer:         options.IntegerRepresentation,
		evaluationOrder: options.EvaluationOrder,
		concurrency:     options.ConcurrencySemantics,
		registry:        registry,
		scheduler:       newScheduler(),
		requirements: newDeclarationRequirementScheduler(
			compareArtifactOwners,
		),
		artifacts: artifactstate.NewGraph(
			compareArtifactOwners,
		),
		sites:                  sites,
		emitters:               make(map[*load.Package]*emitter),
		builders:               make(map[string]*targetFileBuilder),
		packageBuilders:        make(map[*load.Package]*packageTargetBuilder),
		packageExports:         newPackageExportScheduler(),
		environmentBuilders:    make(map[*load.Package]*environmentContractBuilder),
		packageInitializations: newPackageInitializationScheduler(),
		genericOperations:      make(map[genericOperationIdentity]*api.GenericOperationContract),
		genericProfiles:        make(map[genericCallableProfileIdentity]*api.GenericCallableProfile),
		classMembers:           make(map[*types.Func]classMemberContribution),
		goRuntime:              goRuntime,
		compareArtifactOwners:  compareArtifactOwners,
		standardLibrary:        options.StandardLibrary,
		externalFunctions:      make(map[*types.Func]ExternalFunctionObligation),
	}
	for _, sourcePackage := range source.Packages() {
		session.emitters[sourcePackage] = newEmitter(
			sourcePackage,
			session.factory,
			session.registry,
			options.IntegerRepresentation,
			options.EvaluationOrder,
			options.ConcurrencySemantics,
			session.require,
			session,
			session,
			session,
			goRuntime,
		)
	}
	orderedSites := make([]declarationSite, 0, len(sites))
	for _, site := range sites {
		orderedSites = append(orderedSites, site)
	}
	sort.Slice(orderedSites, func(left, right int) bool {
		return declarationindex.CompareSites(
			orderedSites[left],
			orderedSites[right],
		) < 0
	})
	initializers := make(map[*load.Package][]types.Object)
	for _, site := range orderedSites {
		function, ok := site.Declaration.(*ast.FuncDecl)
		if ok && declarationindex.IsPackageInitializer(function) {
			initializers[site.Source] = append(
				initializers[site.Source],
				site.Object,
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
		if err := emitter.names.PreallocatePackageInitializers(objects); err != nil {
			return nil, err
		}
	}
	for _, site := range orderedSites {
		emitter := session.emitters[site.Source]
		if emitter == nil {
			return nil, &ScheduleError{
				Object: site.Object.Name(),
				Reason: "declaration package has no emitter",
			}
		}
		if variable, ok := site.Object.(*types.Var); ok {
			if variable.Name() == "_" {
				continue
			}
			assemblyPath, err := targetoutput.PackageAssemblyPath(site.Source)
			if err != nil {
				return nil, err
			}
			if _, err := emitter.names.ReservePackageVariable(
				variable,
				site.OutputPath,
				assemblyPath,
			); err != nil {
				return nil, err
			}
		} else {
			targetSite, err := session.artifactTargetSite(site)
			if err != nil {
				return nil, err
			}
			if _, err := emitter.names.Reserve(
				site.Object,
				targetSite.SourceFile.Syntax(),
				targetSite.OutputPath,
			); err != nil {
				return nil, err
			}
		}
	}
	return session, nil
}

func sourceArtifactOwnerOrder(
	sites map[types.Object]declarationSite,
) func(api.ArtifactOwner, api.ArtifactOwner) int {
	return func(left api.ArtifactOwner, right api.ArtifactOwner) int {
		leftObject, leftSource := left.Source()
		rightObject, rightSource := right.Source()
		if leftSource && rightSource {
			leftSite, leftIndexed := sites[leftObject]
			rightSite, rightIndexed := sites[rightObject]
			if leftIndexed && rightIndexed {
				if order := declarationindex.CompareSites(
					leftSite,
					rightSite,
				); order != 0 {
					return order
				}
			}
		}
		return emitordering.CompareArtifactOwners(left, right)
	}
}

func (s *programSession) emit(object types.Object) error {
	site, ok := s.sites[object]
	if !ok {
		return s.emitEnvironmentObject(object)
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
	if err := s.recordExternalFunctionObligation(site, object); err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		owner,
		revision.contract,
		revision.dependencies,
		revision.requirements,
	); err != nil {
		return err
	}
	s.commitClassMemberContribution(object, revision.classContribution)
	builder.byOwner[owner] = struct{}{}
	builder.indexByOwner[owner] = len(builder.declarations)
	builder.declarations = append(builder.declarations, targetDeclaration{
		owner:             owner,
		name:              object.Name(),
		position:          object.Pos(),
		statements:        revision.statements,
		placement:         revision.placement,
		eagerDependencies: revision.eagerDependencies,
		temporaryStart:    revision.temporaryStart,
	})
	return s.recordPackageExport(s.packageBuilders[site.Source], object)
}

func (s *programSession) applyRootRequests(
	placement *targetplacement.Owner,
	requests []api.RootRequest,
) error {
	if s.sealed {
		return &ScheduleError{Reason: "root request arrived after target files were sealed"}
	}
	imports := make([]api.RootRequest, 0, len(requests))
	err := api.WalkUniqueRootRequestPayloads(requests, func(request api.RootRequest) error {
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
		return nil
	})
	if err != nil {
		return err
	}
	return placement.Apply(imports)
}

func (s *programSession) builder(site declarationSite) (*targetFileBuilder, error) {
	targetSite, err := s.artifactTargetSite(site)
	if err != nil {
		return nil, err
	}
	return s.builderForFile(
		targetSite.Source,
		targetSite.SourceFile,
		targetSite.OutputPath,
		site.Object.Name(),
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
		placement:     targetplacement.New(),
		byOwner:       make(map[api.ArtifactOwner]struct{}),
		indexByOwner:  make(map[api.ArtifactOwner]int),
	}
	s.builders[outputPath] = builder
	return builder, nil
}
