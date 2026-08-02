package emit

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) validateDeferredCallableRegistry(
	artifact *api.GeneratedArtifact,
) error {
	signature, sourceOK := artifact.DeferredCallableRegistry()
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactDeferredCallableRegistry,
		artifact.ArtifactKey(),
	)
	bound, boundOK := binding.DeferredCallableRegistry()
	if !sourceOK || !found || binding != artifact || !boundOK ||
		!types.Identical(signature, bound) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructDeferredCallableRegistry(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry reconstructed after sealing",
		}
	}
	if err := s.validateDeferredCallableRegistry(artifact); err != nil {
		return err
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"deferred-callable registry",
	)
	if err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	index, exists := builder.indexByOwner[owner]
	var temporaryStart emitnaming.TemporarySnapshot
	if exists {
		temporaryStart = builder.declarations[index].temporaryStart
	}
	revision, err := s.buildDeferredCallableRegistryRevision(
		builder,
		artifact,
		temporaryStart,
		exists,
	)
	if err != nil {
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
	s.artifacts.DiscardDirty(owner)
	declaration := targetDeclaration{
		owner:          owner,
		name:           artifact.TargetName(),
		position:       token.NoPos,
		statements:     revision.statements,
		placement:      revision.placement,
		temporaryStart: revision.temporaryStart,
	}
	if exists {
		declaration.reconstructions =
			builder.declarations[index].reconstructions + 1
		builder.declarations[index] = declaration
		return nil
	}
	builder.byOwner[owner] = struct{}{}
	builder.indexByOwner[owner] = len(builder.declarations)
	builder.declarations = append(builder.declarations, declaration)
	return nil
}

func (s *programSession) buildDeferredCallableRegistryRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry has no concrete name owner",
		}
	}
	if !reconstruction {
		temporaryStart = names.SnapshotTemporaries()
	} else {
		current := names.SnapshotTemporaries()
		names.RestoreTemporaries(temporaryStart)
		defer names.RestoreTemporaries(current)
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	finish, err := names.BeginArtifact(owner, nil, nil, "")
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()
	requirements := s.requirements.appliedFor(owner)
	definitions := 0
	for _, requirement := range requirements {
		if selected, ok := requirement.DeferredCallableRegistry(); ok {
			if selected != artifact {
				return artifactRevision{}, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "deferred-callable registry received a foreign definition",
				}
			}
			definitions++
			continue
		}
		if _, cooperative := requirement.CooperativeCallable(); !cooperative {
			return artifactRevision{}, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "deferred-callable registry received a foreign requirement",
			}
		}
	}
	if definitions != 1 {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry requires one definition",
		}
	}
	context := builder.context.WithArtifactOwner(owner)
	statement, valueType, requests, err := deferredregistry.Build(
		context,
		builder.emitter,
		artifact,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements := []tsgo.Statement{statement}
	placement, dependencies, nextRequirements, err :=
		s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectExplicitValueContract(
		s.factory,
		statement,
		valueType,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     statements,
		placement:      placement,
		dependencies:   dependencies,
		requirements:   nextRequirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}
