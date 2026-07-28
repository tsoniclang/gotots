package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
)

func (s *programSession) emitPackageInitializer(
	builder *packageTargetBuilder,
	initializer *types.Initializer,
) error {
	owner, owned := packageInitializerOwner(initializer)
	if !owned {
		return &ScheduleError{
			Object: builder.sourcePackage.Path(),
			Reason: "package initializer has no exact source owner",
		}
	}
	site, ok := s.sites[owner]
	if !ok || site.source != builder.sourcePackage {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package initializer has no exact source declaration",
		}
	}
	artifactOwner := api.MustSourceArtifactOwner(owner)
	if _, duplicate := builder.initializerByOwner[artifactOwner]; duplicate {
		return &ScheduleError{
			Object: owner.Name(),
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

func packageInitializerOwner(
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
	temporaryStart temporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.assemblyContext.Names().(*fileNames)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: owner.Name(),
			Reason: "package initializer has no concrete name owner",
		}
	}
	if !reconstruction {
		temporaryStart = names.snapshotTemporaries()
	} else {
		current := names.snapshotTemporaries()
		names.restoreTemporaries(temporaryStart)
		defer names.restoreTemporaries(current)
	}
	sourcePath, err := targetoutput.SourcePath(
		site.source,
		site.sourceFile,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	finish, err := names.beginArtifact(
		owner,
		site.declaration,
		site.sourceFile.Syntax(),
		sourcePath,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()
	requirements := s.requirements.appliedFor(owner)
	context, err := withLexicalAnonymousStructs(
		builder.assemblyContext,
		site.declaration,
		owner,
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
	placement, dependencies, err := s.consumeArtifactRequests(
		owner,
		emission.Requests(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectCoverageContract(nil)
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
	variable *types.Var,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "package initializer reconstructed after target files were sealed",
		}
	}
	site, ok := s.sites[variable]
	if !ok {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "dirty package initializer lost its source declaration",
		}
	}
	builder := s.packageBuilders[site.source]
	if builder == nil {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "dirty package initializer lost its package builder",
		}
	}
	owner := api.MustSourceArtifactOwner(variable)
	index, ok := builder.initializerByOwner[owner]
	if !ok {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "dirty package initializer was not emitted first",
		}
	}
	artifact := &builder.initialization[index]
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
