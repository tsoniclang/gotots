package providerboundary

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
		nil,
		"",
		nil,
		sourceArguments,
	)
}

func ToProviderProfileArguments(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	selection CallableProfileSelection,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	return toProviderArgumentsSelected(
		context,
		children,
		parameters,
		selection.canonicalParameters,
		nil,
		"",
		selection.interfaces,
		sourceArguments,
	)
}

func ToProviderProfileArgumentsForBridge(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	return toProviderArgumentsSelected(
		context,
		children,
		parameters,
		nil,
		owner,
		ownerBridge,
		profile,
		sourceArguments,
	)
}

func FromProviderProfileArguments(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	return fromProviderProfileArgumentsSelected(
		context,
		children,
		parameters,
		nil,
		"",
		profile,
		sourceArguments,
	)
}

func FromProviderProfileArgumentsForBridge(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	return fromProviderProfileArgumentsSelected(
		context,
		children,
		parameters,
		owner,
		ownerBridge,
		profile,
		sourceArguments,
	)
}

func fromProviderProfileArgumentsSelected(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceArguments []tsgo.Expression,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	if parameters == nil && len(sourceArguments) == 0 {
		return nil, nil, nil, nil
	}
	if parameters == nil || parameters.Len() != len(sourceArguments) {
		return nil, nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "provider profile argument count does not match the source contract",
		}
	}
	arguments := make([]tsgo.Expression, 0, len(sourceArguments))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for index, sourceArgument := range sourceArguments {
		converted, _, err := fromProviderValueSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			parameters.At(index).Type(),
			api.DirectExpression(sourceArgument),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(before, converted.Before()...)
		arguments = append(arguments, converted.Value())
		requests = append(requests, converted.Requests()...)
	}
	return arguments, before, api.CombineRequests(requests), nil
}

func toProviderArgumentsSelected(
	context api.Context,
	children api.ChildEmitter,
	parameters *types.Tuple,
	canonical []int,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
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
		if slices.Contains(canonical, index) {
			converted, _, err = toProviderCanonicalValue(
				context,
				children,
				profile,
				parameters.At(index).Type(),
				converted,
			)
		} else {
			converted, _, err = toProviderValueSelected(
				context,
				children,
				owner,
				ownerBridge,
				profile,
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
