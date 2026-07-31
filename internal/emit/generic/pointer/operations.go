package pointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
)

func Cell(
	context api.Context,
	source ast.Node,
	element types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	parameter, generic := api.GenericTypeParameter(element)
	if !generic {
		return api.ExpressionEmission{}, false, nil
	}
	result, err := genericoperation.Call(
		context,
		source,
		api.GenericOperationPointerCell,
		[]types.Type{parameter},
		[]types.Type{types.NewPointer(parameter)},
		[]api.ExpressionEmission{value},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	return result, true, nil
}

func Load(
	context api.Context,
	source ast.Node,
	element types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	parameter, generic := api.GenericTypeParameter(element)
	if !generic {
		return api.ExpressionEmission{}, false, nil
	}
	result, err := genericoperation.Call(
		context,
		source,
		api.GenericOperationPointerLoad,
		[]types.Type{types.NewPointer(parameter)},
		[]types.Type{parameter},
		[]api.ExpressionEmission{pointer},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	return result, true, nil
}

func StoreTarget(
	context api.Context,
	source ast.Node,
	element types.Type,
	pointer api.ExpressionEmission,
) (api.StoreTargetEmission, bool, error) {
	parameter, generic := api.GenericTypeParameter(element)
	if !generic {
		return api.StoreTargetEmission{}, false, nil
	}
	pointerType := types.NewPointer(parameter)
	getter, err := genericoperation.Reference(
		context,
		source,
		api.GenericOperationPointerLoad,
		[]types.Type{pointerType},
		[]types.Type{parameter},
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	setter, err := genericoperation.Reference(
		context,
		source,
		api.GenericOperationPointerStore,
		[]types.Type{pointerType, parameter},
		nil,
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	target, err := api.NewFunctionStoreTargetEmission(
		api.DirectExpression(
			context.Factory().Identifier(getter.Name()),
			getter.Requests()...,
		),
		api.DirectExpression(
			context.Factory().Identifier(setter.Name()),
			setter.Requests()...,
		),
		[]api.ExpressionEmission{pointer},
		parameter,
	)
	return target, true, err
}
