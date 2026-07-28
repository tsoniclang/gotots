package emit

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	maprepresentation "github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) validateGeneratedArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if !artifact.Valid() {
		return &ScheduleError{Reason: "generated artifact is invalid"}
	}
	switch artifact.Kind() {
	case api.GeneratedArtifactAnonymousStruct:
		return s.validateAnonymousStructArtifact(artifact)
	case api.GeneratedArtifactMapSpecialization:
		return s.validateMapSpecializationArtifact(artifact)
	default:
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact kind is invalid",
		}
	}
}

func (s *programSession) reconstructGeneratedArtifact(
	artifact *api.GeneratedArtifact,
) error {
	switch artifact.Kind() {
	case api.GeneratedArtifactAnonymousStruct:
		return s.reconstructAnonymousStruct(artifact)
	case api.GeneratedArtifactMapSpecialization:
		return s.reconstructMapSpecialization(artifact)
	default:
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact kind is invalid",
		}
	}
}

func (s *programSession) validateMapSpecializationArtifact(
	artifact *api.GeneratedArtifact,
) error {
	mapType, mapArtifact := artifact.MapType()
	if !mapArtifact {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact is not a map specialization",
		}
	}
	binding, ok := s.registry.GeneratedArtifact(
		api.GeneratedArtifactMapSpecialization,
		artifact.ArtifactKey(),
	)
	bindingType, bindingOK := binding.MapType()
	if !ok ||
		binding != artifact ||
		!bindingOK ||
		!types.Identical(bindingType, mapType) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "map-specialization artifact has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructMapSpecialization(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "map specialization reconstructed after target files were sealed",
		}
	}
	if err := s.validateMapSpecializationArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() != api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "lexical map specialization must reconstruct through its source artifact",
		}
	}
	builder, err := s.mapSpecializationBuilder(artifact)
	if err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	index, exists := builder.indexByOwner[owner]
	var temporaryStart emitnaming.TemporarySnapshot
	if exists {
		temporaryStart = builder.declarations[index].temporaryStart
	}
	revision, err := s.buildMapSpecializationRevision(
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

func (s *programSession) buildMapSpecializationRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "map specialization has no concrete name owner",
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
	if err := validateMapSpecializationRequirements(
		artifact,
		s.requirements.appliedFor(owner),
	); err != nil {
		return artifactRevision{}, err
	}
	mapType, ok := artifact.MapType()
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact is not a map specialization",
		}
	}
	keyType, err := builder.emitter.RepresentedType(
		builder.context.WithRole(api.RoleMapKey),
		nil,
		mapType.Key(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	valueType, err := builder.emitter.RepresentedType(
		builder.context.WithRole(api.RoleMapValue),
		nil,
		mapType.Elem(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	specialization, err := maprepresentation.BuildSpecialization(
		builder.context,
		nil,
		artifact.TargetName(),
		mapType,
		keyType.Value(),
		valueType.Value(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statement := tsgo.Statement(builder.context.Factory().ClassDeclaration(
		[]tsgo.ModifierLike{builder.context.Factory().ExportKeyword()},
		builder.context.Factory().Identifier(artifact.TargetName()),
		nil,
		nil,
		specialization.Members(),
	))
	requests := api.CombineRequests(
		keyType.Requests(),
		valueType.Requests(),
		specialization.Requests(),
	)
	placement, dependencies, err := s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectContract(
		s.factory,
		[]tsgo.Statement{statement},
	)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     []tsgo.Statement{statement},
		placement:      placement,
		dependencies:   dependencies,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func validateMapSpecializationRequirements(
	artifact *api.GeneratedArtifact,
	requirements []api.DeclarationRequirement,
) error {
	for _, requirement := range requirements {
		selected, _, ok := requirement.MapSpecialization()
		if !ok || selected != artifact {
			return &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "map-specialization artifact received a foreign requirement",
			}
		}
	}
	return nil
}

func (s *programSession) mapSpecializationBuilder(
	artifact *api.GeneratedArtifact,
) (*targetFileBuilder, error) {
	if existing := s.builders[artifact.OutputPath()]; existing != nil {
		return existing, nil
	}
	sourcePackage, ok := deterministicSupportPackage(s.source.Packages())
	if !ok {
		return nil, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "map-specialization support has no deterministic source context",
		}
	}
	emitter := s.emitters[sourcePackage]
	if emitter == nil {
		return nil, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "map-specialization support package has no emitter",
		}
	}
	context, err := emitter.generatedContext(
		artifact.OutputPath(),
		s.registry,
	)
	if err != nil {
		return nil, err
	}
	builder := &targetFileBuilder{
		sourcePackage: sourcePackage,
		outputPath:    artifact.OutputPath(),
		emitter:       emitter,
		context:       context,
		placement:     targetplacement.New(),
		byOwner:       make(map[api.ArtifactOwner]struct{}),
		indexByOwner:  make(map[api.ArtifactOwner]int),
	}
	s.builders[artifact.OutputPath()] = builder
	return builder, nil
}
