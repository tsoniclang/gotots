package emit

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	genericcapability "github.com/tsoniclang/gotots/internal/emit/generic/capability"
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
	case api.GeneratedArtifactInterfaceAdapter,
		api.GeneratedArtifactAnonymousInterface,
		api.GeneratedArtifactInterfaceMethodToken,
		api.GeneratedArtifactInterfaceDynamicTypeToken,
		api.GeneratedArtifactProviderInterfaceBridge,
		api.GeneratedArtifactReflectionType,
		api.GeneratedArtifactUnsafeCodec:
		return s.validateRepresentationArtifact(artifact)
	case api.GeneratedArtifactGenericCapability:
		return s.validateGenericCapabilityArtifact(artifact)
	case api.GeneratedArtifactGenericConcretization:
		return s.validateGenericConcretizationArtifact(artifact)
	case api.GeneratedArtifactCallableABI,
		api.GeneratedArtifactInterfaceMethodCallable:
		return s.validateCallableContractArtifact(artifact)
	case api.GeneratedArtifactPointerRepresentation:
		return s.validatePointerRepresentationArtifact(artifact)
	case api.GeneratedArtifactProviderStatefulRepresentation:
		return s.validateProviderStatefulArtifact(artifact)
	case api.GeneratedArtifactDeferredCallableRegistry:
		return s.validateDeferredCallableRegistry(artifact)
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
	var err error
	switch artifact.Kind() {
	case api.GeneratedArtifactAnonymousStruct:
		err = s.reconstructAnonymousStruct(artifact)
	case api.GeneratedArtifactMapSpecialization:
		err = s.reconstructMapSpecialization(artifact)
	case api.GeneratedArtifactInterfaceAdapter,
		api.GeneratedArtifactAnonymousInterface,
		api.GeneratedArtifactInterfaceMethodToken,
		api.GeneratedArtifactInterfaceDynamicTypeToken,
		api.GeneratedArtifactProviderInterfaceBridge,
		api.GeneratedArtifactReflectionType,
		api.GeneratedArtifactUnsafeCodec:
		err = s.reconstructRepresentationArtifact(artifact)
	case api.GeneratedArtifactGenericCapability:
		err = s.reconstructGenericCapabilityArtifact(artifact)
	case api.GeneratedArtifactGenericConcretization:
		err = s.reconstructGenericConcretizationArtifact(artifact)
	case api.GeneratedArtifactCallableABI,
		api.GeneratedArtifactInterfaceMethodCallable:
		err = s.reconstructCallableContractArtifact(artifact)
	case api.GeneratedArtifactPointerRepresentation:
		err = s.reconstructPointerRepresentationArtifact(artifact)
	case api.GeneratedArtifactProviderStatefulRepresentation:
		err = s.reconstructProviderStatefulArtifact(artifact)
	case api.GeneratedArtifactDeferredCallableRegistry:
		err = s.reconstructDeferredCallableRegistry(artifact)
	default:
		err = &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact kind is invalid",
		}
	}
	return api.WrapGeneratedArtifactError(artifact, err)
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
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"map specialization",
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
	revision, err := s.buildMapSpecializationRevision(
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
	context := builder.context.WithArtifactOwner(owner)
	err = maprepresentation.ValidateRequirements(
		api.RoleFileDeclaration,
		artifact,
		s.requirements.appliedFor(owner),
	)
	if err != nil {
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
		context.WithRole(api.RoleMapKey),
		nil,
		mapType.Key(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	mapModel, ok := maprepresentation.Source(context, mapType)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated map specialization has no representation model",
		}
	}
	storageKeyType, err := builder.emitter.RepresentedType(
		context.WithRole(api.RoleStorageType),
		nil,
		mapModel.StorageKey(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	valueType, err := builder.emitter.RepresentedType(
		context.WithRole(api.RoleMapValue),
		nil,
		mapType.Elem(),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	specialization, err := maprepresentation.BuildSpecialization(
		context,
		nil,
		artifact.TargetName(),
		mapType,
		keyType.Value(),
		storageKeyType.Value(),
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
		storageKeyType.Requests(),
		valueType.Requests(),
		specialization.Requests(),
	)
	placement, dependencies, requirements, err :=
		s.consumeArtifactRequests(owner, requests)
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
		requirements:   requirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func (s *programSession) validateGenericCapabilityArtifact(
	artifact *api.GeneratedArtifact,
) error {
	signature, selection, ok := artifact.GenericCapability()
	if !ok {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact is not a generic capability",
		}
	}
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactGenericCapability,
		artifact.ArtifactKey(),
	)
	boundSignature, boundSelection, bound := binding.GenericCapability()
	if !found ||
		binding != artifact ||
		!bound ||
		boundSelection != selection ||
		!types.Identical(boundSignature, signature) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic-capability artifact has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructGenericCapabilityArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic capability reconstructed after target files were sealed",
		}
	}
	if err := s.validateGenericCapabilityArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() != api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "lexical generic capability must reconstruct through its source artifact",
		}
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"generic capability",
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
	revision, err := s.buildGenericCapabilityRevision(
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

func (s *programSession) buildGenericCapabilityRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic capability has no concrete name owner",
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
	if err := genericcapability.ValidateRequirements(
		builder.context.Role(),
		artifact,
		s.requirements.appliedFor(owner),
	); err != nil {
		return artifactRevision{}, err
	}
	facet, err := api.NewGenericCapabilityCallableFacet(artifact)
	if err != nil {
		return artifactRevision{}, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return artifactRevision{}, err
	}
	context = context.WithCooperativeCallable(
		facet,
		observation.Cooperative(),
	)
	statement, requests, err := genericcapability.Build(
		context,
		builder.emitter,
		artifact,
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements := []tsgo.Statement{statement}
	placement, dependencies, requirements, err :=
		s.consumeArtifactRequests(
			owner,
			api.CombineRequests(requests, observation.Requests()),
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

func (s *programSession) compilationGeneratedArtifactBuilder(
	artifact *api.GeneratedArtifact,
	kind string,
) (*targetFileBuilder, error) {
	if existing := s.builders[artifact.OutputPath()]; existing != nil {
		return existing, nil
	}
	sourcePackage, ok := deterministicSupportPackage(s.source.Packages())
	if !ok {
		return nil, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: kind + " support has no deterministic source context",
		}
	}
	emitter := s.emitters[sourcePackage]
	if emitter == nil {
		return nil, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: kind + " support package has no emitter",
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
