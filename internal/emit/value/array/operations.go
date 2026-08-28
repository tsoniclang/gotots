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
		loopZero, err := a.zeroElement(
			context.WithRole(api.RoleArrayElement),
			source,
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
	elementZero, err := a.zeroElement(
		context.WithRole(api.RoleArrayElement),
		source,
	)
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
		stored, err := a.storage(
			context.WithRole(api.RoleArrayReceiver),
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before := append(
			stored.Before(),
			arrayComparisonVariable(
				context,
				tsgo.NodeFlagsConst,
				sourceName,
				stored.Value(),
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
				stored.Requests(),
				elementCopy.Requests(),
				runtimeRequests,
			),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return a.wrap(context, copied)
	}
	stored, err := a.storage(
		context.WithRole(api.RoleArrayReceiver),
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	copied, err := api.NewExpressionEmission(
		stored.Before(),
		callMember(
			context,
			stored.Value(),
			arraymember.Copy,
		),
		stored.Requests(),
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
	leftStorage, err := a.storage(
		context.WithRole(api.RoleDefinedValue),
		api.DirectExpression(left),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	rightStorage, err := a.storage(
		context.WithRole(api.RoleDefinedValue),
		api.DirectExpression(right),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	leftName, err := context.Names().Temporary(api.TemporaryArrayComparison)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	rightName, err := context.Names().Temporary(api.TemporaryArrayComparison)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if a.Length() == 0 {
		before := append(leftStorage.Before(), rightStorage.Before()...)
		before = append(
			before,
			arrayComparisonVariable(
				context,
				tsgo.NodeFlagsConst,
				leftName,
				leftStorage.Value(),
			),
			arrayComparisonVariable(
				context,
				tsgo.NodeFlagsConst,
				rightName,
				rightStorage.Value(),
			),
		)
		return api.NewExpressionEmission(
			before,
			context.Factory().TrueLiteral(),
			api.CombineRequests(
				leftStorage.Requests(),
				rightStorage.Requests(),
			),
		)
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
	before := append(leftStorage.Before(), rightStorage.Before()...)
	before = append(before,
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsConst,
			leftName,
			leftStorage.Value(),
		),
		arrayComparisonVariable(
			context,
			tsgo.NodeFlagsConst,
			rightName,
			rightStorage.Value(),
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
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().Identifier(resultName),
		api.CombineRequests(
			leftStorage.Requests(),
			rightStorage.Requests(),
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
