package maprepresentation

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func RangeKeys(
	context api.Context,
	source ast.Node,
	model Model,
	receiver tsgo.Expression,
) (api.ExpressionEmission, error) {
	if model.source == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "map range has no source model",
		}
	}
	if model.storage == StorageScalar {
		reference, err := context.Names().Runtime(
			api.RuntimeMapKeys,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().CallExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				nil,
				[]tsgo.Expression{receiver},
				tsgo.NodeFlagsNone,
			),
			reference.Requests()...,
		), nil
	}
	if model.storage == StorageSpecialized {
		reference, err := context.Names().MapSpecialization(
			model.sourceType,
			api.MapSpecializationDemandRange,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		name, err := mapruntime.Name(mapruntime.MemberKeys)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			methodCall(context, receiver, name),
			reference.Requests()...,
		), nil
	}
	return api.ExpressionEmission{},
		api.Unsupported(context, api.CategoryExpression, source)
}

func RangeLookupOK(
	context api.Context,
	model Model,
	receiver tsgo.Expression,
	key tsgo.Expression,
) (api.ExpressionEmission, error) {
	if model.source == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "map range lookup has no source model",
		}
	}
	name, err := mapruntime.Name(mapruntime.MemberLookupOK)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		methodCall(context, receiver, name, key),
	), nil
}

func methodCall(
	context api.Context,
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			receiver,
			nil,
			context.Factory().Identifier(name),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}
