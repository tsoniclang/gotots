package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) Zero(
	context api.Context,
	source ast.Node,
) (api.ExpressionEmission, error) {
	elementZero, err := context.Values().Zero(
		context,
		source,
		a.ElementType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(elementZero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if a.aggregate {
		zeroFactory := valueFactory(context, nil, elementZero.Value())
		target, runtimeRequests, err := a.runtimeOperation(
			context,
			api.RuntimeArrayZeroWith,
			a.lengthLiteral(context),
			zeroFactory,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return a.wrap(context, api.DirectExpression(
			target,
			api.CombineRequests(
				elementZero.Requests(),
				runtimeRequests,
			)...,
		))
	}
	typeArguments, typeRequests, err := a.targetTypeArguments(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, requests, err := a.callStatic(
		context,
		arraymember.Zero,
		typeArguments,
		a.lengthLiteral(context),
		elementZero.Value(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result := api.DirectExpression(
		target,
		api.CombineRequests(
			elementZero.Requests(),
			typeRequests,
			requests,
		)...,
	)
	return a.wrap(context, result)
}

func (a RuntimeArray) Copy(
	context api.Context,
	fresh bool,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if fresh {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	if a.aggregate {
		elementZero, err := context.Values().Zero(
			context.WithRole(api.RoleArrayElement),
			nil,
			a.ElementType(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(elementZero.Before()) != 0 {
			return api.ExpressionEmission{}, api.Unsupported(
				context.WithRole(api.RoleArrayElement),
				api.CategoryExpression,
				nil,
			)
		}
		element := context.Factory().Identifier("$value")
		elementCopy, err := context.Values().Copy(
			context.WithRole(api.RoleArrayElement),
			nil,
			a.ElementType(),
			api.DirectExpression(element),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(elementCopy.Before()) != 0 {
			return api.ExpressionEmission{}, api.Unsupported(
				context.WithRole(api.RoleArrayElement),
				api.CategoryExpression,
				nil,
			)
		}
		target, runtimeRequests, err := a.runtimeOperation(
			context,
			api.RuntimeArrayCopyWith,
			a.storage(context, value.Value()),
			valueFactory(context, nil, elementZero.Value()),
			valueFactory(context, []tsgo.ParameterDeclaration{
				context.Factory().ParameterDeclaration(
					nil,
					nil,
					element,
					nil,
					nil,
					nil,
				),
			}, elementCopy.Value()),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		copied, err := api.NewExpressionEmission(
			value.Before(),
			target,
			api.CombineRequests(
				value.Requests(),
				elementZero.Requests(),
				elementCopy.Requests(),
				runtimeRequests,
			),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return a.wrap(context, copied)
	}
	copied, err := api.NewExpressionEmission(
		value.Before(),
		callMember(
			context,
			a.storage(context, value.Value()),
			arraymember.Copy,
		),
		value.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return a.wrap(context, copied)
}

func valueFactory(
	context api.Context,
	parameters []tsgo.ParameterDeclaration,
	value tsgo.Expression,
) tsgo.ArrowFunction {
	return context.Factory().ArrowFunction(
		nil,
		nil,
		parameters,
		nil,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ReturnStatement(value)},
			true,
		),
	)
}

func (a RuntimeArray) Equal(
	context api.Context,
	source ast.Node,
	left tsgo.Expression,
	right tsgo.Expression,
) (api.ExpressionEmission, error) {
	leftName, err := context.Names().Temporary(api.TemporaryArrayComparison)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	rightName, err := context.Names().Temporary(api.TemporaryArrayComparison)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexName, err := context.Names().Temporary(api.TemporaryArrayComparison)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	resultName, err := context.Names().Temporary(api.TemporaryArrayComparison)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index := context.Factory().Identifier(indexName)
	elementEqual, err := context.Values().Equal(
		context.WithRole(api.RoleArrayElement),
		source,
		a.ElementType(),
		callMember(
			context,
			context.Factory().Identifier(leftName),
			arraymember.Get,
			index,
		),
		callMember(
			context,
			context.Factory().Identifier(rightName),
			arraymember.Get,
			index,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	failure := []tsgo.Statement{
		context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().Identifier(resultName),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				context.Factory().FalseLiteral(),
			),
		),
		context.Factory().BreakStatement(nil),
	}
	body := append(
		elementEqual.Before(),
		context.Factory().IfStatement(
			context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
				elementEqual.Value(),
			),
			context.Factory().Block(failure, true),
			nil,
		),
	)
	before := []tsgo.Statement{
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsConst,
			leftName,
			a.storage(context, left),
		),
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsConst,
			rightName,
			a.storage(context, right),
		),
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsLet,
			resultName,
			context.Factory().TrueLiteral(),
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
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().Identifier(resultName),
		elementEqual.Requests(),
	)
}

func (a RuntimeArray) Hash(
	context api.Context,
	source ast.Node,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	arrayName, err := context.Names().Temporary(api.TemporaryArrayHash)
	if err != nil {
		return api.ExpressionEmission{}, err
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
	elementHash, err := context.Values().Hash(
		context.WithRole(api.RoleArrayElement),
		source,
		a.ElementType(),
		callMember(context, array, arraymember.Get, index),
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
		elementHash.Before(),
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
	before := []tsgo.Statement{
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsConst,
			arrayName,
			a.storage(context, value),
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
	}
	return api.NewExpressionEmission(
		before,
		hash,
		api.CombineRequests(
			elementHash.Requests(),
			runtime.Requests(),
		),
	)
}

func arrayComparisonVariable(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			flags,
		),
	)
}
