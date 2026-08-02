package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (s *programSession) ObserveRecoveryCallable(
	context api.Context,
	facet api.CallableFacet,
) (api.RecoveryCallableObservation, error) {
	if !facet.Valid() || !context.ArtifactOwner().Valid() {
		return api.RecoveryCallableObservation{}, &ScheduleError{
			Reason: "recovery callable observation is invalid",
		}
	}
	function, ok := recoveryFacetSource(facet)
	if !ok {
		return api.RecoveryCallableObservation{}, &ScheduleError{
			Object: facet.Owner().Name(),
			Reason: "recovery observation has no exact source callable",
		}
	}
	if s.source.EnvironmentForTypes(function.Pkg()) != nil {
		_, selected, err := context.Names().RecoveryCallable(function)
		if err != nil {
			return api.RecoveryCallableObservation{}, err
		}
		return api.NewRecoveryCallableObservation(selected)
	}
	recovery, err := sourceCallableRecoveryRequirement(
		function,
		s.requirements.appliedFor(api.MustSourceArtifactOwner(function)),
	)
	if err != nil {
		return api.RecoveryCallableObservation{}, err
	}
	var requests []api.RootRequest
	if context.ArtifactOwner() != api.MustSourceArtifactOwner(function) {
		request, requestErr := api.NewOwnedArtifactDependencyRequest(
			api.MustSourceArtifactOwner(function),
			api.ArtifactFacetCallableRecovery,
		)
		if requestErr != nil {
			return api.RecoveryCallableObservation{}, requestErr
		}
		requests = []api.RootRequest{request}
	}
	return api.NewRecoveryCallableObservation(recovery, requests...)
}

func recoveryFacetSource(
	facet api.CallableFacet,
) (*types.Func, bool) {
	if function, ok := facet.SourceFunction(); ok {
		return function.Origin(), true
	}
	if profile, ok := facet.GenericProfile(); ok {
		return profile.Owner().Origin(), true
	}
	if profile, _, ok := facet.GenericProfileABI(); ok {
		return profile.Owner().Origin(), true
	}
	return nil, false
}

func sourceCallableRecoveryRequirement(
	function *types.Func,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	owner := api.MustSourceArtifactOwner(function)
	recovery := false
	for _, requirement := range requirements {
		requirementOwner, _, callable, control, ok :=
			requirement.CallableControl()
		if !ok {
			continue
		}
		if requirementOwner != owner {
			return false, &ScheduleError{
				Object: function.Name(),
				Reason: "recovery requirement has foreign ownership",
			}
		}
		if control != api.CallableControlRecovery {
			continue
		}
		if callable == nil {
			recovery = true
			continue
		}
		if _, direct := callable.(*ast.FuncDecl); direct {
			recovery = true
		}
	}
	return recovery, nil
}
