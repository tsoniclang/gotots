package callable

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ValueSignature(
	signature *types.Signature,
) (*types.Signature, bool) {
	if signature == nil || signature.TypeParams().Len() != 0 {
		return nil, false
	}
	if signature.Recv() == nil && signature.RecvTypeParams().Len() == 0 {
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
) (api.TypeEmission, error) {
	return emitInlineNonNilType(
		context,
		children,
		source,
		signature,
		func(result tsgo.TypeNode) tsgo.TypeNode { return result },
	)
}

func emitInlineNonNilType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
	resultType func(tsgo.TypeNode) tsgo.TypeNode,
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
	return api.DirectType(
		context.Factory().FunctionTypeNode(
			nil,
			target.Parameters(),
			resultType(target.Result()),
		),
		target.Requests()...,
	), nil
}
