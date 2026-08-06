package providerinterfacebridge

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitProfileReverseCapabilityMethods(
	context api.Context,
	providerContext api.Context,
	children api.ChildEmitter,
	bridgeName string,
	bridgeType *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
	capabilities []capabilitySelection,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	groups := make(map[string][]selectedCapabilityMethod)
	var requests []api.RootRequest
	for _, capability := range capabilities {
		for _, method := range capability.methods {
			emission, err := prepareProfileReverseMethod(
				context,
				providerContext,
				children,
				bridgeName,
				bridgeType,
				profile,
				method.method,
				method.certificate,
				context.Factory().Identifier(capability.fieldName),
			)
			if err != nil {
				return nil, nil, err
			}
			groups[emission.name] = append(
				groups[emission.name],
				selectedCapabilityMethod{
					fieldName: capability.fieldName,
					emission:  emission,
				},
			)
			requests = append(requests, emission.requests...)
		}
	}
	return emitCapabilityMethodGroups(
		context,
		bridgeName,
		groups,
		requests,
	)
}
