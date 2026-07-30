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
			initializer = context.Factory().Identifier(reference.Name())
			requests = reference.Requests()
		}
		return []tsgo.Statement{
			interfacemethodtoken.Build(
				builder.context.Factory(),
				artifact.TargetName(),
				[]tsgo.ModifierLike{
					builder.context.Factory().ExportKeyword(),
				},
				initializer,
			),
		}, requests, nil
	case api.GeneratedArtifactInterfaceDynamicTypeToken:
		return []tsgo.Statement{
			interfacedynamictype.Build(
				builder.context.Factory(),
				artifact.TargetName(),
				[]tsgo.ModifierLike{
					builder.context.Factory().ExportKeyword(),
				},
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
			adapterContracts,
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

func (e *emitter) ObservePointerRepresentation(
	consumer api.ArtifactOwner,
	artifact *api.GeneratedArtifact,
	carrierDemand bool,
) (api.PointerRepresentationObservation, error) {
	if e.pointer == nil {
		return api.PointerRepresentationObservation{}, &api.InvariantError{
			Reason: "pointer representation resolver is unavailable",
		}
	}
	return e.pointer.ObservePointerRepresentation(
		consumer,
		artifact,
		carrierDemand,
	)
}

func (s *programSession) ObservePointerRepresentation(
	consumer api.ArtifactOwner,
	artifact *api.GeneratedArtifact,
	carrierDemand bool,
) (api.PointerRepresentationObservation, error) {
	if !consumer.Valid() {
		return api.PointerRepresentationObservation{}, &ScheduleError{
			Reason: "pointer-representation consumer is invalid",
		}
	}
	if err := s.ensurePointerRepresentationBaseline(artifact); err != nil {
		return api.PointerRepresentationObservation{}, err
	}
	representation, err := api.DefaultPointerRepresentation(artifact)
	if err != nil {
		return api.PointerRepresentationObservation{}, err
	}
	if carrierDemand {
		representation = api.PointerRepresentationCarrierCanonical
	} else {
		for _, requirement := range s.requirements.appliedFor(
			api.MustGeneratedArtifactOwner(artifact),
		) {
			selected, carrier, ok := requirement.PointerRepresentation()
			if !ok || selected != artifact {
				return api.PointerRepresentationObservation{}, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "pointer representation has a foreign requirement",
				}
			}
			if carrier {
				representation =
					api.PointerRepresentationCarrierCanonical
			}
		}
	}
	var requests []api.RootRequest
	if carrierDemand {
		request, err := api.NewPointerRepresentationRequest(artifact, true)
		if err != nil {
			return api.PointerRepresentationObservation{}, err
		}
		requests = append(requests, request)
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	if consumer != owner {
		dependency, err := api.NewGeneratedArtifactDependencyRequest(
			artifact,
			api.ArtifactFacetInstanceTypeSurface,
		)
		if err != nil {
			return api.PointerRepresentationObservation{}, err
		}
		requests = append(requests, dependency)
	}
	return api.NewPointerRepresentationObservation(
		representation,
		requests...,
	)
}

func (s *programSession) validatePointerRepresentationArtifact(
	artifact *api.GeneratedArtifact,
) error {
	pointer, ok := artifact.PointerRepresentation()
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactPointerRepresentation,
		artifact.ArtifactKey(),
	)
	bound, boundOK := binding.PointerRepresentation()
	if !ok ||
		!found ||
		binding != artifact ||
		!boundOK ||
		!types.Identical(bound, pointer) ||
		artifact.Placement() != api.GeneratedArtifactPlacementContract {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "pointer representation has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) ensurePointerRepresentationBaseline(
	artifact *api.GeneratedArtifact,
) error {
	if err := s.validatePointerRepresentationArtifact(artifact); err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	if s.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetInstanceTypeSurface,
	) != 0 {
		return nil
	}
	representation, err := api.DefaultPointerRepresentation(artifact)
	if err != nil {
		return err
	}
	contract, err := pointerRepresentationContract(representation)
	if err != nil {
		return err
	}
	return s.artifacts.Commit(owner, contract, nil)
}

func (s *programSession) reconstructPointerRepresentationArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "pointer representation reconstructed after target files were sealed",
		}
	}
	if err := s.validatePointerRepresentationArtifact(artifact); err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	representation, err := api.SelectPointerRepresentation(
		artifact,
		s.requirements.appliedFor(owner),
	)
	if err != nil {
		return err
	}
	contract, err := pointerRepresentationContract(representation)
	if err != nil {
		return err
	}
	if err := s.artifacts.Commit(owner, contract, nil); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	return nil
}

func pointerRepresentationContract(
	representation api.PointerRepresentation,
) (artifactstate.Contract, error) {
	return artifactstate.NewContractFacet(
		api.ArtifactFacetInstanceTypeSurface,
		[]byte(representation.String()),
	)
}

func compareGenericRepresentationRequirements(
	left api.DeclarationRequirement,
	right api.DeclarationRequirement,
) int {
	leftOwner, leftParameter, leftFacet, leftOK :=
		left.GenericRepresentation()
	rightOwner, rightParameter, rightFacet, rightOK :=
		right.GenericRepresentation()
	switch {
	case !leftOK && rightOK:
		return -1
	case leftOK && !rightOK:
		return 1
	case !leftOK:
		return 0
	}
	leftIndex, leftIndexed :=
		api.GenericDeclarationParameterIndex(leftOwner, leftParameter)
	rightIndex, rightIndexed :=
		api.GenericDeclarationParameterIndex(rightOwner, rightParameter)
	switch {
	case !leftIndexed && rightIndexed:
		return -1
	case leftIndexed && !rightIndexed:
		return 1
	case leftIndex < rightIndex:
		return -1
	case leftIndex > rightIndex:
		return 1
	case leftFacet < rightFacet:
		return -1
	case leftFacet > rightFacet:
		return 1
	default:
		return 0
	}
}

func (s *programSession) ResolveGenericRepresentationProfile(
	declaration types.Object,
) (api.GenericRepresentationProfile, bool, error) {
	owner := api.GenericDeclarationOrigin(declaration)
	if owner == nil || len(api.GenericDeclarationParameters(owner)) == 0 {
		return api.GenericRepresentationProfile{}, false, nil
	}
	if _, ok := s.sites[owner]; ok {
		profile, err := api.SelectGenericRepresentationProfile(
			owner,
			s.requirements.appliedFor(api.MustSourceArtifactOwner(owner)),
		)
		return profile, err == nil, err
	}
	sourcePackage := s.source.EnvironmentForTypes(owner.Pkg())
	if sourcePackage == nil {
		return api.GenericRepresentationProfile{}, false, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic representation owner has no declaration",
		}
	}
	var requirements []api.DeclarationRequirement
	if builder := s.environmentBuilders[sourcePackage]; builder != nil {
		requirements = builder.environmentRequirements(owner)
	}
	profile, err := api.SelectGenericRepresentationProfile(
		owner,
		requirements,
	)
	return profile, err == nil, err
}
