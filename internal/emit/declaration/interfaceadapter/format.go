package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func formatStringProperty(
	factory tsgo.Factory,
	sourceType types.Type,
) tsgo.PropertyDeclaration {
	_, basic, ok := formatBasic(factory, sourceType)
	initialValue := tsgo.Expression(factory.FalseLiteral())
	if ok && basic.Kind() == types.String {
		initialValue = factory.TrueLiteral()
	}
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacecontract.FormatStringMember),
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		initialValue,
	)
}

func formatMethod(
	context api.Context,
	sourceType types.Type,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	factory := context.Factory()
	verb := factory.Identifier("verb")
	precision := factory.Identifier("precision")
	body := []tsgo.Statement{returnWhen(
		factory,
		verbEquals(factory, verb, "T"),
		factory.StringLiteral(dynamicTypeSpelling(sourceType), tsgo.TokenFlagsNone),
	)}
	value, basic, basicOK := formatBasic(factory, sourceType)
	if basicOK {
		body = append(
			body,
			formatBasicStatements(factory, value, basic, verb, precision)...,
		)
	}
	body = append(body, factory.ReturnStatement(
		panicruntime.Call(
			factory,
			panicReference.Name(),
			factory.BinaryExpression(
				nil,
				factory.StringLiteral(
					"unsupported fmt verb for ",
					tsgo.TokenFlagsNone,
				),
				nil,
				factory.BinaryOperatorToken(tsgo.BinaryOperatorPlusToken),
				factory.StringLiteral(
					dynamicTypeSpelling(sourceType),
					tsgo.TokenFlagsNone,
				),
			),
		),
	))
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.FormatMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"verb",
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
			),
			parameter(
				factory,
				"_flags",
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
			),
			parameter(
				factory,
				"precision",
				factory.UnionTypeNode([]tsgo.TypeNode{
					factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
					factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
				}),
			),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		factory.Block(body, true),
	), panicReference.Requests(), nil
}

func formatBasic(
	factory tsgo.Factory,
	sourceType types.Type,
) (tsgo.Expression, *types.Basic, bool) {
	value := payload(factory, factory.ThisExpression())
	if defined, ok := definedtype.ResolveBasic(sourceType); ok {
		basic, basicOK := defined.Basic()
		return defined.Unwrap(factory, value), basic, basicOK
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return value, basic, ok
}

func formatBasicStatements(
	factory tsgo.Factory,
	value tsgo.Expression,
	basic *types.Basic,
	verb tsgo.Expression,
	precision tsgo.Expression,
) []tsgo.Statement {
	switch basic.Info() & (types.IsBoolean | types.IsString | types.IsInteger | types.IsFloat) {
	case types.IsBoolean:
		return []tsgo.Statement{returnWhen(
			factory,
			verbOneOf(factory, verb, "v", "t"),
			factory.ConditionalExpression(
				value,
				factory.QuestionToken(),
				factory.StringLiteral("true", tsgo.TokenFlagsNone),
				factory.ColonToken(),
				factory.StringLiteral("false", tsgo.TokenFlagsNone),
			),
		)}
	case types.IsString:
		return formatStringStatements(factory, value, verb, precision)
	case types.IsInteger:
		return formatIntegerStatements(factory, value, verb)
	case types.IsFloat:
		return formatFloatStatements(factory, value, verb, precision)
	default:
		return nil
	}
}

func formatStringStatements(
	factory tsgo.Factory,
	value tsgo.Expression,
	verb tsgo.Expression,
	precision tsgo.Expression,
) []tsgo.Statement {
	selected := factory.ConditionalExpression(
		formatUndefined(factory, precision),
		factory.QuestionToken(),
		value,
		factory.ColonToken(),
		methodCall(
			factory,
			value,
			"slice",
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			precision,
		),
	)
	return []tsgo.Statement{
		returnWhen(factory, verbOneOf(factory, verb, "v", "s"), selected),
		returnWhen(
			factory,
			verbEquals(factory, verb, "q"),
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
	}
}

func formatIntegerStatements(
	factory tsgo.Factory,
	value tsgo.Expression,
	verb tsgo.Expression,
) []tsgo.Statement {
	decimal := methodCall(
		factory,
		value,
		"toString",
		factory.NumericLiteral("10", tsgo.TokenFlagsNone),
	)
	hexadecimal := methodCall(
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
	quotedCharacter := quoteCharacter(factory, character)
	return []tsgo.Statement{
		returnWhen(factory, verbOneOf(factory, verb, "v", "d"), decimal),
		returnWhen(factory, verbEquals(factory, verb, "x"), hexadecimal),
		returnWhen(
			factory,
			verbEquals(factory, verb, "X"),
			methodCall(factory, hexadecimal, "toUpperCase"),
		),
		returnWhen(factory, verbEquals(factory, verb, "c"), character),
		returnWhen(factory, verbEquals(factory, verb, "q"), quotedCharacter),
	}
}

func quoteCharacter(
	factory tsgo.Factory,
	character tsgo.Expression,
) tsgo.Expression {
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
	interior := methodCall(
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

func formatFloatStatements(
	factory tsgo.Factory,
	value tsgo.Expression,
	verb tsgo.Expression,
	precision tsgo.Expression,
) []tsgo.Statement {
	plain := methodCall(factory, value, "toString")
	fixed := factory.ConditionalExpression(
		formatUndefined(factory, precision),
		factory.QuestionToken(),
		plain,
		factory.ColonToken(),
		methodCall(factory, value, "toFixed", precision),
	)
	exponential := factory.ConditionalExpression(
		formatUndefined(factory, precision),
		factory.QuestionToken(),
		methodCall(factory, value, "toExponential"),
		factory.ColonToken(),
		methodCall(factory, value, "toExponential", precision),
	)
	return []tsgo.Statement{
		returnWhen(factory, verbOneOf(factory, verb, "v", "g"), plain),
		returnWhen(factory, verbEquals(factory, verb, "f"), fixed),
		returnWhen(factory, verbEquals(factory, verb, "e"), exponential),
		returnWhen(
			factory,
			verbEquals(factory, verb, "E"),
			methodCall(factory, exponential, "toUpperCase"),
		),
	}
}

func dynamicTypeSpelling(sourceType types.Type) string {
	return types.TypeString(sourceType, func(source *types.Package) string {
		return source.Name()
	})
}

func returnWhen(
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

func verbOneOf(
	factory tsgo.Factory,
	verb tsgo.Expression,
	values ...string,
) tsgo.Expression {
	var result tsgo.Expression
	for _, value := range values {
		selected := verbEquals(factory, verb, value)
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

func verbEquals(
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

func methodCall(
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
