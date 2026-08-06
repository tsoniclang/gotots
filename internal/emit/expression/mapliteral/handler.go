package mapliteral

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	mapType, ok := maprepresentation.Source(context, sourceType)
	if !ok ||
		source.Incomplete ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(sourceType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operands := make([]api.ExpressionEmission, 0, len(source.Elts)*2)
	for _, element := range source.Elts {
		entry, ok := element.(*ast.KeyValueExpr)
		if !ok {
			return api.ExpressionEmission{},
				api.Unsupported(
					context.WithRole(api.RoleCompositeElement),
					api.CategoryExpression,
					element,
				)
		}
		key, err := emitMapKey(
			context.WithRole(api.RoleMapKey),
			children,
			entry.Key,
			mapType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		value, err := emitOperand(
			context.WithRole(api.RoleMapValue),
			children,
			entry.Value,
			mapType.Element(),
			false,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		operands = append(operands, key, value)
	}
	values, before, requests, err := maprepresentation.ArrangeOperands(
		context,
		operands,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	entries := make([]tsgo.Expression, 0, len(source.Elts))
	for index := 0; index < len(values); index += 2 {
		entries = append(
			entries,
			context.Factory().ArrayLiteralExpression(
				[]tsgo.Expression{values[index], values[index+1]},
				false,
			),
		)
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleMapValue),
		source,
		mapType.Element(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(zero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if api.ContainsGenericTypeParameter(sourceType) {
		parameterTypes := []types.Type{mapType.Element()}
		arguments := []api.ExpressionEmission{
			api.DirectExpression(
				zero.Value(),
				api.CombineRequests(requests, zero.Requests())...,
			),
		}
		for index := range source.Elts {
			parameterTypes = append(
				parameterTypes,
				mapType.Key(),
				mapType.Element(),
			)
			arguments = append(
				arguments,
				api.DirectExpression(values[index*2]),
				api.DirectExpression(values[index*2+1]),
			)
		}
		target, err := genericoperation.Call(
			context,
			source,
			api.GenericOperationMapConstruct,
			parameterTypes,
			[]types.Type{sourceType},
			arguments,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			append(before, zero.Before()...),
			target.Value(),
			target.Requests(),
		)
	}
	target, err := maprepresentation.Make(
		context,
		source,
		sourceType,
		zero.Value(),
		context.Factory().NumericLiteral(
			strconv.Itoa(len(entries)),
			tsgo.TokenFlagsNone,
		),
		entries,
		requests,
		zero.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		append(before, target.Before()...),
		target.Value(),
		target.Requests(),
	)
}

func emitMapKey(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	mapType maprepresentation.Model,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil || !types.AssignableTo(sourceType, mapType.Key()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.WithExpectedType(mapType.Key()),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return mapType.TransferKey(context, source, value)
}

func emitOperand(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	targetType types.Type,
	copyValue bool,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil || !types.AssignableTo(sourceType, targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.WithExpectedType(targetType),
		source,
	)
	if err != nil {
		return value, err
	}
	mode := api.ValueTransferRepresentation
	if copyValue {
		mode = api.ValueTransferCopy
	}
	return context.Values().Transfer(
		context,
		source,
		sourceType,
		targetType,
		mode,
		value,
	)
}
