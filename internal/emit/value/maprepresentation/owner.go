package maprepresentation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Source(
	context api.Context,
	sourceType types.Type,
) (*types.Map, bool) {
	mapType, ok := types.Unalias(sourceType).(*types.Map)
	if !ok {
		return nil, false
	}
	if _, keyOK := basictype.PrimitiveAlias(
		context.TypesSizes(),
		mapType.Key(),
	); !keyOK || !types.Comparable(mapType.Key()) {
		return nil, false
	}
	if _, valueOK := basictype.PrimitiveAlias(
		context.TypesSizes(),
		mapType.Elem(),
	); !valueOK {
		return nil, false
	}
	return mapType, true
}

func EmitType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	reference, typeArguments, err := Reference(
		context,
		source,
		sourceType,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			typeArguments,
		),
		reference.Requests()...,
	), nil
}

func Reference(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	phase api.ImportPhase,
) (api.NameReference, []tsgo.TypeNode, error) {
	mapType, ok := Source(context, sourceType)
	if !ok {
		return api.NameReference{}, nil,
			api.Unsupported(context, api.CategoryType, source)
	}
	key, keyRequests, err := scalarType(context, source, mapType.Key())
	if err != nil {
		return api.NameReference{}, nil, err
	}
	value, valueRequests, err := scalarType(context, source, mapType.Elem())
	if err != nil {
		return api.NameReference{}, nil, err
	}
	reference, err := context.Names().Runtime(api.RuntimeMap, phase)
	if err != nil {
		return api.NameReference{}, nil, err
	}
	requests := api.CombineRequests(
		reference.Requests(),
		keyRequests,
		valueRequests,
	)
	reference, err = api.NewNameReference(reference.Name(), requests...)
	if err != nil {
		return api.NameReference{}, nil, err
	}
	return reference, []tsgo.TypeNode{key, value}, nil
}

func Make(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	zero tsgo.Expression,
	size tsgo.Expression,
	entries []tsgo.Expression,
	requests ...[]api.PlacementRequest,
) (tsgo.Expression, []api.PlacementRequest, error) {
	reference, typeArguments, err := Reference(
		context,
		source,
		sourceType,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	makeName, err := mapruntime.Name(mapruntime.MemberMake)
	if err != nil {
		return nil, nil, err
	}
	allRequests := append(
		[][]api.PlacementRequest{reference.Requests()},
		requests...,
	)
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			context.Factory().Identifier(makeName),
			tsgo.NodeFlagsNone,
		),
		nil,
		typeArguments,
		[]tsgo.Expression{
			zero,
			size,
			context.Factory().ArrayLiteralExpression(entries, false),
		},
		tsgo.NodeFlagsNone,
	), api.CombineRequests(allRequests...), nil
}

func scalarType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (tsgo.TypeNode, []api.PlacementRequest, error) {
	alias, ok := basictype.PrimitiveAlias(context.TypesSizes(), sourceType)
	if !ok {
		return nil, nil,
			api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().Primitive(alias)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().TypeReferenceNode(
		context.Factory().Identifier(reference.Name()),
		nil,
	), reference.Requests(), nil
}
