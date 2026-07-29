package emit

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	interfaceadapterdeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/interfaceadapter"
	interfacedynamictype "github.com/tsoniclang/gotots/internal/emit/declaration/interfacedynamictype"
	interfacemethodtoken "github.com/tsoniclang/gotots/internal/emit/declaration/interfacemethodtoken"
	interfacetypedeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/interfacetype"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) validateInterfaceArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if !artifact.Valid() {
		return &ScheduleError{Reason: "interface artifact is invalid"}
	}
	binding, ok := s.registry.GeneratedArtifact(
		artifact.Kind(),
		artifact.ArtifactKey(),
	)
	if !ok || binding != artifact {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "interface artifact has no exact canonical binding",
		}
	}
	switch artifact.Kind() {
	case api.GeneratedArtifactInterfaceAdapter:
		source, sourceOK := artifact.InterfaceAdapterType()
		bound, boundOK := binding.InterfaceAdapterType()
		if !sourceOK || !boundOK || !types.Identical(source, bound) {
			return &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "interface adapter has inconsistent source type",
			}
		}
	case api.GeneratedArtifactAnonymousInterface:
		source, sourceOK := artifact.InterfaceType()
		bound, boundOK := binding.InterfaceType()
		if !sourceOK || !boundOK || !types.Identical(source, bound) {
			return &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "anonymous interface has inconsistent source type",
			}
		}
	case api.GeneratedArtifactInterfaceMethodToken:
		source, sourceOK := artifact.InterfaceMethodSignature()
		bound, boundOK := binding.InterfaceMethodSignature()
		if !sourceOK || !boundOK || !types.Identical(source, bound) {
			return &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "interface method token has inconsistent signature",
			}
		}
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		source, sourceOK := artifact.InterfaceDynamicType()
		bound, boundOK := binding.InterfaceDynamicType()
		if !sourceOK || !boundOK || !types.Identical(source, bound) {
			return &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "interface dynamic-type token has inconsistent source type",
			}
		}
	default:
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "interface artifact kind is invalid",
		}
	}
	return nil
}

func (s *programSession) reconstructInterfaceArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "interface artifact reconstructed after target files were sealed",
		}
	}
	if err := s.validateInterfaceArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() != api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "lexical interface artifact must reconstruct through its source artifact",
		}
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"interface artifact",
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
	revision, err := s.buildInterfaceArtifactRevision(
		builder,
		artifact,
		temporaryStart,
		exists,
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

func (s *programSession) buildInterfaceArtifactRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "interface artifact has no concrete name owner",
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
	if err := exactInterfaceRequirement(
		s.requirements.appliedFor(owner),
		artifact,
	); err != nil {
		return artifactRevision{}, err
	}
	statements, requests, err := buildInterfaceArtifact(
		builder,
		context,
		artifact,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, err := s.consumeArtifactRequests(
		owner,
		requests,
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
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func buildInterfaceArtifact(
	builder *targetFileBuilder,
	context api.Context,
	artifact *api.GeneratedArtifact,
) ([]tsgo.Statement, []api.RootRequest, error) {
	switch artifact.Kind() {
	case api.GeneratedArtifactInterfaceMethodToken:
		return []tsgo.Statement{
			interfacemethodtoken.Build(
				builder.context.Factory(),
				artifact.TargetName(),
			),
		}, nil, nil
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		return []tsgo.Statement{
			interfacedynamictype.Build(
				builder.context.Factory(),
				artifact.TargetName(),
			),
		}, nil, nil
	case api.GeneratedArtifactAnonymousInterface:
		source, ok := artifact.InterfaceType()
		if !ok {
			return nil, nil, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "anonymous-interface source is invalid",
			}
		}
		return interfacetypedeclaration.Build(
			context,
			builder.emitter,
			nil,
			artifact.TargetName(),
			source,
			[]tsgo.ModifierLike{
				builder.context.Factory().ExportKeyword(),
			},
		)
	case api.GeneratedArtifactInterfaceAdapter:
		source, ok := artifact.InterfaceAdapterType()
		if !ok {
			return nil, nil, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "interface-adapter source is invalid",
			}
		}
		return interfaceadapterdeclaration.Build(
			context,
			builder.emitter,
			artifact.TargetName(),
			source,
			[]tsgo.ModifierLike{
				builder.context.Factory().ExportKeyword(),
			},
		)
	default:
		return nil, nil, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "interface artifact kind is invalid",
		}
	}
}

func exactInterfaceRequirement(
	requirements []api.DeclarationRequirement,
	artifact *api.GeneratedArtifact,
) error {
	if len(requirements) != 1 {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "interface artifact requires exactly one definition request",
		}
	}
	selected, ok := requirements[0].GeneratedArtifact()
	if !ok || selected != artifact {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "interface artifact received a foreign requirement",
		}
	}
	return nil
}
