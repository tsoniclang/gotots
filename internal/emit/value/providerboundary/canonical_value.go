package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func providerProfileInterfaceCertificate(
	source *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
) (gostdlib.ProviderCallableProfileInterface, bool, error) {
	if source == nil || source.Obj() == nil || len(profile) == 0 {
		return gostdlib.ProviderCallableProfileInterface{}, false, nil
	}
	identity, err := sourceObjectIdentity(source.Obj())
	if err != nil {
		return gostdlib.ProviderCallableProfileInterface{}, false, err
	}
	for _, selected := range profile {
		if selected.SourceIdentity() == identity {
			return selected, true, nil
		}
	}
	return gostdlib.ProviderCallableProfileInterface{}, false, nil
}

func CanonicalProfileValue(
	context api.Context,
	children api.ChildEmitter,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	raw api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	generated, crossedFromProvider, err := fromProviderValueSelected(
		context,
		children,
		nil,
		"",
		nil,
		sourceType,
		raw,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !crossedFromProvider {
		return api.ExpressionEmission{}, boundaryInvariant(
			context,
			"canonical provider value has no generated-source bridge",
		)
	}
	provider, crossedToProfile, err := toProviderValueSelected(
		context,
		children,
		nil,
		"",
		profile,
		sourceType,
		generated,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !crossedToProfile {
		return api.ExpressionEmission{}, boundaryInvariant(
			context,
			"canonical provider value has no profile bridge",
		)
	}
	return provider, nil
}

func fromProviderCanonicalValue(
	context api.Context,
	children api.ChildEmitter,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return fromProviderValueWithPolicy(
		context,
		children,
		nil,
		"",
		profile,
		sourceType,
		value,
		providerBoundaryLeafProfileOnly,
	)
}

func toProviderCanonicalValue(
	context api.Context,
	children api.ChildEmitter,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return toProviderValueWithPolicy(
		context,
		children,
		nil,
		"",
		profile,
		sourceType,
		value,
		providerBoundaryLeafProfileOnly,
	)
}
