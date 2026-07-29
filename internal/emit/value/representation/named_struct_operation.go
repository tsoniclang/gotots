package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) namedStructOperation(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	operation api.NamedStructOperation,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "named-struct operation source type is invalid",
		}
	}
	typeName := named.Origin().Obj()
	reference, err := context.Names().NamedStructOperation(
		typeName,
		operation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var typeArguments []tsgo.TypeNode
	var typeRequests []api.RootRequest
	var capabilities []tsgo.Expression
	var capabilityRequests []api.RootRequest
	if named.TypeParams().Len() != 0 {
		if named.TypeArgs().Len() != named.TypeParams().Len() {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic named-struct operation is not instantiated",
			}
		}
		operationSet, resolved, resolveErr :=
			context.ResolveGenericNamedStructOperation(typeName, operation)
		if resolveErr != nil {
			return api.ExpressionEmission{}, resolveErr
		}
		if !resolved {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic named-struct operation has no declaration",
			}
		}
		typeArguments, typeRequests, err =
			genericinstance.EmitTypeArguments(
				context,
				owner.children,
				source,
				named.TypeArgs(),
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		capabilities, capabilityRequests, err =
			genericinstance.EmitCapabilities(
				context,
				source,
				operationSet,
				named.TypeArgs(),
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	memberName, err := api.NamedStructOperationMemberName(operation)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments = append(capabilities, arguments...)
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				context.Factory().Identifier(memberName),
				tsgo.NodeFlagsNone,
			),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			reference.Requests(),
			typeRequests,
			capabilityRequests,
		)...,
	), nil
}
