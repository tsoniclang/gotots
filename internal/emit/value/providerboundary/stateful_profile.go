package providerboundary

import (
	"go/types"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type StatefulProfileSelection struct {
	profile       gostdlib.ProviderStatefulProfile
	typeArguments []types.Type
	requests      []api.RootRequest
}

type StatefulMethodBoundary struct {
	method     gostdlib.ProviderStatefulProfileMethod
	interfaces []gostdlib.ProviderCallableProfileInterface
	requests   []api.RootRequest
}

func ResolveStatefulProfile(
	context api.Context,
	source *types.Named,
) (StatefulProfileSelection, bool, error) {
	if source == nil || source.Obj() == nil {
		return StatefulProfileSelection{}, false, boundaryInvariant(
			context,
			"provider stateful-profile source is invalid",
		)
	}
	candidates, profiled, err :=
		context.Names().ProviderStatefulProfileCandidates(source.Origin().Obj())
	if err != nil || !profiled {
		return StatefulProfileSelection{}, false, err
	}
	analyzer := &profileBoundaryAnalyzer{
		context:  context,
		nodes:    make(map[string]*profileBoundaryInterface),
		building: make(map[string]bool),
		visited:  make(map[profileTypeVisit]struct{}),
	}
	if err := analyzer.collectType(source.Underlying(), "", nil); err != nil {
		return StatefulProfileSelection{}, false, err
	}
	affected := analyzer.affectedInterfaces()
	if len(affected) == 0 {
		return StatefulProfileSelection{
			requests: api.CombineRequests(analyzer.requests),
		}, false, nil
	}
	var matches []StatefulProfileSelection
	var mismatches []string
	for _, candidate := range candidates {
		selected, matched, mismatch, matchErr :=
			matchStatefulProfileCandidate(analyzer, candidate)
		if matchErr != nil {
			return StatefulProfileSelection{}, false, matchErr
		}
		if matched {
			matches = append(matches, selected)
		} else {
			mismatches = append(
				mismatches,
				candidate.Profile().ProfileKey()+": "+mismatch,
			)
		}
	}
	if len(matches) != 1 {
		reason := "provider concrete type has no exact certified stateful profile"
		if len(matches) > 1 {
			reason = "provider concrete type has multiple exact certified stateful profiles"
		}
		identity, identityErr := sourceObjectIdentity(source.Obj())
		if identityErr != nil {
			return StatefulProfileSelection{}, false, identityErr
		}
		reason += " for " + identity
		reason += " with affected interfaces [" +
			strings.Join(sortedIdentitySet(affected), ", ") + "]"
		if len(mismatches) != 0 {
			reason += " (" + strings.Join(mismatches, "; ") + ")"
		}
		return StatefulProfileSelection{}, false, boundaryInvariant(context, reason)
	}
	selected := matches[0]
	selected.requests = api.CombineRequests(
		analyzer.requests,
		selected.requests,
	)
	return selected, true, nil
}

func matchStatefulProfileCandidate(
	analyzer *profileBoundaryAnalyzer,
	candidate api.ProviderStatefulProfileCandidate,
) (StatefulProfileSelection, bool, string, error) {
	profile := candidate.Profile()
	profileInterfaces := profile.Interfaces()
	if len(profileInterfaces) != len(analyzer.nodes) {
		return StatefulProfileSelection{}, false, "retained interface set differs", nil
	}
	keys := make(
		[]gostdlib.ProviderCallableProfileKeyInterface,
		0,
		len(profileInterfaces),
	)
	for _, selected := range profileInterfaces {
		node := analyzer.nodes[selected.SourceIdentity()]
		matched, mismatch, keyInterface, err := matchProfileInterface(
			node,
			selected.ProviderInterface(),
		)
		if err != nil {
			return StatefulProfileSelection{}, false, "", err
		}
		if !matched {
			return StatefulProfileSelection{}, false,
				"interface ABI differs for " + selected.SourceIdentity() +
					": " + mismatch, nil
		}
		keys = append(keys, keyInterface)
	}
	key, err := gostdlib.BuildProviderCallableProfileKey(keys, nil)
	if err != nil {
		return StatefulProfileSelection{}, false, "", err
	}
	if key != profile.ProfileKey() {
		return StatefulProfileSelection{}, false, "profile key differs", nil
	}
	typeArguments := candidate.TypeArguments()
	profileTypeArguments := profile.TypeArguments()
	if len(typeArguments) != len(profileTypeArguments) {
		return StatefulProfileSelection{}, false, "interface type arguments differ", nil
	}
	for index, selected := range profileTypeArguments {
		identity, err := namedInterfaceIdentity(typeArguments[index])
		if err != nil {
			return StatefulProfileSelection{}, false, "", err
		}
		if identity != selected {
			return StatefulProfileSelection{}, false,
				"interface type-argument order differs", nil
		}
	}
	return StatefulProfileSelection{
		profile:       profile,
		typeArguments: typeArguments,
	}, true, "", nil
}

func ResolveStatefulMethodEffect(
	context api.Context,
	method *types.Func,
) (gostdlib.EffectKind, bool, []api.RootRequest, error) {
	if method == nil || method.Origin() == nil {
		return gostdlib.EffectInvalid, false, nil, boundaryInvariant(
			context,
			"provider stateful-profile method is invalid",
		)
	}
	signature, ok := method.Origin().Type().(*types.Signature)
	if !ok {
		return gostdlib.EffectInvalid, false, nil, boundaryInvariant(
			context,
			"provider stateful-profile method signature is invalid",
		)
	}
	selection, selected, err := ResolveStatefulMethodBoundary(
		context,
		method,
		signature,
	)
	if err != nil || !selected {
		return gostdlib.EffectInvalid, false, selection.Requests(), err
	}
	return selection.Method().Effect(), true, selection.Requests(), nil
}

func ResolveStatefulMethodBoundary(
	context api.Context,
	method *types.Func,
	signature *types.Signature,
) (StatefulMethodBoundary, bool, error) {
	if method == nil || method.Origin() == nil || signature == nil {
		return StatefulMethodBoundary{}, false, boundaryInvariant(
			context,
			"provider stateful-profile method boundary is invalid",
		)
	}
	owner := api.MethodReceiverTypeName(method.Origin())
	if owner == nil {
		return StatefulMethodBoundary{}, false, nil
	}
	named, ok := types.Unalias(owner.Type()).(*types.Named)
	if !ok {
		return StatefulMethodBoundary{}, false, nil
	}
	selection, selected, err := ResolveStatefulProfile(context, named)
	if err != nil || !selected {
		return StatefulMethodBoundary{
			requests: selection.Requests(),
		}, false, err
	}
	identity, err := sourceObjectIdentity(method.Origin())
	if err != nil {
		return StatefulMethodBoundary{}, false, err
	}
	profileMethod, ok := selection.profile.Method(identity)
	if !ok || !profileMethod.Effect().Valid() {
		return StatefulMethodBoundary{}, false, boundaryInvariant(
			context,
			"selected provider stateful profile omits method "+identity,
		)
	}
	boundary, err := analyzeCallableProfileBoundary(context, signature)
	if err != nil {
		return StatefulMethodBoundary{}, false, err
	}
	return StatefulMethodBoundary{
		method:     profileMethod,
		interfaces: selection.profile.Interfaces(),
		requests: api.CombineRequests(
			selection.Requests(),
			boundary.analyzer.requests,
		),
	}, true, nil
}

func (s StatefulProfileSelection) Profile() gostdlib.ProviderStatefulProfile {
	return s.profile
}

func (s StatefulProfileSelection) TypeArguments() []types.Type {
	return append([]types.Type(nil), s.typeArguments...)
}

func (s StatefulProfileSelection) Requests() []api.RootRequest {
	return api.CombineRequests(s.requests)
}

func (s StatefulMethodBoundary) Method() gostdlib.ProviderStatefulProfileMethod {
	return s.method
}

func (s StatefulMethodBoundary) Interfaces() []gostdlib.ProviderCallableProfileInterface {
	return append(
		[]gostdlib.ProviderCallableProfileInterface(nil),
		s.interfaces...,
	)
}

func (s StatefulMethodBoundary) Requests() []api.RootRequest {
	return api.CombineRequests(s.requests)
}

func namedInterfaceIdentity(source types.Type) (string, error) {
	named, ok := types.Unalias(source).(*types.Named)
	if !ok || named.Obj() == nil {
		return "", &api.NameError{
			Reason: "provider profile type argument is not a named interface",
		}
	}
	if _, ok := named.Underlying().(*types.Interface); !ok {
		return "", &api.NameError{
			Name:   named.Obj().Name(),
			Reason: "provider profile type argument is not an interface",
		}
	}
	return sourceObjectIdentity(named.Obj())
}
