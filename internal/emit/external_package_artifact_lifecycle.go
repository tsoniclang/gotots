package emit

import (
	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	declarationindex "github.com/tsoniclang/gotots/internal/emit/declaration/index"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"go/ast"
	"go/types"
	"slices"
)

func sourceImplementationForPackage(
	certificate *sourceimplementation.Certificate,
	sourcePackage *load.Package,
) (sourceimplementation.Implementation, bool) {
	if certificate == nil {
		return sourceimplementation.Implementation{}, false
	}
	return certificate.ForPackage(sourcePackage)
}

type sourceImplementationInputs struct {
	contracts map[api.ArtifactOwner]sourceImplementationContract
	targets   []sourceimplementation.Target
	registry  *emitnaming.Registry
}

type sourceImplementationContract struct {
	contract             artifactstate.Contract
	dependencies         []api.ArtifactDependency
	outboundRequests     []api.RootRequest
	acceptedRequirements []api.DeclarationRequirement
}

func captureSourceImplementationInputs(
	session *programSession,
	roots []Root,
) (sourceImplementationInputs, error) {
	if session == nil || session.sourceImplementations == nil {
		return sourceImplementationInputs{}, &ScheduleError{
			Reason: "source-implementation certification session is absent",
		}
	}
	if err := session.requireProgramRoots(roots); err != nil {
		return sourceImplementationInputs{}, err
	}
	if err := session.settle(); err != nil {
		return sourceImplementationInputs{}, err
	}
	if err := session.verifyTargetFilesSettled(); err != nil {
		return sourceImplementationInputs{}, err
	}
	files, err := session.assembleTargetFiles()
	if err != nil {
		return sourceImplementationInputs{}, err
	}
	targets, err := sourceImplementationTargets(files)
	if err != nil {
		return sourceImplementationInputs{}, err
	}
	contracts, err := session.captureSourceImplementationContracts()
	if err != nil {
		return sourceImplementationInputs{}, err
	}
	registry, err := session.registry.TransferCanonicalIdentity()
	if err != nil {
		return sourceImplementationInputs{}, err
	}
	return sourceImplementationInputs{
		contracts: contracts,
		targets:   targets,
		registry:  registry,
	}, nil
}

func (s *programSession) captureSourceImplementationContracts() (
	map[api.ArtifactOwner]sourceImplementationContract,
	error,
) {
	result := make(map[api.ArtifactOwner]sourceImplementationContract)
	for _, owner := range s.artifacts.Owners() {
		object, sourceOwned := owner.Source()
		if !sourceOwned || object.Pkg() == nil {
			continue
		}
		sourcePackage := s.source.PackageForTypes(object.Pkg())
		if _, selected := sourceImplementationForPackage(
			s.sourceImplementations,
			sourcePackage,
		); !selected {
			continue
		}
		contract, err := s.artifacts.ObservableContract(owner)
		if err != nil {
			return nil, err
		}
		dependencies, exists := s.artifacts.Dependencies(owner)
		if !exists {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "source-implementation artifact dependencies are absent",
			}
		}
		dependencies = s.sourceImplementationDependencies(dependencies)
		result[owner] = sourceImplementationContract{
			contract:             contract,
			dependencies:         dependencies,
			outboundRequests:     s.requirements.ConsumedRequestsBy(owner),
			acceptedRequirements: s.requirements.SelectedFor(owner),
		}
	}
	for _, implementation := range s.sourceImplementations.Implementations() {
		sourcePackage := s.source.PackageByPath(implementation.PackagePath())
		if sourcePackage == nil || s.packageBuilders[sourcePackage] == nil {
			return nil, sourceImplementationError(
				implementation.PackagePath(),
				&ScheduleError{Reason: "selected package was not materialized for certification"},
			)
		}
	}
	return result, nil
}

func (s *programSession) sourceImplementationDependencies(
	dependencies []api.ArtifactDependency,
) []api.ArtifactDependency {
	result := make([]api.ArtifactDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		object, sourceOwned := dependency.Provider().Source()
		if sourceOwned && object.Pkg() != nil {
			sourcePackage := s.source.PackageForTypes(object.Pkg())
			if _, selected := sourceImplementationForPackage(
				s.sourceImplementations,
				sourcePackage,
			); selected {
				continue
			}
		}
		result = append(result, dependency)
	}
	return result
}

func (s *programSession) sourceImplementationContract(
	owner api.ArtifactOwner,
) bool {
	if s.sourceImplementationContracts == nil {
		return false
	}
	_, ok := s.sourceImplementationContracts[owner]
	return ok
}

func (s *programSession) sourceImplementationOwner(
	owner api.ArtifactOwner,
) bool {
	if s.sourceImplementationContracts == nil {
		return false
	}
	object, sourceOwned := owner.Source()
	if !sourceOwned || object.Pkg() == nil {
		return false
	}
	_, selected := sourceImplementationForPackage(
		s.sourceImplementations,
		s.source.PackageForTypes(object.Pkg()),
	)
	return selected
}

func (s *programSession) acceptSourceImplementationRequirements(
	owner api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	if !s.sourceImplementationOwner(owner) {
		return false, nil
	}
	if !s.sourceImplementationContract(owner) {
		return true, &ScheduleError{
			Object: owner.Name(),
			Reason: "source-implementation observable contract is absent",
		}
	}
	contract := s.sourceImplementationContracts[owner]
	for _, requirement := range requirements {
		if !requirement.Valid() || requirement.Owner() != owner {
			return true, &ScheduleError{
				Object: owner.Name(),
				Reason: "source-implementation contract batch has mixed or invalid ownership",
			}
		}
		if !s.requirements.CertifiedContains(requirement) ||
			!slices.Contains(contract.acceptedRequirements, requirement) {
			return true, &ScheduleError{
				Object: owner.Name(),
				Reason: "source-implementation requirement was not certified",
			}
		}
		if !s.requirements.WasApplied(requirement) {
			return true, &ScheduleError{
				Object: owner.Name(),
				Reason: "source-implementation contract requirement was not accepted by its owner",
			}
		}
	}
	s.artifacts.DiscardDirty(owner)
	return true, nil
}

func (s *programSession) acceptSourceImplementationReconstruction(
	owner api.ArtifactOwner,
) (bool, error) {
	if !s.sourceImplementationOwner(owner) {
		return false, nil
	}
	if !s.sourceImplementationContract(owner) {
		return true, &ScheduleError{
			Object: owner.Name(),
			Reason: "source-implementation observable contract is absent",
		}
	}
	s.artifacts.DiscardDirty(owner)
	return true, nil
}

func (s *programSession) emitSourceImplementationContract(
	object types.Object,
	site declarationSite,
) (bool, error) {
	if s.sourceImplementationContracts == nil {
		return false, nil
	}
	owner := api.MustSourceArtifactOwner(object)
	contract, selected := s.sourceImplementationContracts[owner]
	_, packageSelected := sourceImplementationForPackage(
		s.sourceImplementations,
		site.Source,
	)
	if !packageSelected {
		if selected {
			return true, &ScheduleError{
				Object: object.Name(),
				Reason: "source-implementation contract belongs to an unselected package",
			}
		}
		return false, nil
	}
	if !selected {
		return true, &ScheduleError{
			Object: object.Name(),
			Reason: "source-implementation observable contract is absent",
		}
	}
	for _, dependency := range contract.dependencies {
		if err := s.prepareArtifactDependency(dependency); err != nil {
			return true, err
		}
	}
	if err := s.commitArtifactRevision(
		owner,
		contract.contract,
		contract.dependencies,
		contract.outboundRequests,
	); err != nil {
		return true, err
	}
	if _, variable := object.(*types.Var); variable {
		return true, nil
	}
	return true, s.recordPackageExport(s.packageBuilders[site.Source], object)
}
func sourcePackageRequiresInitialization(sourcePackage *load.Package) bool {
	if sourcePackage == nil || sourcePackage.Types() == nil ||
		sourcePackage.TypesInfo() == nil {
		return false
	}
	for _, name := range sourcePackage.Types().Scope().Names() {
		if _, variable := sourcePackage.Types().Scope().Lookup(name).(*types.Var); variable {
			return true
		}
	}
	for _, sourceFile := range sourcePackage.Files() {
		for _, declaration := range sourceFile.Syntax().Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && declarationindex.IsPackageInitializer(function) {
				return true
			}
		}
	}
	return false
}

func (s *programSession) emitPackageInitializer(
	builder *packageTargetBuilder,
	initializer *types.Initializer,
) error {
	anchor, anchored := packageInitializerAnchor(initializer)
	if !anchored {
		return &ScheduleError{
			Object: builder.sourcePackage.Path(),
			Reason: "package initializer has no source declaration anchor",
		}
	}
	site, ok := s.sites[anchor]
	if !ok || site.Source != builder.sourcePackage {
		return &ScheduleError{
			Object: anchor.Name(),
			Reason: "package initializer has no exact source declaration",
		}
	}
	artifactOwner, err := api.PackageInitializerArtifactOwner(
		builder.sourcePackage.Types(),
		initializer,
	)
	if err != nil {
		return err
	}
	if _, duplicate := builder.initializerByOwner[artifactOwner]; duplicate {
		return &ScheduleError{
			Object: artifactOwner.Name(),
			Reason: "package initializer artifact was emitted more than once",
		}
	}
	revision, err := s.buildPackageInitializerRevision(
		builder,
		initializer,
		site,
		artifactOwner,
		emitnaming.TemporarySnapshot{},
		false,
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		artifactOwner,
		revision.contract,
		revision.dependencies,
		revision.requestRoots,
	); err != nil {
		return err
	}
	builder.initializerByOwner[artifactOwner] = len(builder.initialization)
	builder.initialization = append(
		builder.initialization,
		packageInitializationArtifact{
			owner:          artifactOwner,
			initializer:    initializer,
			site:           site,
			statements:     revision.statements,
			placement:      revision.placement,
			temporaryStart: revision.temporaryStart,
		},
	)
	return nil
}

func packageInitializerAnchor(
	initializer *types.Initializer,
) (*types.Var, bool) {
	if initializer == nil {
		return nil, false
	}
	for _, variable := range initializer.Lhs {
		if variable != nil {
			return variable, true
		}
	}
	return nil, false
}

func (s *programSession) buildPackageInitializerRevision(
	builder *packageTargetBuilder,
	initializer *types.Initializer,
	site declarationSite,
	owner api.ArtifactOwner,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.assemblyContext.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: owner.Name(),
			Reason: "package initializer has no concrete name owner",
		}
	}
	replayCommitted := false
	if !reconstruction {
		temporaryStart = names.SnapshotTemporaries()
	} else {
		finishReplay, replayErr := names.BeginTemporaryReplay(
			owner,
			temporaryStart,
		)
		if replayErr != nil {
			return artifactRevision{}, replayErr
		}
		defer func() { finishReplay(replayCommitted) }()
	}
	sourcePath, err := targetoutput.SourcePath(
		site.Source,
		site.SourceFile,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	finish, err := names.BeginArtifact(
		owner,
		site.Declaration,
		site.SourceFile.Syntax(),
		sourcePath,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()
	requirements := s.requirements.SelectedFor(owner)
	context, err := emitnaming.WithLexicalTypeRequirements(
		builder.assemblyContext.WithArtifactOwner(owner),
		site.Declaration,
		owner,
		requirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	context, err = context.WithCallableControls(
		owner,
		initializer.Rhs,
		requirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	emission, err := packagevariable.EmitInitializer(
		context,
		builder.emitter,
		initializer,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, requestRoots, err :=
		s.consumeArtifactRequests(
			owner,
			emission.Requests(),
		)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectCoverageContract(s.factory, nil)
	if err != nil {
		return artifactRevision{}, err
	}
	replayCommitted = true
	return artifactRevision{
		statements:     emission.Statements(),
		placement:      placement,
		dependencies:   dependencies,
		requestRoots:   requestRoots,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func (s *programSession) reconstructPackageInitializer(
	owner api.ArtifactOwner,
) error {
	sourcePackage, initializer, owned := owner.PackageInitializer()
	if !owned {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty package initializer owner is invalid",
		}
	}
	if s.sealed {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package initializer reconstructed after target files were sealed",
		}
	}
	loaded := s.source.PackageForTypes(sourcePackage)
	builder := s.packageBuilders[loaded]
	if builder == nil {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty package initializer lost its package builder",
		}
	}
	index, ok := builder.initializerByOwner[owner]
	if !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty package initializer was not emitted first",
		}
	}
	artifact := &builder.initialization[index]
	if artifact.initializer != initializer {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty package initializer identity changed",
		}
	}
	revision, err := s.buildPackageInitializerRevision(
		builder,
		artifact.initializer,
		artifact.site,
		owner,
		artifact.temporaryStart,
		true,
	)
	if err != nil {
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
	s.artifacts.DiscardDirty(owner)
	artifact.statements = revision.statements
	artifact.placement = revision.placement
	artifact.reconstructions++
	return nil
}

func (s *programSession) retireCompilationGeneratedArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact retired after target files were sealed",
		}
	}
	if err := s.validateGeneratedArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() != api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "non-compilation generated artifact cannot be retired as a target file",
		}
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	builder := s.builders[artifact.OutputPath()]
	if builder == nil {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "retired generated artifact has no materialized target file",
		}
	}
	index, exists := builder.indexByOwner[owner]
	if !exists || index < 0 || index >= len(builder.declarations) ||
		builder.declarations[index].owner != owner {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "retired generated artifact has no exact materialized declaration",
		}
	}
	contract, err := artifactstate.ProjectCoverageContract(s.factory, nil)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(owner, contract, nil, nil); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	removed, err := s.removeTargetDeclaration(owner)
	if err != nil {
		return err
	}
	if !removed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "retired generated artifact declaration survived",
		}
	}
	return nil
}
