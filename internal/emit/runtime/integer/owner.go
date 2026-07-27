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
		operator, ok := operator(symbol)
		if !ok {
			return nil, &BuildError{
				Symbol: symbol,
				Reason: "runtime integer operation is not installed",
			}
		}
		statements = append(
			statements,
			operation(
				factory,
				contract.ExportedName(),
				panicName,
				operator,
			),
		)
	}
	return statements, nil
}

func operation(
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
