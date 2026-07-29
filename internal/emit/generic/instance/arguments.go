package instance

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitTypeArguments(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	arguments *types.TypeList,
) ([]tsgo.TypeNode, []api.RootRequest, error) {
	if arguments == nil {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic type arguments are absent",
		}
	}
	targets := make([]tsgo.TypeNode, 0, arguments.Len())
	var requests []api.RootRequest
	for index := range arguments.Len() {
		target, err := children.RepresentedType(
			context.WithRole(api.RoleCallArgumentType),
			source,
			arguments.At(index),
		)
		if err != nil {
			return nil, nil, err
		}
		targets = append(targets, target.Value())
		requests = append(requests, target.Requests()...)
	}
	return targets, requests, nil
}

func EmitCapabilities(
	context api.Context,
	source ast.Node,
	operationSet api.GenericOperationSet,
	arguments *types.TypeList,
) ([]tsgo.Expression, []api.RootRequest, error) {
	targets := make(
		[]tsgo.Expression,
		0,
		len(operationSet.Operations()),
	)
	var requests []api.RootRequest
	for _, operation := range operationSet.Operations() {
		signature, err := InstantiateOperation(
			operationSet,
			operation.Signature(),
			arguments,
		)
		if err != nil {
			return nil, nil, err
		}
		var reference api.NameReference
		if api.ContainsGenericTypeParameter(signature) {
			reference, err = context.ProjectGenericOperation(
				source,
				operation,
				signature,
			)
		} else {
			reference, err = context.Names().GenericCapability(
				operation.Selection(),
				signature,
			)
		}
		if err != nil {
			return nil, nil, err
		}
		targets = append(
			targets,
			context.Factory().Identifier(reference.Name()),
		)
		requests = append(requests, reference.Requests()...)
	}
	return targets, requests, nil
}
