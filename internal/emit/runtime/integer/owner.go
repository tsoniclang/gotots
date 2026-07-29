package integer

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
		return nil, &BuildError{Reason: "runtime symbol set is empty"}
	}
	seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	statements := make([]tsgo.Statement, 0, len(symbols))
	for _, symbol := range symbols {
		if _, duplicate := seen[symbol]; duplicate {
			return nil, &BuildError{
				Symbol: symbol,
				Reason: "runtime symbol is duplicated",
			}
		}
		seen[symbol] = struct{}{}
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		if contract.Module() != api.RuntimeModuleInteger {
			return nil, &BuildError{
				Symbol: symbol,
				Reason: "runtime symbol belongs to another module",
			}
		}
		switch symbol {
		case api.RuntimeIntegerDivide, api.RuntimeIntegerRemainder:
			operator, _ := operator(symbol)
			statements = append(statements, bigintDivisionOperation(
				factory,
				contract.ExportedName(),
				panicName,
				operator,
			))
		case api.RuntimeNumberIntDivide:
			statements = append(statements, numberDivisionOperation(
				factory,
				contract.ExportedName(),
				panicName,
			))
		case api.RuntimeNumberIntRemainder:
			statements = append(statements, numberRemainderOperation(
				factory,
				contract.ExportedName(),
				panicName,
			))
		case api.RuntimeIntegerMax, api.RuntimeIntegerMin:
			operator, _ := ordering(symbol)
			statements = append(statements, orderedOperation(
				factory,
				contract.ExportedName(),
				operator,
			))
		default:
			return nil, &BuildError{
				Symbol: symbol,
				Reason: "runtime integer operation is not installed",
			}
		}
	}
	return statements, nil
}

func bigintDivisionOperation(
	factory tsgo.Factory,
	name string,
	panicName string,
	operator tsgo.BinaryOperator,
) tsgo.FunctionDeclaration {
	left := factory.Identifier("left")
	right := factory.Identifier("right")
	bigintType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindBigIntKeyword,
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, left, bigintType),
			parameter(factory, right, bigintType),
		},
		bigintType,
		factory.Block([]tsgo.Statement{
			factory.IfStatement(
				factory.BinaryExpression(
					nil,
					right,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					factory.BigIntLiteral("0n", tsgo.TokenFlagsNone),
				),
				factory.Block([]tsgo.Statement{
					factory.ExpressionStatement(panicruntime.Call(
						factory,
						panicName,
						factory.StringLiteral(
							"integer divide by zero",
							tsgo.TokenFlagsNone,
						),
					)),
				}, true),
				nil,
			),
			factory.ReturnStatement(factory.BinaryExpression(
				nil,
				left,
				nil,
				factory.BinaryOperatorToken(operator),
				right,
			)),
		}, true),
	)
}

func numberDivisionOperation(
	factory tsgo.Factory,
	name string,
	panicName string,
) tsgo.FunctionDeclaration {
	left, right, numberType, guard := numberOperationParts(
		factory,
		panicName,
	)
	quotient := factory.BinaryExpression(
		nil,
		left,
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorSlashToken),
		right,
	)
	return numericFunction(
		factory,
		name,
		left,
		right,
		numberType,
		guard,
		factory.CallExpression(
			factory.PropertyAccessExpression(
				factory.Identifier("Math"),
				nil,
				factory.Identifier("trunc"),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{quotient},
			tsgo.NodeFlagsNone,
		),
	)
}

func numberRemainderOperation(
	factory tsgo.Factory,
	name string,
	panicName string,
) tsgo.FunctionDeclaration {
	left, right, numberType, guard := numberOperationParts(
		factory,
		panicName,
	)
	return numericFunction(
		factory,
		name,
		left,
		right,
		numberType,
		guard,
		factory.BinaryExpression(
			nil,
			left,
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorPercentToken),
			right,
		),
	)
}

func numberOperationParts(
	factory tsgo.Factory,
	panicName string,
) (
	tsgo.Identifier,
	tsgo.Identifier,
	tsgo.TypeNode,
	tsgo.IfStatement,
) {
	left := factory.Identifier("left")
	right := factory.Identifier("right")
	numberType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
	guard := factory.IfStatement(
		factory.BinaryExpression(
			nil,
			right,
			nil,
			factory.BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
		),
		factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(panicruntime.Call(
				factory,
				panicName,
				factory.StringLiteral(
					"integer divide by zero",
					tsgo.TokenFlagsNone,
				),
			)),
		}, true),
		nil,
	)
	return left, right, numberType, guard
}

func numericFunction(
	factory tsgo.Factory,
	name string,
	left tsgo.Identifier,
	right tsgo.Identifier,
	numberType tsgo.TypeNode,
	guard tsgo.IfStatement,
	result tsgo.Expression,
) tsgo.FunctionDeclaration {
	resultName := factory.Identifier("result")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, left, numberType),
			parameter(factory, right, numberType),
		},
		numberType,
		factory.Block([]tsgo.Statement{
			guard,
			factory.VariableStatement(
				nil,
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						factory.VariableDeclaration(
							resultName,
							nil,
							numberType,
							result,
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
			factory.ReturnStatement(factory.ConditionalExpression(
				factory.BinaryExpression(
					nil,
					resultName,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				),
				factory.QuestionToken(),
				factory.NumericLiteral("0", tsgo.TokenFlagsNone),
				factory.ColonToken(),
				resultName,
			)),
		}, true),
	)
}

func orderedOperation(
	factory tsgo.Factory,
	name string,
	operator tsgo.BinaryOperator,
) tsgo.FunctionDeclaration {
	left := factory.Identifier("left")
	right := factory.Identifier("right")
	bigintType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindBigIntKeyword,
	)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, left, bigintType),
			parameter(factory, right, bigintType),
		},
		bigintType,
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(factory.ConditionalExpression(
				factory.BinaryExpression(
					nil,
					left,
					nil,
					factory.BinaryOperatorToken(operator),
					right,
				),
				factory.QuestionToken(),
				left,
				factory.ColonToken(),
				right,
			)),
		}, true),
	)
}

func parameter(
	factory tsgo.Factory,
	name tsgo.Identifier,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		name,
		nil,
		targetType,
		nil,
	)
}

func operator(
	symbol api.RuntimeSymbol,
) (tsgo.BinaryOperator, bool) {
	switch symbol {
	case api.RuntimeIntegerDivide:
		return tsgo.BinaryOperatorSlashToken, true
	case api.RuntimeIntegerRemainder:
		return tsgo.BinaryOperatorPercentToken, true
	default:
		return 0, false
	}
}

func ordering(
	symbol api.RuntimeSymbol,
) (tsgo.BinaryOperator, bool) {
	switch symbol {
	case api.RuntimeIntegerMax:
		return tsgo.BinaryOperatorGreaterThanEqualsToken, true
	case api.RuntimeIntegerMin:
		return tsgo.BinaryOperatorLessThanEqualsToken, true
	default:
		return 0, false
	}
}

type BuildError struct {
	Symbol api.RuntimeSymbol
	Reason string
}

func (e *BuildError) Error() string {
	return fmt.Sprintf(
		"build integer runtime symbol %d: %s",
		e.Symbol,
		e.Reason,
	)
}
