package runtime

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func orderModuleSymbols(
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
) ([]api.RuntimeSymbol, error) {
	selected := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	for _, symbol := range symbols {
		selected[symbol] = struct{}{}
	}
	ordered := make([]api.RuntimeSymbol, 0, len(symbols))
	state := make(map[api.RuntimeSymbol]uint8, len(symbols))
	var visit func(api.RuntimeSymbol) error
	visit = func(symbol api.RuntimeSymbol) error {
		switch state[symbol] {
		case 1:
			return &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "runtime module dependency graph contains a cycle",
			}
		case 2:
			return nil
		}
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return err
		}
		if contract.Module() != module {
			return &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "runtime symbol belongs to another module",
			}
		}
		state[symbol] = 1
		dependencies := contract.Dependencies()
		slices.Sort(dependencies)
		for _, dependency := range dependencies {
			dependencyContract, err := api.RuntimeContract(dependency)
			if err != nil {
				return err
			}
			if dependencyContract.Module() != module {
				continue
			}
			if _, ok := selected[dependency]; !ok {
				return &AssemblyError{
					Module: module,
					Symbol: symbol,
					Reason: "same-module runtime dependency is absent",
				}
			}
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[symbol] = 2
		ordered = append(ordered, symbol)
		return nil
	}
	roots := slices.Clone(symbols)
	slices.Sort(roots)
	for _, symbol := range roots {
		if err := visit(symbol); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}
