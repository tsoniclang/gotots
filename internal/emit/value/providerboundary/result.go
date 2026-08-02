package providerboundary

import (
	"fmt"
	"go/types"
	"slices"
	"strconv"

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

func ToProviderArguments(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	return toProviderArgumentsSelected(
		context,
		children,
		parameters,
		nil,
		sourceArguments,
	)
}

func ToProviderProfileArguments(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	canonical []int,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	return toProviderArgumentsSelected(
		context,
		children,
		parameters,
		canonical,
		sourceArguments,
	)
}

func toProviderArgumentsSelected(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	canonical []int,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	if parameters == nil && len(sourceArguments) == 0 {
		return nil, nil, nil, nil
	}
	if parameters == nil || parameters.Len() != len(sourceArguments) {
		parameterCount := 0
		if parameters != nil {
			parameterCount = parameters.Len()
		}
		return nil, nil, nil, &api.InvariantError{
			Role: context.Role(),
			Reason: fmt.Sprintf(
				"provider argument count does not match the source contract: %d parameters, %d arguments",
				parameterCount,
				len(sourceArguments),
			),
		}
	}
	arguments := make([]tsgo.Expression, 0, len(sourceArguments))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for index, sourceArgument := range sourceArguments {
		converted := api.DirectExpression(sourceArgument)
		var err error
		if !slices.Contains(canonical, index) {
			converted, _, err = ToProviderValue(
				context,
				children,
				nil,
				"",
				parameters.At(index).Type(),
				converted,
			)
		}
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(before, converted.Before()...)
		arguments = append(arguments, converted.Value())
		requests = append(requests, converted.Requests()...)
	}
	return arguments, before, api.CombineRequests(requests), nil
}

func FromProviderProfileResults(
	context api.Context,
	children api.ChildEmitter,
	results *types.Tuple,
	canonical []int,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, _, err := fromProviderResultsSelected(
		context,
		children,
		nil,
		"",
		results,
		canonical,
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
		results,
		nil,
		emission,
	)
}

func fromProviderResultsSelected(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	results *types.Tuple,
	canonical []int,
	emission api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if results == nil || results.Len() == 0 {
		return emission, false, nil
	}
	if results.Len() == 1 {
		if slices.Contains(canonical, 0) {
			return emission, false, nil
		}
		return FromProviderValue(
			context,
			children,
			owner,
			ownerBridge,
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
		if !slices.Contains(canonical, index) {
			converted, convertedValue, convertErr = FromProviderValue(
				context,
				children,
				owner,
				ownerBridge,
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
	if results == nil || results.Len() == 0 {
		return emission, false, nil
	}
	if results.Len() == 1 {
		return ToProviderValue(
			context,
			children,
			owner,
			ownerBridge,
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
		converted, convertedValue, convertErr := ToProviderValue(
			context,
			children,
			owner,
			ownerBridge,
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
	selected, ok := types.Unalias(sourceType).(*types.Named)
	if ok && selected.Obj() != nil {
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
		return fromProviderCallable(
			context,
			children,
			owner,
			ownerBridge,
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
	selected, ok := types.Unalias(sourceType).(*types.Named)
	if ok && selected.Obj() != nil {
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
		return toProviderCallable(
			context,
			children,
			owner,
			ownerBridge,
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
