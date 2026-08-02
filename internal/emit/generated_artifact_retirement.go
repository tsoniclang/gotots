package emit

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
)

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
	if !exists || index != 0 ||
		len(builder.declarations) != 1 ||
		builder.declarations[0].owner != owner {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "retired generated artifact has no exact materialized target file",
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
	delete(s.builders, artifact.OutputPath())
	return nil
}
