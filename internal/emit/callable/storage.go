package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func addressableParameterPrologue(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if signature == nil {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "addressable parameter signature is nil",
		}
	}
	var statements []tsgo.Statement
	var requests []api.RootRequest
	variables := make([]*types.Var, 0, signature.Params().Len()+1)
	names := make([]string, 0, signature.Params().Len()+1)
	if receiver := signature.Recv(); receiver != nil {
		name, err := context.Names().Parameter(
			receiver,
			signature.Params().Len(),
		)
		if err != nil {
			return nil, nil, err
		}
		variables = append(variables, receiver)
		names = append(names, name)
	}
	for index := range signature.Params().Len() {
		parameter := signature.Params().At(index)
		name, err := context.Names().Parameter(parameter, index)
		if err != nil {
			return nil, nil, err
		}
		variables = append(variables, parameter)
		names = append(names, name)
	}
	for index, variable := range variables {
		storageName, selected := context.AddressableStorage().Name(
			context,
			variable,
		)
		if !selected {
			continue
		}
		initial := api.DirectExpression(
			context.Factory().Identifier(names[index]),
		)
		if receiver, ok := context.ValueReceiver(variable); ok {
			initial = api.DirectExpression(receiver.OriginalValue())
			if receiver.CopySelected() {
				copied, err := context.Values().Copy(
					context.WithRole(api.RoleReceiverValue),
					source,
					variable.Type(),
					initial,
				)
				if err != nil {
					return nil, nil, err
				}
				initial = copied
			}
		}
		cell, err := context.AddressableStorage().Cell(
			context,
			children,
			source,
			variable.Type(),
			initial,
		)
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, cell.Before()...)
		statements = append(
			statements,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(storageName),
							nil,
							nil,
							cell.Value(),
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		)
		requests = append(requests, cell.Requests()...)
	}
	return statements, requests, nil
}
