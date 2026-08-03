package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
)

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
	return s.commitArtifactContract(owner, contract, nil)
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
	if err := s.commitArtifactContract(owner, contract, nil); err != nil {
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
		requirements = s.requirements.appliedFor(
			api.MustSourceArtifactOwner(owner),
		)
	}
	profile, err := api.SelectGenericRepresentationProfile(
		owner,
		requirements,
	)
	return profile, err == nil, err
}
