package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func deferredReturnControl(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	results *types.Tuple,
	named bool,
	label string,
) (
	api.ReturnControl,
	[]tsgo.Statement,
	[]api.RootRequest,
	string,
	error,
) {
	count := 0
	if results != nil {
		count = results.Len()
	}
	if count == 0 {
		control, err := api.NewReturnControl(label, "")
		return control, nil, nil, "", err
	}
	if named {
		targets, err := namedReturnTargets(context, results)
		if err != nil {
			return api.ReturnControl{}, nil, nil, "", err
		}
		control, err := api.NewNamedReturnControl(label, targets)
		return control, nil, nil, "", err
	}
	name, err := context.Names().Temporary(api.TemporaryReturnResult)
	if err != nil {
		return api.ReturnControl{}, nil, nil, "", err
	}
	targetType, typeRequests, err := EmitResultType(
		context.WithRole(api.RoleResultType),
		children,
		source,
		results,
	)
	if err != nil {
		return api.ReturnControl{}, nil, nil, "", err
	}
	zero, err := ZeroResult(context, source, results)
	if err != nil {
		return api.ReturnControl{}, nil, nil, "", err
	}
	statements := zero.Before()
	statements = append(
		statements,
		variableStatement(
			context,
			tsgo.NodeFlagsLet,
			name,
			targetType,
			zero.Value(),
		),
	)
	control, err := api.NewReturnControl(label, name)
	return control,
		statements,
		api.CombineRequests(typeRequests, zero.Requests()),
		name,
		err
}

func namedReturnTargets(
	context api.Context,
	results *types.Tuple,
) ([]api.StoreTargetEmission, error) {
	targets := make([]api.StoreTargetEmission, 0, results.Len())
	for index := range results.Len() {
		result := results.At(index)
		if result.Name() == "" {
			return nil, &api.InvariantError{
				Role:   api.RoleReturnResult,
				Reason: "named return tuple contains an unnamed result",
			}
		}
		targetName, err := context.Names().Result(result, index)
		if err != nil {
			return nil, err
		}
		target, err := api.NewStoreTargetEmission(
			context.Factory().Identifier(targetName),
			result.Type(),
			nil,
		)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func ZeroResult(
	context api.Context,
	source ast.Node,
	results *types.Tuple,
) (api.ExpressionEmission, error) {
	values := make([]tsgo.Expression, 0, results.Len())
	var before []tsgo.Statement
	var requests []api.RootRequest
	for index := range results.Len() {
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleReturnResult),
			source,
			results.At(index).Type(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, zero.Before()...)
		values = append(values, zero.Value())
		requests = append(requests, zero.Requests()...)
	}
	value := values[0]
	if len(values) > 1 {
		value = context.Factory().ArrayLiteralExpression(values, false)
	}
	return api.NewExpressionEmission(before, value, requests)
}

func deferredFinalReturn(
	context api.Context,
	results *types.Tuple,
	named bool,
	resultName string,
) ([]tsgo.Statement, []api.RootRequest, error) {
	count := 0
	if results != nil {
		count = results.Len()
	}
	if count == 0 {
		return nil, nil, nil
	}
	if !named {
		if resultName == "" {
			return nil, nil, &api.InvariantError{
				Role:   api.RoleReturnResult,
				Reason: "deferred callable result storage is absent",
			}
		}
		return []tsgo.Statement{
			context.Factory().ReturnStatement(
				context.Factory().Identifier(resultName),
			),
		}, nil, nil
	}
	values := make([]tsgo.Expression, 0, count)
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for index := range count {
		result := results.At(index)
		targetName, err := context.Names().Result(result, index)
		if err != nil {
			return nil, nil, err
		}
		value := api.DirectExpression(
			context.Factory().Identifier(targetName),
		)
		value, err = context.Values().Transfer(
			context.WithRole(api.RoleReturnResult),
			nil,
			result.Type(),
			result.Type(),
			api.ValueTransferCopy,
			value,
		)
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, value.Before()...)
		values = append(values, value.Value())
		requests = append(requests, value.Requests()...)
	}
	value := values[0]
	if len(values) > 1 {
		value = context.Factory().ArrayLiteralExpression(values, false)
	}
	statements = append(
		statements,
		context.Factory().ReturnStatement(value),
	)
	return statements, requests, nil
}
