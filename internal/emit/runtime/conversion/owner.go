package conversion

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
) ([]tsgo.Statement, error) {
	if len(symbols) != 1 || symbols[0] != api.RuntimeNumberToBigInt {
		return nil, &BuildError{
			Reason: "conversion runtime requires exactly RuntimeNumberToBigInt",
		}
	}
	contract, err := api.RuntimeContract(api.RuntimeNumberToBigInt)
	if err != nil {
		return nil, err
	}
	value := factory.Identifier("value")
	finite := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("Number"),
			nil,
			factory.Identifier("isFinite"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
	truncated := factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("Math"),
			nil,
			factory.Identifier("trunc"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
	selected := factory.ConditionalExpression(
		finite,
		factory.QuestionToken(),
		truncated,
		factory.ColonToken(),
		factory.NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	target := factory.CallExpression(
		factory.Identifier("BigInt"),
		nil,
		nil,
		[]tsgo.Expression{selected},
		tsgo.NodeFlagsNone,
	)
	return []tsgo.Statement{
		factory.FunctionDeclaration(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			nil,
			factory.Identifier(contract.ExportedName()),
			nil,
			[]tsgo.ParameterDeclaration{
				factory.ParameterDeclaration(
					nil,
					nil,
					value,
					nil,
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindNumberKeyword,
					),
					nil,
				),
			},
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword),
			factory.Block(
				[]tsgo.Statement{factory.ReturnStatement(target)},
				true,
			),
		),
	}, nil
}

type BuildError struct {
	Reason string
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("build conversion runtime: %s", e.Reason)
}
