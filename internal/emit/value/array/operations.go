package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (a RuntimeArray) Zero(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
) (api.ExpressionEmission, error) {
	if a.aggregate {
		if a.Length() == 0 {
			target, requests, err := a.runtimeOperation(
				context,
				children,
				api.RuntimeArrayAllocate,
				a.lengthLiteral(context),
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			return a.wrap(context, api.DirectExpression(target, requests...))
		}
		loopZero, err := context.Values().Zero(
			context,
			source,
			a.ElementType(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		loopZero, err = a.storeElement(context, source, loopZero)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		resultName, err := context.Names().Temporary(
			api.TemporaryArrayConstruction,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		indexName, err := context.Names().Temporary(
			api.TemporaryArrayConstruction,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		target, runtimeRequests, err := a.runtimeOperation(
			context,
			children,
			api.RuntimeArrayAllocate,
			a.lengthLiteral(context),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		result := context.Factory().Identifier(resultName)
		index := context.Factory().Identifier(indexName)
		body := append(
			loopZero.Before(),
			context.Factory().ExpressionStatement(callMember(
				context,
				result,
				arraymember.Set,
				index,
				loopZero.Value(),
			)),
		)
		before := append(
			[]tsgo.Statement{},
			arrayComparisonVariable(
				context,
				tsgo.NodeFlagsConst,
				resultName,
				target,
			),
			arrayConstructionLoop(
				context,
				index,
				a.lengthLiteral(context),
				"0",
				body,
			),
		)
		emission, err := api.NewExpressionEmission(
			before,
			result,
			api.CombineRequests(
				loopZero.Requests(),
				runtimeRequests,
			),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return a.wrap(context, emission)
	}
	elementZero, err := context.Values().Zero(
		context,
		source,
		a.ElementType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementZero, err = a.storeElement(context, source, elementZero)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(elementZero.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	typeArguments, typeRequests, err := a.targetTypeArguments(context, children)
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
	children api.ChildEmitter,
	source ast.Node,
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
		sourceName, err := context.Names().Temporary(
			api.TemporaryArrayConstruction,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		resultName, err := context.Names().Temporary(
			api.TemporaryArrayConstruction,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		indexName, err := context.Names().Temporary(
			api.TemporaryArrayConstruction,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		sourceValue := context.Factory().Identifier(sourceName)
		result := context.Factory().Identifier(resultName)
		index := context.Factory().Identifier(indexName)
		loaded, err := a.loadElement(
			context,
			source,
			callMember(
				context,
				sourceValue,
				arraymember.Get,
				index,
			),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		elementCopy, err := context.Values().Transfer(
			context.WithRole(api.RoleArrayElement),
			source,
			a.ElementType(),
			a.ElementType(),
			api.ValueTransferCopy,
			loaded,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		elementCopy, err = a.storeElement(context, source, elementCopy)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		target, runtimeRequests, err := a.runtimeOperation(
			context,
			children,
			api.RuntimeArrayAllocate,
			a.lengthLiteral(context),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before := append(
			value.Before(),
			arrayComparisonVariable(
				context,
				tsgo.NodeFlagsConst,
				sourceName,
				a.storage(context, value.Value()),
			),
		)
		before = append(
			before,
			arrayComparisonVariable(
				context,
				tsgo.NodeFlagsConst,
				resultName,
				target,
			),
			arrayConstructionLoop(
				context,
				index,
				a.lengthLiteral(context),
				"0",
				append(
					elementCopy.Before(),
					context.Factory().ExpressionStatement(callMember(
						context,
						result,
						arraymember.Set,
						index,
						elementCopy.Value(),
					)),
				),
			),
		)
		copied, err := api.NewExpressionEmission(
			before,
			result,
			api.CombineRequests(
				value.Requests(),
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

func arrayConstructionLoop(
	context api.Context,
	index tsgo.Identifier,
	length tsgo.Expression,
	start string,
	body []tsgo.Statement,
) tsgo.ForStatement {
	return context.Factory().ForStatement(
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					index,
					nil,
					nil,
					context.Factory().NumericLiteral(
						start,
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
			length,
		),
		context.Factory().PostfixUnaryExpression(
			index,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		context.Factory().Block(body, true),
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
	leftElement, err := a.loadElement(
		context,
		source,
		callMember(
			context,
			context.Factory().Identifier(leftName),
			arraymember.Get,
			index,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	rightElement, err := a.loadElement(
		context,
		source,
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
	elementEqual, err := context.Values().Equal(
		context.WithRole(api.RoleArrayElement),
		source,
		a.ElementType(),
		leftElement.Value(),
		rightElement.Value(),
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
		leftElement.Before(),
		rightElement.Before()...,
	)
	body = append(
		body,
		elementEqual.Before()...,
	)
	body = append(
		body,
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
		api.CombineRequests(
			leftElement.Requests(),
			rightElement.Requests(),
			elementEqual.Requests(),
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
