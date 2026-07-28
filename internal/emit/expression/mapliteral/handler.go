package mapliteral

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	mapType, ok := maprepresentation.Source(context, sourceType)
	if !ok ||
		source.Type == nil ||
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
		key, err := emitOperand(
			context.WithRole(api.RoleMapKey),
			children,
			entry.Key,
			mapType.Key(),
			false,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		key, err = maprepresentation.ProjectKey(
			context.WithRole(api.RoleMapKey),
			entry.Key,
			mapType.Key(),
			key,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		value, err := emitOperand(
			context.WithRole(api.RoleMapValue),
			children,
			entry.Value,
			mapType.Elem(),
			true,
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
		mapType.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(zero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, targetRequests, err := maprepresentation.Make(
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
	return api.NewExpressionEmission(before, target, targetRequests)
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
	if err != nil || !copyValue {
		return value, err
	}
	return context.Values().Copy(context, source, targetType, value)
}
