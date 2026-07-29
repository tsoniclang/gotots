package genericfunctionvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, bool, error) {
	switch source.(type) {
	case *ast.IndexExpr, *ast.IndexListExpr:
	default:
		return api.ExpressionEmission{}, false, nil
	}
	owner, instance, ok := genericinstance.FunctionEvidence(
		context.TypesInfo(),
		source,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	signature, ok := instance.Type.(*types.Signature)
	if !ok ||
		signature.Recv() != nil ||
		!callable.Supports(signature) ||
		!types.Identical(context.TypesInfo().TypeOf(source), signature) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operationSet, resolved, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !resolved ||
		instance.TypeArgs == nil ||
		instance.TypeArgs.Len() != len(operationSet.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := callable.EmitAdapter(context, children, source, signature)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	typeArguments, typeRequests, err := genericinstance.EmitTypeArguments(
		context,
		children,
		source,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	capabilities, capabilityRequests, err := genericinstance.EmitCapabilities(
		context,
		source,
		operationSet,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	capabilityArguments, err := genericabi.JoinCapabilities(
		owner,
		operationSet.Operations(),
		capabilities,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reference, err := context.Names().Reference(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments := append(
		capabilityArguments,
		target.ParameterReferences(context.Factory())...,
	)
	return api.DirectExpression(
		context.Factory().ArrowFunction(
			nil,
			nil,
			target.Parameters(),
			target.Result(),
			context.Factory().EqualsGreaterThanToken(),
			context.Factory().CallExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				typeArguments,
				arguments,
				tsgo.NodeFlagsNone,
			),
		),
		api.CombineRequests(
			target.Requests(),
			typeRequests,
			capabilityRequests,
			reference.Requests(),
		)...,
	), true, nil
}
