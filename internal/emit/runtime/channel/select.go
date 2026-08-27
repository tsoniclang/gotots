package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) selectReadyFunction() tsgo.FunctionDeclaration {
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(b.selectReadyName),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"cases",
				b.arrayType(b.typeReference(b.caseName)),
			),
		},
		b.unionType(b.numberType(), b.undefinedType()),
		b.selectReadyBody(),
	)
}

func (b builder) selectReadyBody() tsgo.Block {
	return b.factory.Block([]tsgo.Statement{
		b.variable(
			tsgo.NodeFlagsConst,
			"attempt",
			b.selectAttemptResultType(),
			b.call(b.id(b.selectAttemptName), b.id("cases")),
		),
		b.factory.IfStatement(
			b.strictUndefined(b.id("attempt")),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.undefined()),
			}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"failure",
			b.unionType(b.objectType(), b.undefinedType()),
			b.element(b.id("attempt"), b.number("1")),
		),
		b.factory.IfStatement(
			b.strictDefined(b.id("failure")),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.rethrow(b.id("failure"))),
			}, true),
			nil,
		),
		b.returnStatement(b.element(b.id("attempt"), b.number("0"))),
	}, true)
}

func (b builder) selectAttemptFunction() tsgo.FunctionDeclaration {
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(b.selectAttemptName),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"cases",
				b.arrayType(b.typeReference(b.caseName)),
			),
		},
		b.selectAttemptResultType(),
		b.selectAttemptBody(),
	)
}

func (b builder) selectAttemptResultType() tsgo.TypeNode {
	return b.unionType(
		b.tupleType(
			b.numberType(),
			b.unionType(b.objectType(), b.undefinedType()),
		),
		b.undefinedType(),
	)
}

func (b builder) selectAttemptBody() tsgo.Block {
	index := b.variableDeclaration(
		"index",
		b.numberType(),
		b.number("0"),
	)
	return b.factory.Block([]tsgo.Statement{
		b.variable(
			tsgo.NodeFlagsConst,
			"ready",
			b.arrayType(b.numberType()),
			b.arrayLiteral(),
		),
		b.factory.ForStatement(
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{index},
				tsgo.NodeFlagsLet,
			),
			b.binary(
				b.id("index"),
				tsgo.BinaryOperatorLessThanToken,
				b.arrayLength(b.id("cases")),
			),
			b.increment(b.id("index")),
			b.factory.Block([]tsgo.Statement{
				b.factory.IfStatement(
					b.methodCall(
						b.denseElement(
							b.id("cases"),
							b.id("index"),
							b.typeReference(b.caseName),
						),
						"ready",
					),
					b.factory.Block([]tsgo.Statement{
						b.expression(b.methodCall(
							b.id("ready"),
							"push",
							b.id("index"),
						)),
					}, true),
					nil,
				),
			}, true),
		),
		b.factory.IfStatement(
			b.strictEqual(
				b.arrayLength(b.id("ready")),
				b.number("0"),
			),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.undefined()),
			}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"readyIndex",
			b.numberType(),
			b.staticCall(
				"Math",
				"floor",
				b.multiply(
					b.staticCall("Math", "random"),
					b.arrayLength(b.id("ready")),
				),
			),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"selectedIndex",
			b.numberType(),
			b.denseElement(
				b.id("ready"),
				b.id("readyIndex"),
				b.numberType(),
			),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"outcome",
			b.selectCommitType(),
			b.methodCall(
				b.denseElement(
					b.id("cases"),
					b.id("selectedIndex"),
					b.typeReference(b.caseName),
				),
				"commit",
			),
		),
		b.factory.IfStatement(
			b.strictEqual(
				b.id("outcome"),
				b.factory.FalseLiteral(),
			),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.undefined()),
			}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"failure",
			b.unionType(b.objectType(), b.undefinedType()),
			b.factory.ConditionalExpression(
				b.strictEqual(
					b.id("outcome"),
					b.factory.TrueLiteral(),
				),
				b.factory.QuestionToken(),
				b.undefined(),
				b.factory.ColonToken(),
				b.id("outcome"),
			),
		),
		b.returnStatement(b.arrayLiteral(
			b.id("selectedIndex"),
			b.id("failure"),
		)),
	}, true)
}

func (b builder) selectFunction() tsgo.FunctionDeclaration {
	return b.factory.FunctionDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		nil,
		b.id(b.selectName),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"cases",
				b.arrayType(b.typeReference(b.caseName)),
			),
		},
		b.numberType(),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"immediate",
				b.unionType(b.numberType(), b.undefinedType()),
				b.call(b.id(b.selectReadyName), b.id("cases")),
			),
			b.factory.IfStatement(
				b.strictUndefined(b.id("immediate")),
				b.factory.Block([]tsgo.Statement{
					b.expression(b.panic("serial select would block")),
				}, true),
				nil,
			),
			b.returnStatement(b.id("immediate")),
		}, true),
	)
}
