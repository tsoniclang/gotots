package slicevalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func MakeAggregate(
	context api.Context,
	targetElement api.TypeEmission,
	source ast.Node,
	elementType types.Type,
	length tsgo.Expression,
	capacity tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	next, err := zeroElement(context, source, elementType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	resultName, err := context.Names().Temporary(
		api.TemporarySliceConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexName, err := context.Names().Temporary(
		api.TemporarySliceConstruction,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	allocation, err := context.Names().Runtime(
		api.RuntimeSliceStorage,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result := context.Factory().Identifier(resultName)
	index := context.Factory().Identifier(indexName)
	target := context.Factory().CallExpression(
		context.Factory().Identifier(allocation.Name()),
		nil,
		[]tsgo.TypeNode{targetElement.Value()},
		[]tsgo.Expression{length, capacity},
		tsgo.NodeFlagsNone,
	)
	statements := append([]tsgo.Statement(nil), before...)
	statements = append(
		statements,
		sliceVariable(
			context,
			tsgo.NodeFlagsConst,
			resultName,
			target,
		),
		sliceLoop(
			context,
			index,
			sliceProperty(
				context,
				result,
				runtimeslice.MemberName(runtimeslice.MemberCapacity),
			),
			"0",
			append(
				next.Before(),
				context.Factory().ExpressionStatement(sliceCall(
					context,
					result,
					runtimeslice.StorageInitializeMember,
					index,
					next.Value(),
				)),
			),
		),
	)
	return api.NewExpressionEmission(
		statements,
		result,
		api.CombineRequests(
			requests,
			targetElement.Requests(),
			next.Requests(),
			allocation.Requests(),
		),
	)
}

func sliceVariable(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				context.Factory().Identifier(name),
				nil,
				nil,
				value,
			)},
			flags,
		),
	)
}

func sliceLoop(
	context api.Context,
	index tsgo.Identifier,
	length tsgo.Expression,
	start string,
	body []tsgo.Statement,
) tsgo.ForStatement {
	return sliceLoopFrom(
		context,
		index,
		length,
		context.Factory().NumericLiteral(start, tsgo.TokenFlagsNone),
		body,
	)
}

func sliceLoopFrom(
	context api.Context,
	index tsgo.Identifier,
	length tsgo.Expression,
	start tsgo.Expression,
	body []tsgo.Statement,
) tsgo.ForStatement {
	return context.Factory().ForStatement(
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				index,
				nil,
				nil,
				start,
			)},
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

func sliceProperty(
	context api.Context,
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		receiver,
		nil,
		context.Factory().Identifier(name),
		tsgo.NodeFlagsNone,
	)
}

func sliceCall(
	context api.Context,
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		sliceProperty(context, receiver, name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func sliceBinary(
	context api.Context,
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return context.Factory().BinaryExpression(
		nil,
		left,
		nil,
		context.Factory().BinaryOperatorToken(operator),
		right,
	)
}

func sliceAssign(
	context api.Context,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.ExpressionStatement {
	return context.Factory().ExpressionStatement(sliceBinary(
		context,
		left,
		tsgo.BinaryOperatorEqualsToken,
		right,
	))
}

func sliceNumber(context api.Context, value string) tsgo.NumericLiteral {
	return context.Factory().NumericLiteral(value, tsgo.TokenFlagsNone)
}
