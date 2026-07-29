package address

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type storageProjection struct {
	storage     api.TypeEmission
	toStorage   tsgo.ArrowFunction
	fromStorage tsgo.ArrowFunction
	requests    []api.RootRequest
}

func buildStorageProjection(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	logical api.TypeEmission,
) (storageProjection, error) {
	storage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
	)
	if err != nil {
		return storageProjection{}, err
	}
	toStorage, toRequests, err := storageConverter(
		context,
		source,
		sourceType,
		logical.Value(),
		storage.Value(),
		context.Values().ToStorage,
	)
	if err != nil {
		return storageProjection{}, err
	}
	fromStorage, fromRequests, err := storageConverter(
		context,
		source,
		sourceType,
		storage.Value(),
		logical.Value(),
		context.Values().FromStorage,
	)
	if err != nil {
		return storageProjection{}, err
	}
	return storageProjection{
		storage:     storage,
		toStorage:   toStorage,
		fromStorage: fromStorage,
		requests: api.CombineRequests(
			logical.Requests(),
			storage.Requests(),
			toRequests,
			fromRequests,
		),
	}, nil
}

type storageConversion func(
	api.Context,
	ast.Node,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, error)

func storageConverter(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	inputType tsgo.TypeNode,
	outputType tsgo.TypeNode,
	convert storageConversion,
) (tsgo.ArrowFunction, []api.RootRequest, error) {
	const parameterName = "$value"
	converted, err := convert(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
		api.DirectExpression(context.Factory().Identifier(parameterName)),
	)
	if err != nil {
		return nil, nil, err
	}
	body := append(
		converted.Before(),
		context.Factory().ReturnStatement(converted.Value()),
	)
	return context.Factory().ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			context.Factory().ParameterDeclaration(
				nil,
				nil,
				context.Factory().Identifier(parameterName),
				nil,
				inputType,
				nil,
			),
		},
		outputType,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(body, true),
	), converted.Requests(), nil
}
