package maprepresentation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
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
	if _, keyOK := directKey(
		context,
		mapType.Key(),
	); !keyOK || !types.Comparable(mapType.Key()) {
		return nil, false
	}
	if !representedBasic(context, mapType.Elem()) {
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
	keyBasic, _ := directKey(context, mapType.Key())
	key, keyRequests, err := primitiveType(context, source, keyBasic)
	if err != nil {
		return api.NameReference{}, nil, err
	}
	value, valueRequests, err := representedType(
		context,
		source,
		mapType.Elem(),
	)
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
	requests ...[]api.RootRequest,
) (tsgo.Expression, []api.RootRequest, error) {
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
		[][]api.RootRequest{reference.Requests()},
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

func ProjectKey(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, ok := directKey(context, sourceType); !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if model, ok := definedtype.Resolve(sourceType); ok {
		if expression, ok := source.(ast.Expr); ok {
			facts, found := context.TypesInfo().Types[expression]
			if found && facts.Value != nil {
				return constantvalue.EmitValue(
					context,
					expression,
					model.Underlying(),
					facts.Value,
				)
			}
		}
		return api.NewExpressionEmission(
			value.Before(),
			model.Unwrap(context.Factory(), value.Value()),
			value.Requests(),
		)
	}
	return value, nil
}

func directKey(
	context api.Context,
	sourceType types.Type,
) (*types.Basic, bool) {
	var basic *types.Basic
	if model, ok := definedtype.Resolve(sourceType); ok {
		basic = model.Underlying()
	} else {
		basic, _ = types.Unalias(sourceType).(*types.Basic)
	}
	if basic == nil ||
		basic.Info()&types.IsUntyped != 0 ||
		basic.Info()&(types.IsBoolean|types.IsInteger|types.IsString) == 0 {
		return nil, false
	}
	_, represented := basictype.PrimitiveAlias(context.TypesSizes(), basic)
	return basic, represented
}

func representedBasic(
	context api.Context,
	sourceType types.Type,
) bool {
	if _, ok := definedtype.Resolve(sourceType); ok {
		return true
	}
	_, ok := basictype.PrimitiveAlias(context.TypesSizes(), sourceType)
	return ok
}

func representedType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (tsgo.TypeNode, []api.RootRequest, error) {
	if model, ok := definedtype.Resolve(sourceType); ok {
		reference, err := context.Names().TypeReference(model.TypeName())
		if err != nil {
			return nil, nil, err
		}
		return context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			nil,
		), reference.Requests(), nil
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return nil, nil,
			api.Unsupported(context, api.CategoryType, source)
	}
	return primitiveType(context, source, basic)
}

func primitiveType(
	context api.Context,
	source ast.Node,
	basic *types.Basic,
) (tsgo.TypeNode, []api.RootRequest, error) {
	alias, ok := basictype.PrimitiveAlias(context.TypesSizes(), basic)
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
