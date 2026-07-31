package runtime

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildMap(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
) ([]Definition, error) {
	panicContract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		return nil, err
	}
	result := make([]Definition, 0, len(symbols))
	seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	for _, symbol := range symbols {
		if _, duplicate := seen[symbol]; duplicate {
			return nil, &AssemblyError{
				Module: api.RuntimeModuleMap,
				Symbol: symbol,
				Reason: "map runtime symbol is duplicated",
			}
		}
		seen[symbol] = struct{}{}
		statement, err := mapruntime.Build(
			factory,
			symbol,
			panicContract.ExportedName(),
		)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(symbol, statement)
		if err != nil {
			return nil, err
		}
		result = append(result, definition)
	}
	return result, nil
}

type AssemblyError struct {
	Module api.RuntimeModule
	Symbol api.RuntimeSymbol
	Reason string
}

func (e *AssemblyError) Error() string {
	return fmt.Sprintf(
		"assemble runtime module %d symbol %d: %s",
		e.Module,
		e.Symbol,
		e.Reason,
	)
}
