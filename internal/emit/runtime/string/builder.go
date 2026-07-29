package stringruntime

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
	panicName string,
) ([]tsgo.Statement, error) {
	if len(symbols) == 0 {
		return nil, &BuildError{}
	}
	seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	statements := make([]tsgo.Statement, 0, len(symbols))
	for _, symbol := range symbols {
		if _, duplicate := seen[symbol]; duplicate {
			return nil, &BuildError{Symbol: symbol}
		}
		seen[symbol] = struct{}{}
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		var statement tsgo.Statement
		switch symbol {
		case api.RuntimeStringIndex:
			statement = stringIndex(
				factory,
				contract.ExportedName(),
				panicName,
			)
		case api.RuntimeStringSlice:
			statement = stringSlice(
				factory,
				contract.ExportedName(),
				panicName,
			)
		case api.RuntimeStringMax:
			statement = orderedString(
				factory,
				contract.ExportedName(),
				tsgo.BinaryOperatorGreaterThanEqualsToken,
			)
		case api.RuntimeStringMin:
			statement = orderedString(
				factory,
				contract.ExportedName(),
				tsgo.BinaryOperatorLessThanEqualsToken,
			)
		case api.RuntimeStringEncodeRune:
			statement = encodeRune(factory, contract.ExportedName())
		case api.RuntimeStringDecodeRune:
			statement = decodeRune(factory, contract.ExportedName())
		default:
			return nil, &BuildError{Symbol: symbol}
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

func orderedString(
	factory tsgo.Factory,
	exportedName string,
	operator tsgo.BinaryOperator,
) tsgo.FunctionDeclaration {
	left := factory.Identifier("left")
	right := factory.Identifier("right")
	targetType := stringType(factory)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(exportedName),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, left, targetType, false),
			parameter(factory, right, targetType, false),
		},
		targetType,
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(factory.ConditionalExpression(
				binary(factory, left, operator, right),
				factory.QuestionToken(),
				left,
				factory.ColonToken(),
				right,
			)),
		}, true),
	)
}

func stringIndex(
	factory tsgo.Factory,
	exportedName string,
	panicName string,
) tsgo.FunctionDeclaration {
	value := factory.Identifier("value")
	index := factory.Identifier("index")
	offset := factory.Identifier("offset")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(exportedName),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, stringType(factory), false),
			parameter(factory, index, indexType(factory), false),
		},
		numberType(factory),
		factory.Block(
			[]tsgo.Statement{
				constNumber(factory, offset, index),
				boundsCheck(
					factory,
					panicName,
					or(
						factory,
						notSafeInteger(factory, offset),
						or(
							factory,
							lessThan(factory, offset, numeric(factory, "0")),
							greaterThanOrEqual(
								factory,
								offset,
								length(factory, value),
							),
						),
					),
					"Go string index out of range",
				),
				factory.ReturnStatement(
					methodCall(
						factory,
						value,
						"charCodeAt",
						[]tsgo.Expression{offset},
					),
				),
			},
			true,
		),
	)
}

func stringSlice(
	factory tsgo.Factory,
	exportedName string,
	panicName string,
) tsgo.FunctionDeclaration {
	value := factory.Identifier("value")
	low := factory.Identifier("low")
	high := factory.Identifier("high")
	start := factory.Identifier("start")
	end := factory.Identifier("end")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(exportedName),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, stringType(factory), false),
			parameter(factory, low, indexType(factory), false),
			parameter(factory, high, indexType(factory), true),
		},
		stringType(factory),
		factory.Block(
			[]tsgo.Statement{
				constNumber(factory, start, low),
				constSliceHigh(factory, end, high, value),
				boundsCheck(
					factory,
					panicName,
					or(
						factory,
						notSafeInteger(factory, start),
						or(
							factory,
							notSafeInteger(factory, end),
							or(
								factory,
								lessThan(factory, start, numeric(factory, "0")),
								or(
									factory,
									greaterThan(factory, start, end),
									greaterThan(factory, end, length(factory, value)),
								),
							),
						),
					),
					"Go string slice bounds out of range",
				),
				factory.ReturnStatement(
					methodCall(
						factory,
						value,
						"slice",
						[]tsgo.Expression{start, end},
					),
				),
			},
			true,
		),
	)
}

func parameter(
	factory tsgo.Factory,
	name tsgo.Identifier,
	targetType tsgo.TypeNode,
	optional bool,
) tsgo.ParameterDeclaration {
	var question tsgo.QuestionToken
	if optional {
		question = factory.QuestionToken()
	}
	return factory.ParameterDeclaration(nil, nil, name, question, targetType, nil)
}

func stringType(factory tsgo.Factory) tsgo.KeywordTypeNode {
	return factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword)
}

func numberType(factory tsgo.Factory) tsgo.KeywordTypeNode {
	return factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func indexType(factory tsgo.Factory) tsgo.UnionTypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		numberType(factory),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
	})
}

func constNumber(
	factory tsgo.Factory,
	target tsgo.Identifier,
	source tsgo.Identifier,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					target,
					nil,
					nil,
					factory.CallExpression(
						factory.Identifier("Number"),
						nil,
						nil,
						[]tsgo.Expression{source},
						tsgo.NodeFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func constSliceHigh(
	factory tsgo.Factory,
	target tsgo.Identifier,
	source tsgo.Identifier,
	value tsgo.Identifier,
) tsgo.VariableStatement {
	condition := binary(
		factory,
		source,
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		factory.Identifier("undefined"),
	)
	selected := factory.ConditionalExpression(
		condition,
		factory.QuestionToken(),
		length(factory, value),
		factory.ColonToken(),
		factory.CallExpression(
			factory.Identifier("Number"),
			nil,
			nil,
			[]tsgo.Expression{source},
			tsgo.NodeFlagsNone,
		),
	)
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(target, nil, nil, selected),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func boundsCheck(
	factory tsgo.Factory,
	panicName string,
	condition tsgo.Expression,
	message string,
) tsgo.IfStatement {
	return factory.IfStatement(
		condition,
		factory.Block(
			[]tsgo.Statement{
				factory.ExpressionStatement(
					panicruntime.Call(
						factory,
						panicName,
						factory.StringLiteral(
							message,
							tsgo.TokenFlagsNone,
						),
					),
				),
			},
			true,
		),
		nil,
	)
}

func notSafeInteger(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.PrefixUnaryExpression {
	return factory.PrefixUnaryExpression(
		tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
		methodCall(factory, factory.Identifier("Number"), "isSafeInteger", []tsgo.Expression{value}),
	)
}

func length(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		value,
		nil,
		factory.Identifier("length"),
		tsgo.NodeFlagsNone,
	)
}

func methodCall(
	factory tsgo.Factory,
	receiver tsgo.Expression,
	name string,
	arguments []tsgo.Expression,
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

func numeric(factory tsgo.Factory, value string) tsgo.NumericLiteral {
	return factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func or(
	factory tsgo.Factory,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return binary(factory, left, tsgo.BinaryOperatorBarBarToken, right)
}

func lessThan(
	factory tsgo.Factory,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return binary(factory, left, tsgo.BinaryOperatorLessThanToken, right)
}

func greaterThan(
	factory tsgo.Factory,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return binary(factory, left, tsgo.BinaryOperatorGreaterThanToken, right)
}

func greaterThanOrEqual(
	factory tsgo.Factory,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return binary(factory, left, tsgo.BinaryOperatorGreaterThanEqualsToken, right)
}

func binary(
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

type BuildError struct {
	Symbol api.RuntimeSymbol
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("build string runtime symbol %d: unsupported symbol", e.Symbol)
}
