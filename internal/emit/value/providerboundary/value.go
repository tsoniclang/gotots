package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func FromProviderValue(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return fromProviderValueSelected(
		context,
		children,
		owner,
		ownerBridge,
		nil,
		sourceType,
		value,
	)
}

func fromProviderValueSelected(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	converted, scalar, changed, err := fromProviderScalar(
		context,
		sourceType,
		value,
	)
	if err != nil || scalar {
		return converted, changed, err
	}
	selected, ok := types.Unalias(sourceType).(*types.Named)
	if ok && selected.Obj() != nil {
		_, profileOwned, profileErr :=
			providerProfileInterfaceCertificate(selected, profile)
		if profileErr != nil {
			return api.ExpressionEmission{}, false, profileErr
		}
		if profileOwned {
			reference, found, referenceErr :=
				context.Names().ProviderProfileInterfaceBridge(selected, profile)
			if referenceErr != nil || !found {
				if referenceErr != nil {
					return api.ExpressionEmission{}, false, referenceErr
				}
				return api.ExpressionEmission{}, false, boundaryInvariant(
					context,
					"provider profile-interface bridge is absent",
				)
			}
			return bridgeEmission(
				context,
				value,
				reference.Bridge().Name(),
				api.ProviderBridgeFromMember,
				reference.Requests(),
			)
		}
		if owner != nil && types.Identical(selected, owner) {
			if ownerBridge == "" {
				return api.ExpressionEmission{}, false, &api.InvariantError{
					Role:   context.Role(),
					Reason: "provider boundary self-bridge name is empty",
				}
			}
			return bridgeEmission(
				context,
				value,
				ownerBridge,
				api.ProviderBridgeFromMember,
				nil,
			)
		}
		reference, provider, err := context.Names().ProviderInterfaceBridge(selected)
		if err != nil {
			return api.ExpressionEmission{}, false, err
		}
		if provider {
			return bridgeEmission(
				context,
				value,
				reference.Name(),
				api.ProviderBridgeFromMember,
				reference.Requests(),
			)
		}
	}
	if signature, callableType, model := callableType(sourceType); callableType {
		return fromProviderCallableSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			signature,
			model,
			value,
		)
	}
	return value, false, nil
}

func ToProviderValue(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return toProviderValueSelected(
		context,
		children,
		owner,
		ownerBridge,
		nil,
		sourceType,
		value,
	)
}

func toProviderValueSelected(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	converted, scalar, changed, err := toProviderScalar(
		context,
		sourceType,
		value,
	)
	if err != nil || scalar {
		return converted, changed, err
	}
	selected, ok := types.Unalias(sourceType).(*types.Named)
	if ok && selected.Obj() != nil {
		_, profileOwned, profileErr :=
			providerProfileInterfaceCertificate(selected, profile)
		if profileErr != nil {
			return api.ExpressionEmission{}, false, profileErr
		}
		if profileOwned {
			reference, found, referenceErr :=
				context.Names().ProviderProfileInterfaceBridge(selected, profile)
			if referenceErr != nil || !found {
				if referenceErr != nil {
					return api.ExpressionEmission{}, false, referenceErr
				}
				return api.ExpressionEmission{}, false, boundaryInvariant(
					context,
					"provider profile-interface bridge is absent",
				)
			}
			return bridgeEmission(
				context,
				value,
				reference.Bridge().Name(),
				api.ProviderBridgeToMember,
				reference.Requests(),
			)
		}
		if owner != nil && types.Identical(selected, owner) {
			if ownerBridge == "" {
				return api.ExpressionEmission{}, false, &api.InvariantError{
					Role:   context.Role(),
					Reason: "provider boundary self-bridge name is empty",
				}
			}
			return bridgeEmission(
				context,
				value,
				ownerBridge,
				api.ProviderBridgeToMember,
				nil,
			)
		}
		reference, provider, err := context.Names().ProviderInterfaceBridge(selected)
		if err != nil {
			return api.ExpressionEmission{}, false, err
		}
		if provider {
			return bridgeEmission(
				context,
				value,
				reference.Name(),
				api.ProviderBridgeToMember,
				reference.Requests(),
			)
		}
	}
	if signature, callableType, model := callableType(sourceType); callableType {
		return toProviderCallableSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			signature,
			model,
			value,
		)
	}
	return value, false, nil
}

func bridgeEmission(
	context api.Context,
	value api.ExpressionEmission,
	name string,
	member string,
	requests []api.RootRequest,
) (api.ExpressionEmission, bool, error) {
	target, err := api.NewExpressionEmission(
		value.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(name),
				nil,
				context.Factory().Identifier(member),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{value.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(value.Requests(), requests),
	)
	return target, true, err
}
