package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) Hash(
	context api.Context,
	source ast.Node,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	stored, err := a.storage(
		context.WithRole(api.RoleDefinedValue),
		api.DirectExpression(value),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arrayName, err := context.Names().Temporary(api.TemporaryArrayHash)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if a.Length() == 0 {
		return api.NewExpressionEmission(
			append(
				stored.Before(),
				arrayComparisonVariable(
					context,
					tsgo.NodeFlagsConst,
					arrayName,
					stored.Value(),
				),
			),
			context.Factory().NumericLiteral(
				"2166136261",
				tsgo.TokenFlagsNone,
			),
			stored.Requests(),
		)
	}
	indexName, err := context.Names().Temporary(api.TemporaryArrayHash)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	hashName, err := context.Names().Temporary(api.TemporaryArrayHash)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	array := context.Factory().Identifier(arrayName)
	index := context.Factory().Identifier(indexName)
	hash := context.Factory().Identifier(hashName)
	element, err := a.loadElement(
		context,
		source,
		callMember(context, array, arraymember.Get, index),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementHash, err := context.Values().Hash(
		context.WithRole(api.RoleArrayElement),
		source,
		a.ElementType(),
		element.Value(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeMapHash,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	mixed := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			context.Factory().Identifier(mapruntime.HashMixMember),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{hash, elementHash.Value()},
		tsgo.NodeFlagsNone,
	)
	body := append(
		element.Before(),
		elementHash.Before()...,
	)
	body = append(
		body,
		context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				hash,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				mixed,
			),
		),
	)
	before := append(stored.Before(),
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsConst,
			arrayName,
			stored.Value(),
		),
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsLet,
			hashName,
			context.Factory().NumericLiteral(
				"2166136261",
				tsgo.TokenFlagsNone,
			),
		),
		context.Factory().ForStatement(
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						index,
						nil,
						nil,
						context.Factory().NumericLiteral(
							"0",
							tsgo.TokenFlagsNone,
						),
					),
				},
				tsgo.NodeFlagsLet,
			),
			context.Factory().BinaryExpression(
				nil,
				index,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorLessThanToken,
				),
				a.lengthLiteral(context),
			),
			context.Factory().PostfixUnaryExpression(
				index,
				tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
			),
			context.Factory().Block(body, true),
		),
	)
	return api.NewExpressionEmission(
		before,
		hash,
		api.CombineRequests(
			stored.Requests(),
			elementHash.Requests(),
			element.Requests(),
			runtime.Requests(),
		),
	)
}
