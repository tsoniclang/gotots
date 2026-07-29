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
				b.factory.ThrowStatement(b.id("failure")),
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
						b.element(
							b.id("cases"),
							b.id("index"),
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
			"selectedIndex",
			b.numberType(),
			b.element(
				b.id("ready"),
				b.staticCall(
					"Math",
					"floor",
					b.multiply(
						b.staticCall("Math", "random"),
						b.arrayLength(b.id("ready")),
					),
				),
			),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"outcome",
			b.selectCommitType(),
			b.methodCall(
				b.element(b.id("cases"), b.id("selectedIndex")),
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
		b.promiseType(b.numberType()),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"immediate",
				b.unionType(b.numberType(), b.undefinedType()),
				b.call(b.id(b.selectReadyName), b.id("cases")),
			),
			b.factory.IfStatement(
				b.strictDefined(b.id("immediate")),
				b.factory.Block([]tsgo.Statement{
					b.returnStatement(b.promiseResolve(
						b.id("immediate"),
					)),
				}, true),
				nil,
			),
			b.returnStatement(b.newPromise(
				b.numberType(),
				b.selectExecutor(),
			)),
		}, true),
	)
}

func (b builder) selectExecutor() tsgo.ArrowFunction {
	cancelType := b.functionType(nil, b.voidType())
	index := b.variableDeclaration(
		"index",
		b.numberType(),
		b.number("0"),
	)
	remaining := b.variableDeclaration(
		"remaining",
		b.numberType(),
		b.subtract(
			b.arrayLength(b.id("order")),
			b.number("1"),
		),
	)
	registration := b.variableDeclaration(
		"registration",
		b.numberType(),
		b.number("0"),
	)
	return b.arrow(
		[]tsgo.ParameterDeclaration{
			b.factory.ParameterDeclaration(
				nil,
				nil,
				b.id("resolve"),
				nil,
				nil,
				nil,
			),
			b.factory.ParameterDeclaration(
				nil,
				nil,
				b.id("reject"),
				nil,
				nil,
				nil,
			),
		},
		b.voidType(),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsLet,
				"settled",
				b.booleanType(),
				b.factory.FalseLiteral(),
			),
			b.variable(
				tsgo.NodeFlagsConst,
				"cancellations",
				b.arrayType(cancelType),
				b.arrayLiteral(),
			),
			b.variable(
				tsgo.NodeFlagsConst,
				"order",
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
					b.expression(b.methodCall(
						b.id("order"),
						"push",
						b.id("index"),
					)),
				}, true),
			),
			b.factory.ForStatement(
				b.factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{remaining},
					tsgo.NodeFlagsLet,
				),
				b.binary(
					b.id("remaining"),
					tsgo.BinaryOperatorGreaterThanToken,
					b.number("0"),
				),
				b.assign(
					b.id("remaining"),
					b.subtract(b.id("remaining"), b.number("1")),
				),
				b.factory.Block([]tsgo.Statement{
					b.variable(
						tsgo.NodeFlagsConst,
						"swapIndex",
						b.numberType(),
						b.staticCall(
							"Math",
							"floor",
							b.multiply(
								b.staticCall("Math", "random"),
								b.add(
									b.id("remaining"),
									b.number("1"),
								),
							),
						),
					),
					b.variable(
						tsgo.NodeFlagsConst,
						"held",
						b.numberType(),
						b.element(
							b.id("order"),
							b.id("remaining"),
						),
					),
					b.expression(b.assign(
						b.element(
							b.id("order"),
							b.id("remaining"),
						),
						b.element(
							b.id("order"),
							b.id("swapIndex"),
						),
					)),
					b.expression(b.assign(
						b.element(
							b.id("order"),
							b.id("swapIndex"),
						),
						b.id("held"),
					)),
				}, true),
			),
			b.factory.ForStatement(
				b.factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{registration},
					tsgo.NodeFlagsLet,
				),
				b.binary(
					b.id("registration"),
					tsgo.BinaryOperatorLessThanToken,
					b.arrayLength(b.id("order")),
				),
				b.increment(b.id("registration")),
				b.factory.Block([]tsgo.Statement{
					b.variable(
						tsgo.NodeFlagsConst,
						"caseIndex",
						b.numberType(),
						b.element(
							b.id("order"),
							b.id("registration"),
						),
					),
					b.expression(b.methodCall(
						b.id("cancellations"),
						"push",
						b.methodCall(
							b.element(
								b.id("cases"),
								b.id("caseIndex"),
							),
							"subscribe",
							b.selectClaim(b.id("caseIndex")),
						),
					)),
				}, true),
			),
		}, true),
	)
}

func (b builder) selectClaim(index tsgo.Expression) tsgo.ArrowFunction {
	cancel := b.variableDeclaration("cancel", nil, nil)
	return b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"failure",
				b.unionType(b.objectType(), b.undefinedType()),
			),
		},
		b.booleanType(),
		b.factory.Block([]tsgo.Statement{
			b.factory.IfStatement(
				b.id("settled"),
				b.factory.Block([]tsgo.Statement{
					b.returnStatement(b.factory.FalseLiteral()),
				}, true),
				nil,
			),
			b.expression(b.assign(
				b.id("settled"),
				b.factory.TrueLiteral(),
			)),
			b.factory.ForOfStatement(
				nil,
				b.factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{cancel},
					tsgo.NodeFlagsConst,
				),
				b.id("cancellations"),
				b.factory.Block([]tsgo.Statement{
					b.expression(b.call(b.id("cancel"))),
				}, true),
			),
			b.factory.IfStatement(
				b.strictDefined(b.id("failure")),
				b.factory.Block([]tsgo.Statement{
					b.expression(b.call(
						b.id("reject"),
						b.id("failure"),
					)),
					b.returnStatement(b.factory.TrueLiteral()),
				}, true),
				nil,
			),
			b.expression(b.call(b.id("resolve"), index)),
			b.returnStatement(b.factory.TrueLiteral()),
		}, true),
	)
}
