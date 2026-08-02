package emit

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	providerstatefulrepresentation "github.com/tsoniclang/gotots/internal/emit/declaration/providerstatefulrepresentation"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) validateProviderStatefulArtifact(
	artifact *api.GeneratedArtifact,
) error {
	source, sourceOK := artifact.ProviderStatefulRepresentationType()
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactProviderStatefulRepresentation,
		artifact.ArtifactKey(),
	)
	bound, boundOK := binding.ProviderStatefulRepresentationType()
	if !sourceOK || !found || binding != artifact || !boundOK ||
		!types.Identical(source, bound) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful-representation artifact has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructProviderStatefulArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation reconstructed after target files were sealed",
		}
	}
	if err := s.validateProviderStatefulArtifact(artifact); err != nil {
		return err
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"provider stateful representation",
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
	revision, err := s.buildProviderStatefulRevision(
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

func (s *programSession) buildProviderStatefulRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation has no concrete name owner",
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
	if len(requirements) != 1 {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation requires exactly one definition request",
		}
	}
	selectedRequirement, ok :=
		requirements[0].ProviderStatefulRepresentation()
	if !ok || selectedRequirement != artifact {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation received a foreign requirement",
		}
	}
	source, ok := artifact.ProviderStatefulRepresentationType()
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation source is invalid",
		}
	}
	context := builder.context.WithArtifactOwner(owner)
	selection, profiled, err := providerboundary.ResolveStatefulProfile(
		context,
		source,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	profileKey := ""
	var typeArguments []tsgo.TypeNode
	var requests []api.RootRequest
	if profiled {
		profileKey = selection.Profile().ProfileKey()
		for _, selected := range selection.TypeArguments() {
			represented, representErr := interfacetype.EmitNonNil(
				context.WithRole(api.RoleDefinedTypeArgument),
				builder.emitter,
				nil,
				selected,
			)
			if representErr != nil {
				return artifactRevision{}, representErr
			}
			typeArguments = append(typeArguments, represented.Value())
			requests = append(requests, represented.Requests()...)
		}
	}
	typeTarget, err := names.ProviderStatefulProfileTarget(
		source.Obj(),
		profileKey,
		api.ImportPhaseType,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	valueTarget, err := names.ProviderStatefulProfileTarget(
		source.Obj(),
		profileKey,
		api.ImportPhaseValue,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements, buildRequests, err := providerstatefulrepresentation.Build(
		context.Factory(),
		artifact.TargetName(),
		typeTarget,
		valueTarget,
		typeArguments,
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, nextRequirements, err :=
		s.consumeArtifactRequests(
			owner,
			api.CombineRequests(
				requests,
				selection.Requests(),
				buildRequests,
			),
		)
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
		requirements:   nextRequirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}
