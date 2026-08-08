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
	return target, nil
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
	for index := range arguments {
		parameter, ok := selected.Parameter(index)
		if !ok {
			return nil, nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "callable argument projection is absent",
			}
		}
		if parameter.Projection() != callableabi.ProjectionIdentity {
			return nil, nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "callable parameter is not identity-preserving",
			}
		}
	}
	return arguments, nil, nil, nil
}
