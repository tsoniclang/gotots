package float

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// Build emits the float runtime module. goFloat32 rounds a number to its nearest
// binary32 through Math.fround. Rounding a correctly-rounded binary64 result to
// binary32 is exact for the basic arithmetic operations — binary64 carries at
// least 2p+2 bits for p = 24 — so goFloat32 applied to a binary64 operation
// yields the same value Go's float32 operation would.
func Build(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
) ([]tsgo.Statement, error) {
	if len(symbols) == 0 {
		return nil, &BuildError{Reason: "runtime symbol set is empty"}
	}
	seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	statements := make([]tsgo.Statement, 0, len(symbols))
	for _, symbol := range symbols {
		if _, duplicate := seen[symbol]; duplicate {
			return nil, &BuildError{Symbol: symbol, Reason: "runtime symbol is duplicated"}
		}
		seen[symbol] = struct{}{}
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		if contract.Module() != api.RuntimeModuleFloat {
			return nil, &BuildError{Symbol: symbol, Reason: "runtime symbol belongs to another module"}
		}
		if symbol != api.RuntimeFloat32Round {
			return nil, &BuildError{Symbol: symbol, Reason: "runtime float operation is not installed"}
		}
		statements = append(statements, round32(factory, contract.ExportedName()))
	}
	return statements, nil
}

func round32(factory tsgo.Factory, name string) tsgo.FunctionDeclaration {
	value := factory.Identifier("value")
	numberType := factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		[]tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(nil, nil, value, nil, numberType, nil),
		},
		numberType,
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(factory.CallExpression(
				factory.PropertyAccessExpression(
					factory.Identifier("Math"),
					nil,
					factory.Identifier("fround"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{value},
				tsgo.NodeFlagsNone,
			)),
		}, true),
	)
}

type BuildError struct {
	Symbol api.RuntimeSymbol
	Reason string
}

func (e *BuildError) Error() string {
	return fmt.Sprintf("build float runtime symbol %d: %s", e.Symbol, e.Reason)
}
