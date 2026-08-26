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
	selection CallableProfileSelection,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, _, err := fromProviderResultsSelected(
		context,
		children,
		nil,
		"",
		selection.canonicalResults,
		selection.interfaces,
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
	providerContext, err := providerRepresentationContext(context, profile)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	targetType, err := children.RepresentedType(
		providerContext.WithRole(api.RoleResultType),
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

func providerRepresentationContext(
	context api.Context,
	profile []gostdlib.ProviderCallableProfileInterface,
) (api.Context, error) {
	selected, err := context.WithProviderScalarRepresentation()
	if err != nil {
		return api.Context{}, err
	}
	if len(profile) == 0 {
		return selected, nil
	}
	return selected.WithProviderProfile(profile)
}
