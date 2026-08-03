package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) ObserveCooperativeCallable(
	context api.Context,
	facet api.CallableFacet,
) (api.CooperativeCallableObservation, error) {
	return s.observeCooperativeCallable(
		context.ArtifactOwner(),
		&context,
		facet,
	)
}

func (s *programSession) observeCooperativeCallable(
	consumer api.ArtifactOwner,
	context *api.Context,
	facet api.CallableFacet,
) (api.CooperativeCallableObservation, error) {
	if !consumer.Valid() || !facet.Valid() {
		return api.CooperativeCallableObservation{}, &ScheduleError{
			Reason: "cooperative callable facet is invalid",
		}
	}
	if source, ok := facet.Owner().Source(); ok && source != nil &&
		s.source.EnvironmentForTypes(source.Pkg()) != nil {
		function, callable := source.(*types.Func)
		if callable {
			var profileRequests []api.RootRequest
			if context != nil && function.Signature().Recv() != nil {
				effect, selected, requests, err :=
					providerboundary.ResolveStatefulMethodEffect(
						*context,
						function,
					)
				if err != nil {
					return api.CooperativeCallableObservation{}, err
				}
				profileRequests = requests
				if selected {
					return api.NewCooperativeCallableObservation(
						effect.MaySuspend(),
						profileRequests...,
					)
				}
			}
			effect, providerOwned, err :=
				s.registry.ProviderCallableEffect(function)
			if err != nil {
				return api.CooperativeCallableObservation{}, err
			}
			if providerOwned {
				return api.NewCooperativeCallableObservation(
					effect.MaySuspend(),
					profileRequests...,
				)
			}
		}
	}
	cooperative := false
	for _, requirement := range s.requirements.appliedFor(
		facet.Owner(),
	) {
		selected, selectedCooperative :=
			requirement.CooperativeCallable()
		if !selectedCooperative {
			continue
		}
		if selected.Owner() != facet.Owner() {
			return api.CooperativeCallableObservation{}, &ScheduleError{
				Object: facet.Owner().Name(),
				Reason: "cooperative callable has inconsistent ownership",
			}
		}
		if selected == facet {
			cooperative = true
			break
		}
	}
	var requests []api.RootRequest
	if consumer != facet.Owner() {
		switch facet.Kind() {
		case api.CallableFacetSource,
			api.CallableFacetABI,
			api.CallableFacetInterfaceMethod,
			api.CallableFacetGenericCapability,
			api.CallableFacetGenericOperation:
			request, err := api.NewOwnedArtifactDependencyRequest(
				facet.Owner(),
				api.ArtifactFacetCallableSignature,
			)
			if err != nil {
				return api.CooperativeCallableObservation{}, err
			}
			requests = append(requests, request)
		case api.CallableFacetFunctionLiteral,
			api.CallableFacetPackageInitializer:
			return api.CooperativeCallableObservation{}, &ScheduleError{
				Object: facet.Owner().Name(),
				Reason: "lexical callable facet escaped its source artifact",
			}
		default:
			return api.CooperativeCallableObservation{}, &ScheduleError{
				Object: facet.Owner().Name(),
				Reason: "cooperative callable facet kind is invalid",
			}
		}
	}
	return api.NewCooperativeCallableObservation(
		cooperative,
		requests...,
	)
}

func (s *programSession) validateCallableContractArtifact(
	artifact *api.GeneratedArtifact,
) error {
	signature, ok := callableContractSignature(artifact)
	binding, found := s.registry.GeneratedArtifact(
		artifact.Kind(),
		artifact.ArtifactKey(),
	)
	boundSignature, bound := callableContractSignature(binding)
	if !ok ||
		!found ||
		binding != artifact ||
		!bound ||
		!types.Identical(boundSignature, signature) ||
		artifact.Placement() != api.GeneratedArtifactPlacementContract {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable contract artifact has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructCallableContractArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable contract reconstructed after target files were sealed",
		}
	}
	if err := s.validateCallableContractArtifact(artifact); err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	applied := s.requirements.appliedFor(owner)
	if len(applied) == 0 && s.requirementRemovalOwner == owner {
		contract, err := s.callableContract(false)
		if err != nil {
			return err
		}
		if err := s.commitArtifactContract(owner, contract, nil); err != nil {
			return err
		}
		s.artifacts.DiscardDirty(owner)
		return nil
	}
	cooperative, err := callableContractRequirements(
		applied,
		artifact,
	)
	if err != nil {
		return err
	}
	contract, err := s.callableContract(cooperative)
	if err != nil {
		return err
	}
	if err := s.commitArtifactContract(owner, contract, nil); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	return nil
}

func (s *programSession) ensureCallableContractBaseline(
	artifact *api.GeneratedArtifact,
) error {
	owner := api.MustGeneratedArtifactOwner(artifact)
	if s.artifacts.FacetRevision(
		owner,
		api.ArtifactFacetCallableSignature,
	) != 0 {
		return nil
	}
	if err := s.validateCallableContractArtifact(artifact); err != nil {
		return err
	}
	contract, err := s.callableContract(false)
	if err != nil {
		return err
	}
	return s.commitArtifactContract(owner, contract, nil)
}

func (s *programSession) callableContract(
	cooperative bool,
) (artifactstate.Contract, error) {
	encoded, err := s.callableSignatureFacet(cooperative)
	if err != nil {
		return artifactstate.Contract{}, err
	}
	return artifactstate.NewContractFacet(
		api.ArtifactFacetCallableSignature,
		encoded,
	)
}

func (s *programSession) callableSignatureFacet(
	cooperative bool,
) ([]byte, error) {
	var result tsgo.TypeNode = s.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
	if cooperative {
		result = callable.PromiseResult(s.factory, result)
	}
	return tsgo.EncodeNode(
		s.factory.FunctionTypeNode(nil, nil, result),
	)
}

func callableContractRequirements(
	requirements []api.DeclarationRequirement,
	artifact *api.GeneratedArtifact,
) (bool, error) {
	definitions := 0
	cooperative := false
	for _, requirement := range requirements {
		if selected, ok := callableContractDefinition(
			requirement,
			artifact.Kind(),
		); ok {
			if selected != artifact {
				return false, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "callable contract received a foreign definition",
				}
			}
			definitions++
			continue
		}
		facet, ok := requirement.CooperativeCallable()
		selected, callable := callableContractFacet(facet)
		if !ok || !callable || selected != artifact || cooperative {
			return false, &ScheduleError{
				Object: artifact.TargetName(),
				Reason: "callable contract received a foreign requirement",
			}
		}
		cooperative = true
	}
	if definitions != 1 {
		return false, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "callable contract requires exactly one definition request",
		}
	}
	return cooperative, nil
}

func callableContractSignature(
	artifact *api.GeneratedArtifact,
) (*types.Signature, bool) {
	if artifact == nil {
		return nil, false
	}
	switch artifact.Kind() {
	case api.GeneratedArtifactCallableABI:
		return artifact.CallableABI()
	case api.GeneratedArtifactInterfaceMethodCallable:
		return artifact.InterfaceMethodCallableSignature()
	default:
		return nil, false
	}
}

func callableContractDefinition(
	requirement api.DeclarationRequirement,
	kind api.GeneratedArtifactKind,
) (*api.GeneratedArtifact, bool) {
	switch kind {
	case api.GeneratedArtifactCallableABI:
		return requirement.CallableABI()
	case api.GeneratedArtifactInterfaceMethodCallable:
		return requirement.InterfaceMethodCallable()
	default:
		return nil, false
	}
}

func callableContractFacet(
	facet api.CallableFacet,
) (*api.GeneratedArtifact, bool) {
	if artifact, ok := facet.ABI(); ok {
		return artifact, true
	}
	return facet.InterfaceMethod()
}
