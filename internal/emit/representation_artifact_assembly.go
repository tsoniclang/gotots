package emit

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	interfaceadapterdeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/interfaceadapter"
	interfacedynamictype "github.com/tsoniclang/gotots/internal/emit/declaration/interfacedynamictype"
	interfacemethodtoken "github.com/tsoniclang/gotots/internal/emit/declaration/interfacemethodtoken"
	interfacetypedeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/interfacetype"
	providerinterfacebridge "github.com/tsoniclang/gotots/internal/emit/declaration/providerinterfacebridge"
	reflectiontypedeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/reflectiontype"
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
	case api.GeneratedArtifactProviderInterfaceBridge:
		source, sourceOK := artifact.ProviderInterfaceBridgeType()
		bound, boundOK := binding.ProviderInterfaceBridgeType()
		if !sourceOK || !boundOK || !types.Identical(source, bound) {
			return &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "provider-interface bridge has inconsistent source type",
			}
		}
	case api.GeneratedArtifactReflectionType:
		source, reflectionType, sourceOK := artifact.ReflectionType()
		bound, boundReflectionType, boundOK := binding.ReflectionType()
		if !sourceOK || !boundOK || !types.Identical(source, bound) ||
			reflectionType != boundReflectionType {
			return &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "reflection type has inconsistent source contract",
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
	requirements := s.requirements.appliedFor(owner)
	var adapterContracts []*types.Interface
	if artifact.Kind() == api.GeneratedArtifactInterfaceAdapter {
		adapterContracts, err = interfaceadapterdeclaration.Contracts(
			artifact,
			requirements,
		)
	} else {
		err = exactInterfaceRequirement(requirements, artifact)
	}
	if err != nil {
		return artifactRevision{}, err
	}
	statements, requests, err := buildInterfaceArtifact(
		builder,
		context,
		artifact,
		adapterContracts,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, requirements, err :=
		s.consumeArtifactRequests(
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
		requirements:   requirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func buildInterfaceArtifact(
	builder *targetFileBuilder,
	context api.Context,
	artifact *api.GeneratedArtifact,
	adapterContracts []*types.Interface,
) ([]tsgo.Statement, []api.RootRequest, error) {
	switch artifact.Kind() {
	case api.GeneratedArtifactInterfaceMethodToken:
		runtime, ok := artifact.InterfaceMethodRuntime()
		if !ok {
			return nil, nil, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "interface-method token runtime identity is invalid",
			}
		}
		var initializer tsgo.Expression
		var requests []api.RootRequest
		if runtime != api.RuntimeInvalid {
			reference, err := context.Names().Runtime(
				runtime,
				api.ImportPhaseValue,
			)
			if err != nil {
				return nil, nil, err
			}
			initializer = reference.Expression(context.Factory())
			requests = reference.Requests()
		}
		return []tsgo.Statement{
			interfacemethodtoken.BuildIsolated(
				builder.context.Factory(),
				artifact.TargetName(),
				[]tsgo.ModifierLike{
					builder.context.Factory().ExportKeyword(),
				},
				initializer,
			),
		}, requests, nil
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		source, ok := artifact.InterfaceDynamicType()
		if !ok {
			return nil, nil, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "interface dynamic-type source is invalid",
			}
		}
		return []tsgo.Statement{
			interfacedynamictype.BuildIsolated(
				builder.context.Factory(),
				artifact.TargetName(),
				[]tsgo.ModifierLike{
					builder.context.Factory().ExportKeyword(),
				},
				types.Comparable(source),
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
		return interfacetypedeclaration.BuildIsolated(
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
			adapterContracts,
			[]tsgo.ModifierLike{
				builder.context.Factory().ExportKeyword(),
			},
		)
	case api.GeneratedArtifactProviderInterfaceBridge:
		source, ok := artifact.ProviderInterfaceBridgeType()
		if !ok {
			return nil, nil, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "provider-interface bridge source is invalid",
			}
		}
		return providerinterfacebridge.Build(
			context,
			builder.emitter,
			artifact.TargetName(),
			source,
			[]tsgo.ModifierLike{
				builder.context.Factory().ExportKeyword(),
			},
		)
	case api.GeneratedArtifactReflectionType:
		return reflectiontypedeclaration.Build(context, artifact)
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
			Reason: fmt.Sprintf(
				"interface artifact requires exactly one definition request, received %d",
				len(requirements),
			),
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
