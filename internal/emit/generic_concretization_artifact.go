package emit

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	genericconcretization "github.com/tsoniclang/gotots/internal/emit/generic/concretization"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) validateGenericConcretizationArtifact(
	artifact *api.GeneratedArtifact,
) error {
	concretization, ok := artifact.GenericConcretization()
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactGenericConcretization,
		artifact.ArtifactKey(),
	)
	bound, boundOK := binding.GenericConcretization()
	if !ok || !found || binding != artifact || !boundOK ||
		bound != concretization ||
		!types.Identical(bound.Signature(), concretization.Signature()) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic concretization has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructGenericConcretizationArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic concretization reconstructed after target files were sealed",
		}
	}
	if err := s.validateGenericConcretizationArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() != api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "lexical generic concretization must reconstruct through its source artifact",
		}
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"generic concretization",
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
	revision, err := s.buildGenericConcretizationRevision(
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

func (s *programSession) buildGenericConcretizationRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic concretization has no concrete name owner",
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
	context := builder.context.WithArtifactOwner(owner)
	deferred, err := exactGenericConcretizationRequirement(
		artifact,
		s.requirements.appliedFor(owner),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements, requests, err := genericconcretization.Build(
		context,
		builder.emitter,
		artifact,
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
		deferred,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, requirements, err :=
		s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectContract(s.factory, statements)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     statements,
		placement:      placement,
		dependencies:   dependencies,
		requirements:   requirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func exactGenericConcretizationRequirement(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	if len(requirements) < 1 || len(requirements) > 2 {
		return false, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic concretization requirements are not exact",
		}
	}
	bound, boundOK := artifact.GenericConcretization()
	base := false
	deferred := false
	for _, requirement := range requirements {
		selected, ok := requirement.GenericConcretization()
		generated, generatedOK := requirement.GeneratedArtifact()
		if !ok || !generatedOK || generated != artifact || !boundOK ||
			selected != bound {
			return false, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "generic concretization received a foreign requirement",
			}
		}
		if requirement.DeferredGenericConcretization() {
			if deferred {
				return false, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "generic concretization has duplicate deferred demand",
				}
			}
			deferred = true
		} else {
			if base {
				return false, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "generic concretization has duplicate definition demand",
				}
			}
			base = true
		}
	}
	if !base {
		return false, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic concretization lacks its definition demand",
		}
	}
	return deferred, nil
}
