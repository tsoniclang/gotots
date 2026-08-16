package emit

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	genericcapability "github.com/tsoniclang/gotots/internal/emit/generic/capability"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) ensureGenericCapabilityBaseline(
	artifact *api.GeneratedArtifact,
) error {
	owner := api.MustGeneratedArtifactOwner(artifact)
	if s.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetCallableSignature,
	) != 0 {
		return nil
	}
	if err := s.validateGenericCapabilityArtifact(artifact); err != nil {
		return err
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"generic capability baseline",
	)
	if err != nil {
		return err
	}
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic capability baseline has no concrete name owner",
		}
	}
	if names.ArtifactEmissionActive() {
		return nil
	}
	temporaryStart := names.SnapshotTemporaries()
	statement, requests, err := func() (
		tsgo.Statement,
		[]api.RootRequest,
		error,
	) {
		finish, beginErr := names.BeginArtifact(owner, nil, nil, "")
		if beginErr != nil {
			return nil, nil, beginErr
		}
		defer finish()
		context := builder.context.WithArtifactOwner(owner)
		facet, facetErr := api.NewGenericCapabilityCallableFacet(artifact)
		if facetErr != nil {
			return nil, nil, facetErr
		}
		observation, observationErr :=
			context.ObserveCooperativeCallable(facet)
		if observationErr != nil {
			return nil, nil, observationErr
		}
		context = context.WithCooperativeCallable(
			facet,
			observation.Cooperative(),
		)
		result, surfaceRequests, surfaceErr :=
			genericcapability.BuildSurface(
				context,
				builder.emitter,
				artifact,
				[]tsgo.ModifierLike{
					context.Factory().ExportKeyword(),
				},
			)
		return result, api.CombineRequests(
			surfaceRequests,
			observation.Requests(),
		), surfaceErr
	}()
	names.RestoreTemporaries(temporaryStart)
	if err != nil {
		return err
	}
	_, dependencies, requirements, err := s.consumeArtifactRequests(
		owner,
		requests,
	)
	if err != nil {
		return err
	}
	contract, err := artifactstate.ProjectContract(
		s.factory,
		[]tsgo.Statement{statement},
	)
	if err != nil {
		return err
	}
	return s.commitArtifactRevision(
		owner,
		contract,
		dependencies,
		requirements,
	)
}
