package reflectiontype

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func reflectionSliceBox(
	scaffold *locationScaffold,
	wrapSlice func(tsgo.Expression) tsgo.Expression,
	value tsgo.Expression,
) tsgo.NewExpression {
	return scaffold.factory.NewExpression(
		scaffold.adapter.Expression(scaffold.factory),
		nil,
		[]tsgo.Expression{wrapSlice(value)},
	)
}

func reflectionSliceVariable(
	factory tsgo.Factory,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier(name),
				nil,
				nil,
				value,
			)},
			flags,
		),
	)
}

func reflectionSliceMember(
	factory tsgo.Factory,
	receiver tsgo.Expression,
	member string,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		receiver,
		nil,
		factory.Identifier(member),
		tsgo.NodeFlagsNone,
	)
}

func reflectionSliceCall(
	factory tsgo.Factory,
	receiver tsgo.Expression,
	member string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		reflectionSliceMember(factory, receiver, member),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func reflectionSliceBinary(
	factory tsgo.Factory,
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		left,
		nil,
		factory.BinaryOperatorToken(operator),
		right,
	)
}

func reflectionSliceAssignment(
	factory tsgo.Factory,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.ExpressionStatement {
	return factory.ExpressionStatement(reflectionSliceBinary(
		factory,
		left,
		tsgo.BinaryOperatorEqualsToken,
		right,
	))
}

func reflectionSliceLoop(
	factory tsgo.Factory,
	index string,
	start tsgo.Expression,
	end tsgo.Expression,
	body []tsgo.Statement,
) tsgo.ForStatement {
	identifier := factory.Identifier(index)
	return factory.ForStatement(
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				identifier,
				nil,
				nil,
				start,
			)},
			tsgo.NodeFlagsLet,
		),
		reflectionSliceBinary(
			factory,
			identifier,
			tsgo.BinaryOperatorLessThanToken,
			end,
		),
		factory.PostfixUnaryExpression(
			identifier,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		factory.Block(body, true),
	)
}

func reflectionSliceIncomingLoop(
	factory tsgo.Factory,
	body []tsgo.Statement,
) []tsgo.Statement {
	index := factory.Identifier("incomingIndex")
	value := factory.Identifier("incomingValue")
	loopBody := append([]tsgo.Statement(nil), body...)
	loopBody = append(loopBody, factory.ExpressionStatement(
		factory.PostfixUnaryExpression(
			index,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
	))
	return []tsgo.Statement{
		reflectionSliceVariable(
			factory,
			tsgo.NodeFlagsLet,
			"incomingIndex",
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
		),
		factory.ForOfStatement(
			nil,
			factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{factory.VariableDeclaration(
					value,
					nil,
					nil,
					nil,
				)},
				tsgo.NodeFlagsConst,
			),
			factory.Identifier("incoming"),
			factory.Block(loopBody, true),
		),
	}
}

func reflectionSliceNumberConversion(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		api.TargetIntrinsicNumber.Expression(factory),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}
