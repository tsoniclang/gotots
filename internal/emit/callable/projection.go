package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func AdaptProjectedSourceValue(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	provider *types.Func,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	selected, ok := context.ResolveCallableABI(provider)
	if !ok || !hasProjectedParameter(selected) {
		return target, nil
	}
	signature, ok := provider.Type().(*types.Signature)
	if !ok || signature.Recv() != nil {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	adapter, err := EmitAdapter(context, children, source, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments, before, projectionRequests, err := ProjectArguments(
		context,
		source,
		signature,
		adapter.ParameterReferences(context.Factory()),
		selected,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	requests := api.CombineRequests(
		target.Requests(),
		adapter.Requests(),
		projectionRequests,
	)
	if len(before) != 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "callable value projection produced evaluation prerequisites",
		}
	}
	call := context.Factory().CallExpression(
		target.Value(),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		target.Before(),
		context.Factory().ArrowFunction(
			nil,
			nil,
			adapter.Parameters(),
			adapter.Result(),
			context.Factory().EqualsGreaterThanToken(),
			call,
		),
		requests,
	)
}

func ProjectArguments(
	context api.Context,
	source ast.Node,
	signature *types.Signature,
	arguments []tsgo.Expression,
	selected callableabi.Callable,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	if !selected.Valid() {
		return arguments, nil, nil, nil
	}
	if signature == nil || signature.Params().Len() != len(arguments) {
		return nil, nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "callable argument projection cardinality differs",
		}
	}
	projectedArguments := append([]tsgo.Expression(nil), arguments...)
	var before []tsgo.Statement
	var requests []api.RootRequest
	for index := range projectedArguments {
		parameter, ok := selected.Parameter(index)
		if !ok {
			return nil, nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "callable argument projection is absent",
			}
		}
		switch parameter.Projection() {
		case callableabi.ProjectionIdentity:
		case callableabi.ProjectionPointeeValue:
			projected, err := context.PointeeValues().ProjectedPointee(
				context.WithRole(api.RoleCallArgument),
				source,
				signature.Params().At(index).Type(),
				api.DirectExpression(projectedArguments[index]),
				parameter.NilPolicy(),
			)
			if err != nil {
				return nil, nil, nil, err
			}
			before = append(before, projected.Before()...)
			projectedArguments[index] = projected.Value()
			requests = api.CombineRequests(requests, projected.Requests())
		default:
			return nil, nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "callable value uses an unsupported parameter projection",
			}
		}
	}
	return projectedArguments, before, requests, nil
}

func hasProjectedParameter(selected callableabi.Callable) bool {
	for _, parameter := range selected.Parameters() {
		if parameter.Projection() != callableabi.ProjectionIdentity {
			return true
		}
	}
	return false
}
