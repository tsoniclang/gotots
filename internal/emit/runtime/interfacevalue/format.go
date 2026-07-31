package interfacevalue

import (
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func formatClass(factory tsgo.Factory, panicName string) tsgo.ClassDeclaration {
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier("GoInterfaceFormat"),
		nil,
		nil,
		[]tsgo.ClassElement{
			formatOtherMethod(factory, panicName),
			formatBooleanMethod(factory, panicName),
			formatStringMethod(factory, panicName),
			formatIntegerMethod(factory, panicName),
			formatFloatMethod(factory, panicName),
		},
	)
}

func formatOtherMethod(factory tsgo.Factory, panicName string) tsgo.MethodDeclaration {
	return formatRuntimeMethod(
		factory,
		panicName,
		interfacecontract.FormatOtherMember,
		nil,
		nil,
	)
}

func formatBooleanMethod(factory tsgo.Factory, panicName string) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	verb := factory.Identifier("verb")
	return formatRuntimeMethod(
		factory,
		panicName,
		interfacecontract.FormatBooleanMember,
		[]tsgo.ParameterDeclaration{formatParameter(
			factory,
			"value",
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		)},
		[]tsgo.Statement{formatReturnWhen(
			factory,
			formatVerbOneOf(factory, verb, "v", "t"),
			factory.ConditionalExpression(
				value,
				factory.QuestionToken(),
				factory.StringLiteral("true", tsgo.TokenFlagsNone),
				factory.ColonToken(),
				factory.StringLiteral("false", tsgo.TokenFlagsNone),
			),
		)},
	)
}

func formatStringMethod(factory tsgo.Factory, panicName string) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	verb := factory.Identifier("verb")
	precision := factory.Identifier("precision")
	selected := factory.ConditionalExpression(
		formatUndefined(factory, precision),
		factory.QuestionToken(),
		value,
		factory.ColonToken(),
		formatMemberCall(
			factory,
			value,
			"slice",
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			precision,
		),
	)
	return formatRuntimeMethod(
		factory,
		panicName,
		interfacecontract.FormatStringValueMember,
		[]tsgo.ParameterDeclaration{
			formatParameter(
				factory,
				"value",
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
			),
			formatParameter(factory, "precision", formatPrecisionType(factory)),
		},
		[]tsgo.Statement{
			formatReturnWhen(
				factory,
				formatVerbOneOf(factory, verb, "v", "s"),
				selected,
			),
			formatReturnWhen(
				factory,
				formatVerbEquals(factory, verb, "q"),
				factory.CallExpression(
					factory.PropertyAccessExpression(
						factory.Identifier("JSON"),
						nil,
						factory.Identifier("stringify"),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{selected},
					tsgo.NodeFlagsNone,
				),
			),
		},
	)
}

func formatIntegerMethod(factory tsgo.Factory, panicName string) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	verb := factory.Identifier("verb")
	decimal := formatMemberCall(
		factory,
		value,
		"toString",
		factory.NumericLiteral("10", tsgo.TokenFlagsNone),
	)
	hexadecimal := formatMemberCall(
		factory,
		value,
		"toString",
		factory.NumericLiteral("16", tsgo.TokenFlagsNone),
	)
	character := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("String"),
			nil,
			factory.Identifier("fromCodePoint"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{factory.CallExpression(
			factory.Identifier("Number"),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		)},
		tsgo.NodeFlagsNone,
	)
	return formatRuntimeMethod(
		factory,
		panicName,
		interfacecontract.FormatIntegerMember,
		[]tsgo.ParameterDeclaration{formatParameter(
			factory,
			"value",
			factory.UnionTypeNode([]tsgo.TypeNode{
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
			}),
		)},
		[]tsgo.Statement{
			formatReturnWhen(factory, formatVerbOneOf(factory, verb, "v", "d"), decimal),
			formatReturnWhen(factory, formatVerbEquals(factory, verb, "x"), hexadecimal),
			formatReturnWhen(
				factory,
				formatVerbEquals(factory, verb, "X"),
				formatMemberCall(factory, hexadecimal, "toUpperCase"),
			),
			formatReturnWhen(factory, formatVerbEquals(factory, verb, "c"), character),
			formatReturnWhen(
				factory,
				formatVerbEquals(factory, verb, "q"),
				formatQuoteCharacter(factory, character),
			),
		},
	)
}

func formatFloatMethod(factory tsgo.Factory, panicName string) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	verb := factory.Identifier("verb")
	precision := factory.Identifier("precision")
	plain := formatMemberCall(factory, value, "toString")
	fixed := factory.ConditionalExpression(
		formatUndefined(factory, precision),
		factory.QuestionToken(),
		plain,
		factory.ColonToken(),
		formatMemberCall(factory, value, "toFixed", precision),
	)
	exponential := factory.ConditionalExpression(
		formatUndefined(factory, precision),
		factory.QuestionToken(),
		formatMemberCall(factory, value, "toExponential"),
		factory.ColonToken(),
		formatMemberCall(factory, value, "toExponential", precision),
	)
	return formatRuntimeMethod(
		factory,
		panicName,
		interfacecontract.FormatFloatMember,
		[]tsgo.ParameterDeclaration{
			formatParameter(
				factory,
				"value",
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
			),
			formatParameter(factory, "precision", formatPrecisionType(factory)),
		},
		[]tsgo.Statement{
			formatReturnWhen(factory, formatVerbOneOf(factory, verb, "v", "g"), plain),
			formatReturnWhen(factory, formatVerbEquals(factory, verb, "f"), fixed),
			formatReturnWhen(factory, formatVerbEquals(factory, verb, "e"), exponential),
			formatReturnWhen(
				factory,
				formatVerbEquals(factory, verb, "E"),
				formatMemberCall(factory, exponential, "toUpperCase"),
			),
		},
	)
}

func formatRuntimeMethod(
	factory tsgo.Factory,
	panicName string,
	name string,
	leading []tsgo.ParameterDeclaration,
	body []tsgo.Statement,
) tsgo.MethodDeclaration {
	typeName := factory.Identifier("typeName")
	verb := factory.Identifier("verb")
	parameters := append(
		append([]tsgo.ParameterDeclaration(nil), leading...),
		formatParameter(
			factory,
			"typeName",
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		),
		formatParameter(
			factory,
			"verb",
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		),
	)
	statements := []tsgo.Statement{formatReturnWhen(
		factory,
		formatVerbEquals(factory, verb, "T"),
		typeName,
	)}
	statements = append(statements, body...)
	statements = append(statements, factory.ReturnStatement(panicruntime.Call(
		factory,
		panicName,
		factory.BinaryExpression(
			nil,
			factory.StringLiteral("unsupported fmt verb for ", tsgo.TokenFlagsNone),
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorPlusToken),
			typeName,
		),
	)))
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		nil,
		parameters,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		factory.Block(statements, true),
	)
}

func formatQuoteCharacter(factory tsgo.Factory, character tsgo.Expression) tsgo.Expression {
	encoded := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("JSON"),
			nil,
			factory.Identifier("stringify"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{character},
		tsgo.NodeFlagsNone,
	)
	interior := formatMemberCall(
		factory,
		encoded,
		"slice",
		factory.NumericLiteral("1", tsgo.TokenFlagsNone),
		factory.PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			factory.NumericLiteral("1", tsgo.TokenFlagsNone),
		),
	)
	return factory.BinaryExpression(
		nil,
		factory.BinaryExpression(
			nil,
			factory.StringLiteral("'", tsgo.TokenFlagsNone),
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorPlusToken),
			interior,
		),
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorPlusToken),
		factory.StringLiteral("'", tsgo.TokenFlagsNone),
	)
}

func formatReturnWhen(
	factory tsgo.Factory,
	condition tsgo.Expression,
	value tsgo.Expression,
) tsgo.IfStatement {
	return factory.IfStatement(
		condition,
		factory.Block([]tsgo.Statement{factory.ReturnStatement(value)}, true),
		nil,
	)
}

func formatVerbOneOf(
	factory tsgo.Factory,
	verb tsgo.Expression,
	values ...string,
) tsgo.Expression {
	var result tsgo.Expression
	for _, value := range values {
		selected := formatVerbEquals(factory, verb, value)
		if result == nil {
			result = selected
			continue
		}
		result = factory.BinaryExpression(
			nil,
			result,
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorBarBarToken),
			selected,
		)
	}
	return result
}

func formatVerbEquals(
	factory tsgo.Factory,
	verb tsgo.Expression,
	value string,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		verb,
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsEqualsEqualsToken),
		factory.StringLiteral(value, tsgo.TokenFlagsNone),
	)
}

func formatUndefined(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsEqualsEqualsToken),
		factory.Identifier("undefined"),
	)
}

func formatMemberCall(
	factory tsgo.Factory,
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			receiver,
			nil,
			factory.Identifier(name),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func formatParameter(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}

func formatPrecisionType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
}
