package providerinterfacebridge

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type profileCapabilitySelection struct {
	name            string
	rawGuardName    string
	targetBridge    api.ProviderProfileBridgeReference
	targetCanonical api.InterfaceContractReference
}

func selectProfileCapabilities(
	context api.Context,
	bridgeName string,
	base *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
	contracts []ProfileCapabilityContract,
) ([]profileCapabilitySelection, []api.RootRequest, error) {
	if len(contracts) == 0 {
		return nil, nil, nil
	}
	baseIdentity, err := gostdlibsource.ObjectIdentity(base.Obj())
	if err != nil {
		return nil, nil, err
	}
	baseContract, ok := base.Underlying().(*types.Interface)
	if !ok {
		return nil, nil, shapeError(bridgeName, "profile capability base is invalid")
	}
	selected := make([]profileCapabilitySelection, 0, len(contracts))
	var requests []api.RootRequest
	for _, contract := range contracts {
		target, targetProfile, targetProfiled :=
			contract.Target.ProviderProfileInterfaceBridge()
		if !targetProfiled || len(targetProfile) == 0 {
			return nil, nil, shapeError(
				bridgeName,
				"profile capability target bridge is invalid",
			)
		}
		targetContract, ok := target.Underlying().(*types.Interface)
		if !ok || !types.Implements(target, baseContract.Complete()) ||
			types.Implements(base, targetContract.Complete()) {
			return nil, nil, shapeError(
				bridgeName,
				"profile capability is not a strict extension of its base",
			)
		}
		targetIdentity, err := gostdlibsource.ObjectIdentity(target.Obj())
		if err != nil {
			return nil, nil, err
		}
		if !profileInterfaceCertified(profile, baseIdentity) ||
			!profileInterfaceCertified(targetProfile, targetIdentity) {
			return nil, nil, shapeError(
				bridgeName,
				"profile capability target has no complete bridge certificate",
			)
		}
		targetBridge, found, err := context.Names().ProviderProfileInterfaceBridge(
			target,
			targetProfile,
		)
		if err != nil {
			return nil, nil, err
		}
		if !found {
			return nil, nil, shapeError(
				bridgeName,
				"profile capability target bridge is absent",
			)
		}
		canonical, err := context.Names().InterfaceContract(target)
		if err != nil {
			return nil, nil, err
		}
		name, err := api.ProviderProfileCapabilityName(
			bridgeName,
			contract.Target.ArtifactKey(),
		)
		if err != nil {
			return nil, nil, err
		}
		selected = append(selected, profileCapabilitySelection{
			name:            name,
			rawGuardName:    name + "$raw",
			targetBridge:    targetBridge,
			targetCanonical: canonical,
		})
		requests = append(requests, targetBridge.Requests()...)
		requests = append(requests, canonical.Requests()...)
	}
	return selected, api.CombineRequests(requests), nil
}

func profileInterfaceCertified(
	profile []gostdlib.ProviderCallableProfileInterface,
	identity string,
) bool {
	for _, selected := range profile {
		if !selected.Valid() ||
			selected.ProviderInterface().Mode() !=
				gostdlib.ProviderInterfaceModeBridge {
			continue
		}
		if selected.SourceIdentity() == identity {
			return true
		}
	}
	return false
}
