package emit

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/externals"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	declarationindex "github.com/tsoniclang/gotots/internal/emit/declaration/index"
	externalfunction "github.com/tsoniclang/gotots/internal/emit/externalfunction"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/emit/runtime/gocontract"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type ProgramEmission struct {
	files                       []TargetFile
	environmentObligations      []EnvironmentObligation
	environmentProfile          EnvironmentProfile
	externalFunctionObligations []ExternalFunctionObligation
	runtimePackage              RuntimePackage
	packageDependencies         []PackageDependency
}

type PackageDependency struct {
	name    string
	version string
}

type RuntimePackage struct {
	assembled runtimeemission.Package
}

type declarationSite = declarationindex.Site

type programSession struct {
	source                        *load.Program
	factory                       tsgo.Factory
	scalar                        api.ScalarABI
	providerScalar                api.ScalarABI
	evaluationOrder               api.EvaluationOrder
	registry                      *emitnaming.Registry
	scheduler                     *scheduler
	requirements                  *declarationRequirementScheduler
	artifacts                     *artifactstate.Graph
	sites                         map[types.Object]declarationSite
	emitters                      map[*load.Package]*emitter
	builders                      map[string]*targetFileBuilder
	packageBuilders               map[*load.Package]*packageTargetBuilder
	packageExports                *packageExportScheduler
	environmentBuilders           map[*load.Package]*environmentContractBuilder
	packageInitializations        *packageInitializationScheduler
	genericOperations             map[genericOperationIdentity]*api.GenericOperationContract
	genericConcretizations        map[genericConcretizationIdentity]*api.GenericConcretization
	classMembers                  map[*types.Func]classMemberContribution
	goRuntime                     *gocontract.Contract
	runtimePackage                RuntimePackage
	compareArtifactOwners         func(api.ArtifactOwner, api.ArtifactOwner) int
	requirementRemovalOwner       api.ArtifactOwner
	standardLibrary               *gostdlibcertify.Certificate
	sourceImplementations         *sourceimplementation.Certificate
	externalFunctions             map[*types.Func]ExternalFunctionObligation
	externalFunctionBindings      map[*types.Func]api.ExternalFunctionTarget
	sourceImplementationContracts map[api.ArtifactOwner]sourceImplementationContract
	sourceImplementationTargets   []sourceimplementation.Target
	preparedDeclarationRequests   map[api.RootRequest]struct{}
	preparedRequirements          map[api.DeclarationRequirement]struct{}
	sealed                        bool
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
	for _, root := range roots {
		if !root.valid() {
			return ProgramEmission{},
				&ScheduleError{Reason: "emission root is invalid"}
		}
	}
	session, err := newProgramSession(source, options)
	if err != nil {
		return ProgramEmission{}, err
	}
	orderedRoots := slices.Clone(roots)
	sort.Slice(orderedRoots, func(left, right int) bool {
		return compareRoots(
			orderedRoots[left],
			orderedRoots[right],
			session.compareArtifactOwners,
		) < 0
	})
	if options.SourceImplementations != nil {
		inputs, captureErr := captureSourceImplementationInputs(
			session,
			orderedRoots,
		)
		if captureErr != nil {
			return ProgramEmission{}, captureErr
		}
		session, err = newProgramSessionWithRegistry(
			source,
			options,
			inputs.registry,
		)
		if err != nil {
			return ProgramEmission{}, err
		}
		session.sourceImplementationContracts = inputs.contracts
		session.sourceImplementationTargets = inputs.targets
	}
	return compileProgramSession(session, orderedRoots, options)
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
	objects, err := declarationindex.FileObjects(sourcePackage, sourceFile)
	if err != nil {
		return nil, err
	}
	roots := make([]Root, 0, len(objects))
	for _, object := range objects {
		root, err := newRoot(object, RootFileCoverage)
		if err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}
	return roots, nil
}

func selectedPackageDependencies(
	options Options,
	runtimePackage RuntimePackage,
) ([]PackageDependency, error) {
	var dependencies []PackageDependency
	if runtimePackage.Name() != "" {
		dependencies = append(dependencies, PackageDependency{
			name: runtimePackage.Name(), version: runtimePackage.Version(),
		})
	}
	if options.StandardLibrary != nil {
		dependencies = append(dependencies, PackageDependency{
			name: gostdlib.PackageName, version: gostdlib.PackageVersion,
		})
	}
	if options.ExternalProvider != nil {
		dependencies = append(dependencies, PackageDependency{
			name: externals.PackageName, version: externals.PackageVersion,
		})
	}
	for _, dependency := range dependencies {
		if dependency.name == "" || dependency.version == "" {
			return nil, &ScheduleError{Reason: "package dependency identity is incomplete"}
		}
	}
	sort.Slice(dependencies, func(left, right int) bool {
		return dependencies[left].name < dependencies[right].name
	})
	for index := 1; index < len(dependencies); index++ {
		if dependencies[index-1].name == dependencies[index].name {
			return nil, &ScheduleError{Reason: "package dependency has multiple owners"}
		}
	}
	return dependencies, nil
}

func newProgramSession(
	source *load.Program,
	options Options,
) (*programSession, error) {
	return newProgramSessionWithRegistry(source, options, nil)
}

func newProgramSessionWithRegistry(
	source *load.Program,
	options Options,
	registry *emitnaming.Registry,
) (*programSession, error) {
	if registry != nil {
		if err := registry.ClaimFinalSession(); err != nil {
			return nil, err
		}
	}
	scalar, err := programScalarABI(source, options.IntegerRepresentation)
	if err != nil {
		return nil, err
	}
	providerScalar, err := certifiedProviderScalarABI(
		options.StandardLibrary,
		scalar.NativeIntegerWidth(),
	)
	if err != nil {
		return nil, err
	}
	sites, err := declarationindex.Program(source)
	if err != nil {
		return nil, err
	}
	externalBindings, externalModules, err := externalfunction.Resolve(
		source,
		sites,
		options.ExternalProvider,
		options.StandardLibrary,
		providerScalar,
	)
	if err != nil {
		return nil, err
	}
	if registry == nil {
		registry = emitnaming.NewRegistry()
		if err := registry.IndexCompilationTargets(
			source.Packages(),
			source.EnvironmentPackages(),
			options.StandardLibrary,
			externalModules,
		); err != nil {
			return nil, err
		}
	}
	goRuntime, err := gocontract.Resolve(source)
	if err != nil {
		return nil, err
	}
	compareArtifactOwners := sourceArtifactOwnerOrder(sites)
	session := &programSession{
		source:          source,
		factory:         tsgo.NewFactory(),
		scalar:          scalar,
		providerScalar:  providerScalar,
		evaluationOrder: options.EvaluationOrder,
		registry:        registry,
		scheduler:       newScheduler(),
		requirements: newDeclarationRequirementScheduler(
			compareArtifactOwners,
		),
		artifacts: artifactstate.NewGraph(
			compareArtifactOwners,
		),
		sites:                       sites,
		emitters:                    make(map[*load.Package]*emitter),
		builders:                    make(map[string]*targetFileBuilder),
		packageBuilders:             make(map[*load.Package]*packageTargetBuilder),
		packageExports:              newPackageExportScheduler(),
		environmentBuilders:         make(map[*load.Package]*environmentContractBuilder),
		packageInitializations:      newPackageInitializationScheduler(),
		genericOperations:           make(map[genericOperationIdentity]*api.GenericOperationContract),
		genericConcretizations:      make(map[genericConcretizationIdentity]*api.GenericConcretization),
		classMembers:                make(map[*types.Func]classMemberContribution),
		goRuntime:                   goRuntime,
		compareArtifactOwners:       compareArtifactOwners,
		standardLibrary:             options.StandardLibrary,
		sourceImplementations:       options.SourceImplementations,
		externalFunctions:           make(map[*types.Func]ExternalFunctionObligation),
		externalFunctionBindings:    externalBindings,
		preparedDeclarationRequests: make(map[api.RootRequest]struct{}),
		preparedRequirements:        make(map[api.DeclarationRequirement]struct{}),
	}
	for _, sourcePackage := range source.Packages() {
		implementationContract := false
		if options.SourceImplementations != nil {
			_, implementationContract = options.SourceImplementations.ForPackage(
				sourcePackage,
			)
		}
		session.emitters[sourcePackage] = newEmitter(
			sourcePackage,
			session.factory,
			session.registry,
			scalar,
			providerScalar,
			options.EvaluationOrder,
			session,
			session,
			session,
			session,
			session,
			goRuntime,
			implementationContract,
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

func (s *programSession) emit(object types.Object) error {
	site, ok := s.sites[object]
	if !ok {
		return s.emitEnvironmentObject(object)
	}
	if emitted, err := s.emitSourceImplementationContract(object, site); emitted || err != nil {
		return err
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
		emitnaming.TemporarySnapshot{},
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
		revision.requestRoots,
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
	placementRequests := make([]api.RootRequest, 0, len(requests))
	err := api.WalkUniqueRootRequestPayloads(requests, func(request api.RootRequest) error {
		switch request.Kind() {
		case api.RootRequestImport:
			placementRequests = append(placementRequests, request)
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
	return placement.Apply(placementRequests)
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
