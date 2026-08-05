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
		api.TargetIntrinsicPromise.TypeName(factory),
		[]tsgo.TypeNode{result},
	)
}

func IndirectResult(
	context api.Context,
	result tsgo.TypeNode,
) (api.TypeEmission, error) {
	if context.ConcurrencySemantics() !=
		api.ConcurrencySemanticsCooperative {
		return api.DirectType(result), nil
	}
	return AwaitableResult(context, result)
}

func AwaitableResult(
	context api.Context,
	result tsgo.TypeNode,
) (api.TypeEmission, error) {
	reference, err := context.Names().Runtime(
		api.RuntimeAwaitable,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			[]tsgo.TypeNode{result},
		),
		reference.Requests()...,
	), nil
}

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
	cooperative bool,
) (api.TypeEmission, error) {
	return emitInlineNonNilType(
		context,
		children,
		source,
		signature,
		func(result tsgo.TypeNode) tsgo.TypeNode {
			if cooperative {
				return PromiseResult(context.Factory(), result)
			}
			return result
		},
	)
}

func EmitInlineAwaitableType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
	awaitable bool,
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
	result := api.DirectType(target.Result())
	if awaitable {
		result, err = IndirectResult(context, target.Result())
		if err != nil {
			return api.TypeEmission{}, err
		}
	}
	return api.DirectType(
		context.Factory().FunctionTypeNode(
			nil,
			target.Parameters(),
			result.Value(),
		),
		api.CombineRequests(target.Requests(), result.Requests())...,
	), nil
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
