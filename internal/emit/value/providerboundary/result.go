package providerboundary

import (
	"go/types"
	"slices"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func FromProviderResults(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	results *types.Tuple,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, _, err := fromProviderResults(
		context,
		children,
		owner,
		ownerBridge,
		results,
		emission,
	)
	return target, err
}

func FromProviderProfileResults(
	context api.Context,
	children api.ChildEmitter,
	results *types.Tuple,
	profile gostdlib.ProviderCallableProfile,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, _, err := fromProviderResultsSelected(
		context,
		children,
		nil,
		"",
		profile.CanonicalResults(),
		profile.Interfaces(),
		results,
		emission,
	)
	return target, err
}

func FromProviderProfileResultsForBridge(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	results *types.Tuple,
	profile []gostdlib.ProviderCallableProfileInterface,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, _, err := fromProviderResultsSelected(
		context,
		children,
		owner,
		ownerBridge,
		nil,
		profile,
		results,
		emission,
	)
	return target, err
}

func fromProviderResults(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	results *types.Tuple,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return fromProviderResultsSelected(
		context,
		children,
		owner,
		ownerBridge,
		nil,
		nil,
		results,
		emission,
	)
}

func fromProviderResultsSelected(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	canonical []int,
	profile []gostdlib.ProviderCallableProfileInterface,
	results *types.Tuple,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if results == nil || results.Len() == 0 {
		return emission, false, nil
	}
	if results.Len() == 1 {
		if slices.Contains(canonical, 0) {
			return fromProviderCanonicalValue(
				context,
				children,
				profile,
				results.At(0).Type(),
				emission,
			)
		}
		return fromProviderValueSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			results.At(0).Type(),
			emission,
		)
	}
	temporary, err := context.Names().Temporary(api.TemporaryMultipleResults)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	before := append(emission.Before(), context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(temporary),
					nil,
					nil,
					emission.Value(),
				),
			},
			tsgo.NodeFlagsConst,
		),
	))
	elements := make([]tsgo.Expression, 0, results.Len())
	requests := emission.Requests()
	changed := false
	for index := range results.Len() {
		value := context.Factory().ElementAccessExpression(
			context.Factory().Identifier(temporary),
			nil,
			context.Factory().NumericLiteral(
				strconv.Itoa(index),
				tsgo.TokenFlagsNone,
			),
			tsgo.NodeFlagsNone,
		)
		converted := api.DirectExpression(value)
		convertedValue := false
		var convertErr error
		if slices.Contains(canonical, index) {
			converted, convertedValue, convertErr = fromProviderCanonicalValue(
				context,
				children,
				profile,
				results.At(index).Type(),
				converted,
			)
		} else {
			converted, convertedValue, convertErr = fromProviderValueSelected(
				context,
				children,
				owner,
				ownerBridge,
				profile,
				results.At(index).Type(),
				converted,
			)
		}
		if convertErr != nil {
			return api.ExpressionEmission{}, false, convertErr
		}
		changed = changed || convertedValue
		before = append(before, converted.Before()...)
		elements = append(elements, converted.Value())
		requests = append(requests, converted.Requests()...)
	}
	if !changed {
		return emission, false, nil
	}
	targetType, err := children.RepresentedType(
		context.WithRole(api.RoleResultType),
		nil,
		results,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	target, err := api.NewExpressionEmission(
		before,
		context.Factory().SatisfiesExpression(
			context.Factory().ArrayLiteralExpression(elements, false),
			targetType.Value(),
		),
		api.CombineRequests(requests, targetType.Requests()),
	)
	return target, changed, err
}

func toProviderResults(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	results *types.Tuple,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	return toProviderResultsSelected(
		context,
		children,
		owner,
		ownerBridge,
		nil,
		results,
		emission,
	)
}

func ToProviderProfileResults(
	context api.Context,
	children api.ChildEmitter,
	results *types.Tuple,
	profile []gostdlib.ProviderCallableProfileInterface,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, _, err := toProviderResultsSelected(
		context,
		children,
		nil,
		"",
		profile,
		results,
		emission,
	)
	return target, err
}

func ToProviderProfileResultsForBridge(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	results *types.Tuple,
	profile []gostdlib.ProviderCallableProfileInterface,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, _, err := toProviderResultsSelected(
		context,
		children,
		owner,
		ownerBridge,
		profile,
		results,
		emission,
	)
	return target, err
}

func toProviderResultsSelected(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	results *types.Tuple,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if results == nil || results.Len() == 0 {
		return emission, false, nil
	}
	if results.Len() == 1 {
		return toProviderValueSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			results.At(0).Type(),
			emission,
		)
	}
	temporary, err := context.Names().Temporary(api.TemporaryMultipleResults)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	before := append(emission.Before(), context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(temporary),
					nil,
					nil,
					emission.Value(),
				),
			},
			tsgo.NodeFlagsConst,
		),
	))
	elements := make([]tsgo.Expression, 0, results.Len())
	requests := emission.Requests()
	changed := false
	for index := range results.Len() {
		value := context.Factory().ElementAccessExpression(
			context.Factory().Identifier(temporary),
			nil,
			context.Factory().NumericLiteral(
				strconv.Itoa(index),
				tsgo.TokenFlagsNone,
			),
			tsgo.NodeFlagsNone,
		)
		converted, convertedValue, convertErr := toProviderValueSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			results.At(index).Type(),
			api.DirectExpression(value),
		)
		if convertErr != nil {
			return api.ExpressionEmission{}, false, convertErr
		}
		changed = changed || convertedValue
		before = append(before, converted.Before()...)
		elements = append(elements, converted.Value())
		requests = append(requests, converted.Requests()...)
	}
	if !changed {
		return emission, false, nil
	}
	target, err := api.NewExpressionEmission(
		before,
		context.Factory().ArrayLiteralExpression(elements, false),
		api.CombineRequests(requests),
	)
	return target, changed, err
}

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
