package emit

import (
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	anonymousstructdeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

func (s *programSession) validateAnonymousStructArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if !artifact.Valid() {
		return &ScheduleError{Reason: "anonymous-struct artifact is invalid"}
	}
	structType, structural := artifact.StructType()
	if !structural {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact is not an anonymous struct",
		}
	}
	binding, ok := s.registry.GeneratedArtifact(
		api.GeneratedArtifactAnonymousStruct,
		artifact.ArtifactKey(),
	)
	if !ok ||
		binding != artifact ||
		!types.Identical(
			binding.SourceType(),
			structType,
		) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "anonymous-struct artifact has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructAnonymousStruct(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "anonymous struct reconstructed after target files were sealed",
		}
	}
	if err := s.validateAnonymousStructArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() !=
		api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "lexical anonymous struct must reconstruct through its source artifact",
		}
	}
	builder, err := s.anonymousStructBuilder()
	if err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	index, exists := builder.indexByOwner[owner]
	var temporaryStart emitnaming.TemporarySnapshot
	if exists {
		temporaryStart = builder.declarations[index].temporaryStart
	}
	revision, err := s.buildAnonymousStructRevision(
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

func (s *programSession) buildAnonymousStructRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "anonymous struct has no concrete name owner",
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

	operations, err := anonymousStructOperations(
		s.requirements.appliedFor(owner),
		artifact,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	structType, ok := artifact.StructType()
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact is not an anonymous struct",
		}
	}
	emission, err := anonymousstructdeclaration.EmitAnonymous(
		builder.context,
		builder.emitter,
		structType,
		artifact.TargetName(),
		operations,
		true,
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
	statements := emission.Declarations()
	contract, err := artifactstate.ProjectContract(
		s.factory,
		statements,
	)
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

func anonymousStructOperations(
	requirements []api.DeclarationRequirement,
	artifact *api.GeneratedArtifact,
) ([]api.NamedStructOperation, error) {
	var operations []api.NamedStructOperation
	for _, requirement := range requirements {
		selected, demand, ok := requirement.AnonymousStruct()
		if !ok || selected != artifact {
			return nil, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "anonymous-struct artifact received a foreign requirement",
			}
		}
		switch demand {
		case api.AnonymousStructDemandDefinition:
		case api.AnonymousStructDemandZero:
			operations = append(
				operations,
				api.NamedStructOperationZero,
			)
		case api.AnonymousStructDemandCopy:
			operations = append(
				operations,
				api.NamedStructOperationCopy,
			)
		case api.AnonymousStructDemandEqual:
			operations = append(
				operations,
				api.NamedStructOperationEqual,
			)
		case api.AnonymousStructDemandHash:
			operations = append(
				operations,
				api.NamedStructOperationHash,
			)
		default:
			return nil, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "anonymous-struct demand is invalid",
			}
		}
	}
	return operations, nil
}

func (s *programSession) anonymousStructBuilder() (
	*targetFileBuilder,
	error,
) {
	if existing := s.builders[output.AnonymousStructSupportPath]; existing != nil {
		return existing, nil
	}
	sourcePackage, ok := deterministicSupportPackage(s.source.Packages())
	if !ok {
		return nil, &ScheduleError{
			Reason: "anonymous-struct support has no deterministic source context",
		}
	}
	emitter := s.emitters[sourcePackage]
	if emitter == nil {
		return nil, &ScheduleError{
			Reason: "anonymous-struct support package has no emitter",
		}
	}
	context, err := emitter.generatedContext(
		output.AnonymousStructSupportPath,
		s.registry,
	)
	if err != nil {
		return nil, err
	}
	builder := &targetFileBuilder{
		sourcePackage: sourcePackage,
		outputPath:    output.AnonymousStructSupportPath,
		emitter:       emitter,
		context:       context,
		placement:     targetplacement.New(),
		byOwner:       make(map[api.ArtifactOwner]struct{}),
		indexByOwner:  make(map[api.ArtifactOwner]int),
	}
	s.builders[output.AnonymousStructSupportPath] = builder
	return builder, nil
}

func deterministicSupportPackage(
	sourcePackages []*load.Package,
) (*load.Package, bool) {
	packages := append([]*load.Package(nil), sourcePackages...)
	sort.Slice(packages, func(left, right int) bool {
		return packages[left].Path() < packages[right].Path()
	})
	for _, sourcePackage := range packages {
		if len(sourcePackage.Files()) != 0 {
			return sourcePackage, true
		}
	}
	return nil, false
}
