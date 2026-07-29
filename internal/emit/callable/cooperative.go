package callable

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func PromiseResult(
	factory tsgo.Factory,
	result tsgo.TypeNode,
) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier("Promise"),
		[]tsgo.TypeNode{result},
	)
}

func ValueSignature(
	signature *types.Signature,
) (*types.Signature, bool) {
	if !Supports(signature) {
		return nil, false
	}
	if signature.Recv() == nil {
		return signature, true
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		signature.Params(),
		signature.Results(),
		signature.Variadic(),
	), true
}

func EmitInlineNonNilType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
	cooperative bool,
) (api.TypeEmission, error) {
	target, err := emitRepresented(
		context,
		children,
		source,
		signature,
		api.RoleCallableParameter,
		api.RoleCallableResult,
		func(_ *types.Var, index int) (string, error) {
			return "$" + strconv.Itoa(index), nil
		},
		false,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	target, err = withRecoveryAuthority(context, target)
	if err != nil {
		return api.TypeEmission{}, err
	}
	result := target.Result()
	if cooperative {
		result = PromiseResult(context.Factory(), result)
	}
	return api.DirectType(
		context.Factory().FunctionTypeNode(
			nil,
			target.Parameters(),
			result,
		),
		target.Requests()...,
	), nil
}
