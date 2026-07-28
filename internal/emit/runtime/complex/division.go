package complex

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func buildDivision(
	factory tsgo.Factory,
	name string,
) tsgo.FunctionDeclaration {
	target := builder{factory: factory}
	numberType := target.numberType()
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		target.id(name),
		nil,
		[]tsgo.ParameterDeclaration{
			target.parameter("a", numberType),
			target.parameter("b", numberType),
			target.parameter("c", numberType),
			target.parameter("d", numberType),
		},
		factory.TupleTypeNode([]tsgo.TypeNode{numberType, numberType}),
		factory.Block(target.divisionBody(), true),
	)
}

func (b builder) divisionBody() []tsgo.Statement {
	statements := []tsgo.Statement{
		b.variable(tsgo.NodeFlagsLet, "e", b.numberType(), nil),
		b.variable(tsgo.NodeFlagsLet, "f", b.numberType(), nil),
		b.primaryDivision(),
		b.correction(),
		b.factory.ReturnStatement(b.factory.ArrayLiteralExpression(
			[]tsgo.Expression{b.id("e"), b.id("f")},
			false,
		)),
	}
	return statements
}

func (b builder) primaryDivision() tsgo.IfStatement {
	realDominant := b.binary(
		b.mathCall("abs", b.id("c")),
		tsgo.BinaryOperatorGreaterThanEqualsToken,
		b.mathCall("abs", b.id("d")),
	)
	ratioCD := b.binary(b.id("d"), tsgo.BinaryOperatorSlashToken, b.id("c"))
	denominatorCD := b.binary(
		b.id("c"),
		tsgo.BinaryOperatorPlusToken,
		b.binary(b.id("ratio"), tsgo.BinaryOperatorAsteriskToken, b.id("d")),
	)
	realCD := b.binary(
		b.binary(
			b.id("a"),
			tsgo.BinaryOperatorPlusToken,
			b.binary(b.id("b"), tsgo.BinaryOperatorAsteriskToken, b.id("ratio")),
		),
		tsgo.BinaryOperatorSlashToken,
		b.id("denominator"),
	)
	imagCD := b.binary(
		b.binary(
			b.id("b"),
			tsgo.BinaryOperatorMinusToken,
			b.binary(b.id("a"), tsgo.BinaryOperatorAsteriskToken, b.id("ratio")),
		),
		tsgo.BinaryOperatorSlashToken,
		b.id("denominator"),
	)
	ratioDC := b.binary(b.id("c"), tsgo.BinaryOperatorSlashToken, b.id("d"))
	denominatorDC := b.binary(
		b.id("d"),
		tsgo.BinaryOperatorPlusToken,
		b.binary(b.id("ratio"), tsgo.BinaryOperatorAsteriskToken, b.id("c")),
	)
	realDC := b.binary(
		b.binary(
			b.binary(b.id("a"), tsgo.BinaryOperatorAsteriskToken, b.id("ratio")),
			tsgo.BinaryOperatorPlusToken,
			b.id("b"),
		),
		tsgo.BinaryOperatorSlashToken,
		b.id("denominator"),
	)
	imagDC := b.binary(
		b.binary(
			b.binary(b.id("b"), tsgo.BinaryOperatorAsteriskToken, b.id("ratio")),
			tsgo.BinaryOperatorMinusToken,
			b.id("a"),
		),
		tsgo.BinaryOperatorSlashToken,
		b.id("denominator"),
	)
	return b.factory.IfStatement(
		realDominant,
		b.factory.Block(
			b.divisionBranch(ratioCD, denominatorCD, realCD, imagCD),
			true,
		),
		b.factory.Block(
			b.divisionBranch(ratioDC, denominatorDC, realDC, imagDC),
			true,
		),
	)
}

func (b builder) divisionBranch(
	ratio tsgo.Expression,
	denominator tsgo.Expression,
	realPart tsgo.Expression,
	imaginaryPart tsgo.Expression,
) []tsgo.Statement {
	return []tsgo.Statement{
		b.variable(tsgo.NodeFlagsConst, "ratio", nil, ratio),
		b.variable(tsgo.NodeFlagsConst, "denominator", nil, denominator),
		b.assign("e", realPart),
		b.assign("f", imaginaryPart),
	}
}

func (b builder) correction() tsgo.IfStatement {
	bothNaN := b.binary(
		b.numberCall("isNaN", b.id("e")),
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		b.numberCall("isNaN", b.id("f")),
	)
	return b.factory.IfStatement(
		bothNaN,
		b.factory.Block([]tsgo.Statement{
			b.factory.IfStatement(
				b.zeroDivisorCondition(),
				b.zeroDivisorCorrection(),
				b.factory.IfStatement(
					b.infiniteNumeratorCondition(),
					b.infiniteNumeratorCorrection(),
					b.factory.IfStatement(
						b.infiniteDivisorCondition(),
						b.infiniteDivisorCorrection(),
						nil,
					),
				),
			),
		}, true),
		nil,
	)
}

func (b builder) zeroDivisorCondition() tsgo.Expression {
	divisorZero := b.binary(
		b.binary(
			b.id("c"),
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			b.zero(),
		),
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		b.binary(
			b.id("d"),
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			b.zero(),
		),
	)
	numeratorKnown := b.binary(
		b.prefix(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			b.numberCall("isNaN", b.id("a")),
		),
		tsgo.BinaryOperatorBarBarToken,
		b.prefix(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			b.numberCall("isNaN", b.id("b")),
		),
	)
	return b.binary(
		divisorZero,
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		numeratorKnown,
	)
}

func (b builder) zeroDivisorCorrection() tsgo.Block {
	signedInfinity := b.factory.ConditionalExpression(
		b.negativeSign(b.id("c")),
		b.factory.QuestionToken(),
		b.prefix(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			b.id("Infinity"),
		),
		b.factory.ColonToken(),
		b.id("Infinity"),
	)
	return b.factory.Block([]tsgo.Statement{
		b.variable(
			tsgo.NodeFlagsConst,
			"signedInfinity",
			nil,
			signedInfinity,
		),
		b.assign(
			"e",
			b.binary(
				b.id("signedInfinity"),
				tsgo.BinaryOperatorAsteriskToken,
				b.id("a"),
			),
		),
		b.assign(
			"f",
			b.binary(
				b.id("signedInfinity"),
				tsgo.BinaryOperatorAsteriskToken,
				b.id("b"),
			),
		),
	}, true)
}

func (b builder) infiniteNumeratorCondition() tsgo.Expression {
	numeratorInfinite := b.binary(
		b.isInfinite(b.id("a")),
		tsgo.BinaryOperatorBarBarToken,
		b.isInfinite(b.id("b")),
	)
	divisorFinite := b.binary(
		b.numberCall("isFinite", b.id("c")),
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		b.numberCall("isFinite", b.id("d")),
	)
	return b.binary(
		numeratorInfinite,
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		divisorFinite,
	)
}

func (b builder) infiniteNumeratorCorrection() tsgo.Block {
	return b.factory.Block([]tsgo.Statement{
		b.assign("a", b.infToOne(b.id("a"))),
		b.assign("b", b.infToOne(b.id("b"))),
		b.assign("e", b.binary(
			b.id("Infinity"),
			tsgo.BinaryOperatorAsteriskToken,
			b.binary(
				b.binary(b.id("a"), tsgo.BinaryOperatorAsteriskToken, b.id("c")),
				tsgo.BinaryOperatorPlusToken,
				b.binary(b.id("b"), tsgo.BinaryOperatorAsteriskToken, b.id("d")),
			),
		)),
		b.assign("f", b.binary(
			b.id("Infinity"),
			tsgo.BinaryOperatorAsteriskToken,
			b.binary(
				b.binary(b.id("b"), tsgo.BinaryOperatorAsteriskToken, b.id("c")),
				tsgo.BinaryOperatorMinusToken,
				b.binary(b.id("a"), tsgo.BinaryOperatorAsteriskToken, b.id("d")),
			),
		)),
	}, true)
}

func (b builder) infiniteDivisorCondition() tsgo.Expression {
	divisorInfinite := b.binary(
		b.isInfinite(b.id("c")),
		tsgo.BinaryOperatorBarBarToken,
		b.isInfinite(b.id("d")),
	)
	numeratorFinite := b.binary(
		b.numberCall("isFinite", b.id("a")),
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		b.numberCall("isFinite", b.id("b")),
	)
	return b.binary(
		divisorInfinite,
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		numeratorFinite,
	)
}

func (b builder) infiniteDivisorCorrection() tsgo.Block {
	return b.factory.Block([]tsgo.Statement{
		b.assign("c", b.infToOne(b.id("c"))),
		b.assign("d", b.infToOne(b.id("d"))),
		b.assign("e", b.binary(
			b.zero(),
			tsgo.BinaryOperatorAsteriskToken,
			b.binary(
				b.binary(b.id("a"), tsgo.BinaryOperatorAsteriskToken, b.id("c")),
				tsgo.BinaryOperatorPlusToken,
				b.binary(b.id("b"), tsgo.BinaryOperatorAsteriskToken, b.id("d")),
			),
		)),
		b.assign("f", b.binary(
			b.zero(),
			tsgo.BinaryOperatorAsteriskToken,
			b.binary(
				b.binary(b.id("b"), tsgo.BinaryOperatorAsteriskToken, b.id("c")),
				tsgo.BinaryOperatorMinusToken,
				b.binary(b.id("a"), tsgo.BinaryOperatorAsteriskToken, b.id("d")),
			),
		)),
	}, true)
}

func (b builder) isInfinite(value tsgo.Expression) tsgo.Expression {
	return b.binary(
		b.mathCall("abs", value),
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		b.id("Infinity"),
	)
}

func (b builder) negativeSign(value tsgo.Expression) tsgo.Expression {
	return b.binary(
		b.binary(value, tsgo.BinaryOperatorLessThanToken, b.zero()),
		tsgo.BinaryOperatorBarBarToken,
		b.objectIs(value, b.negativeZero()),
	)
}

func (b builder) infToOne(value tsgo.Expression) tsgo.Expression {
	signedOne := b.factory.ConditionalExpression(
		b.negativeSign(value),
		b.factory.QuestionToken(),
		b.prefix(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			b.one(),
		),
		b.factory.ColonToken(),
		b.one(),
	)
	signedZero := b.factory.ConditionalExpression(
		b.negativeSign(value),
		b.factory.QuestionToken(),
		b.negativeZero(),
		b.factory.ColonToken(),
		b.zero(),
	)
	return b.factory.ConditionalExpression(
		b.isInfinite(value),
		b.factory.QuestionToken(),
		signedOne,
		b.factory.ColonToken(),
		signedZero,
	)
}
