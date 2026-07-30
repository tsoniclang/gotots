package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	emitstorage "github.com/tsoniclang/gotots/internal/emit/storage"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
)

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
		nil,
		false,
	)
	if err != nil {
		return err
	}
	if err := s.artifacts.Commit(
		artifactOwner,
		revision.contract,
		revision.dependencies,
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
	if !reconstruction {
		temporaryStart = names.SnapshotTemporaries()
	} else {
		current := names.SnapshotTemporaries()
		names.RestoreTemporaries(temporaryStart)
		defer names.RestoreTemporaries(current)
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
	requirements := s.requirements.appliedFor(owner)
	context, err := emitnaming.WithLexicalTypeRequirements(
		builder.assemblyContext.WithArtifactOwner(owner),
		site.Declaration,
		owner,
		requirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	context, err = emitstorage.ApplyRequirements(
		context,
		initializer.Rhs,
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
	callableFacet, err := api.NewPackageInitializerCallableFacet(owner)
	if err != nil {
		return artifactRevision{}, err
	}
	observation, err := context.ObserveCooperativeCallable(callableFacet)
	if err != nil {
		return artifactRevision{}, err
	}
	context = context.WithCooperativeCallable(
		callableFacet,
		observation.Cooperative(),
	)
	emission, err := packagevariable.EmitInitializer(
		context,
		builder.emitter,
		initializer,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, err := s.consumeArtifactRequests(
		owner,
		api.CombineRequests(
			emission.Requests(),
			observation.Requests(),
		),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectCoverageContract(s.factory, nil)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     emission.Statements(),
		placement:      placement,
		dependencies:   dependencies,
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
	if err := s.artifacts.Commit(
		owner,
		revision.contract,
		revision.dependencies,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	artifact.statements = revision.statements
	artifact.placement = revision.placement
	artifact.reconstructions++
	return nil
}
