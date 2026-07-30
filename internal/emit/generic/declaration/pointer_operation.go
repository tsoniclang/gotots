package declaration

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitPointerOperationType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	operation *api.GenericOperationContract,
) (api.TypeEmission, bool, error) {
	element, ok := api.GenericPointerOperationElement(
		operation.Selection(),
		operation.Signature(),
	)
	if !ok {
		return api.TypeEmission{}, false, nil
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleParameterType),
		source,
		element,
	)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	pointerSource := types.NewPointer(element)
	pointer, err := pointertype.EmitRepresented(
		context.WithRole(api.RoleParameterType),
		children,
		source,
		pointerSource,
	)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	nonNilPointer, err := pointertype.EmitNonNilRepresented(
		context.WithRole(api.RoleResultType),
		children,
		source,
		pointerSource,
	)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	var parameters []tsgo.ParameterDeclaration
	var result tsgo.TypeNode
	switch operation.Operation() {
	case api.GenericOperationPointerCell:
		parameters = []tsgo.ParameterDeclaration{
			operationParameter(context, "$0", logical.Value()),
		}
		result = nonNilPointer.Value()
	case api.GenericOperationPointerLoad:
		parameters = []tsgo.ParameterDeclaration{
			operationParameter(context, "$0", pointer.Value()),
		}
		result = logical.Value()
	case api.GenericOperationPointerStore:
		parameters = []tsgo.ParameterDeclaration{
			operationParameter(context, "$0", pointer.Value()),
			operationParameter(context, "$1", logical.Value()),
		}
		result = context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		)
	default:
		return api.TypeEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic pointer operation is invalid",
		}
	}
	return api.DirectType(
		context.Factory().FunctionTypeNode(nil, parameters, result),
		api.CombineRequests(
			logical.Requests(),
			pointer.Requests(),
			nonNilPointer.Requests(),
		)...,
	), true, nil
}

func operationParameter(
	context api.Context,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		context.Factory().Identifier(name),
		nil,
		targetType,
		nil,
	)
}
