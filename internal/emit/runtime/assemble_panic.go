package runtime

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	panicnilruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panicnil"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildPanic(
	factory tsgo.Factory,
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
) ([]Definition, error) {
	if module == api.RuntimeModulePanic {
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		valueContract, err := api.RuntimeContract(api.RuntimeInterfaceValue)
		if err != nil {
			return nil, err
		}
		runtimeValueContract, err := api.RuntimeContract(api.RuntimePanicValue)
		if err != nil {
			return nil, err
		}
		recoveryContract, err := api.RuntimeContract(api.RuntimeRecovery)
		if err != nil {
			return nil, err
		}
		deferPopContract, err := api.RuntimeContract(api.RuntimeDeferPop)
		if err != nil {
			return nil, err
		}
		errorTokenContract, err := api.RuntimeContract(
			api.RuntimeErrorMethodToken,
		)
		if err != nil {
			return nil, err
		}
		runtimeErrorTokenContract, err := api.RuntimeContract(
			api.RuntimeRuntimeErrorToken,
		)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
		for _, symbol := range symbols {
			if _, duplicate := seen[symbol]; duplicate {
				return nil, &AssemblyError{
					Module: module,
					Symbol: symbol,
					Reason: "panic runtime symbol is duplicated",
				}
			}
			seen[symbol] = struct{}{}
			statement, err := panicruntime.Build(
				factory,
				symbol,
				panicContract.ExportedName(),
				valueContract.ExportedName(),
				runtimeValueContract.ExportedName(),
				recoveryContract.ExportedName(),
				deferPopContract.ExportedName(),
				errorTokenContract.ExportedName(),
				runtimeErrorTokenContract.ExportedName(),
			)
			if err != nil {
				return nil, err
			}
			definition, err := NewDefinition(symbol, statement)
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModulePanicNil {
		errorContract, err := api.RuntimeContract(api.RuntimePanicNilError)
		if err != nil {
			return nil, err
		}
		valueContract, err := api.RuntimeContract(api.RuntimePanicNilValue)
		if err != nil {
			return nil, err
		}
		runtimeValueContract, err := api.RuntimeContract(api.RuntimePanicValue)
		if err != nil {
			return nil, err
		}
		interfaceValueContract, err := api.RuntimeContract(api.RuntimeInterfaceValue)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
		for _, symbol := range symbols {
			if _, duplicate := seen[symbol]; duplicate {
				return nil, &AssemblyError{
					Module: module,
					Symbol: symbol,
					Reason: "panic-nil runtime symbol is duplicated",
				}
			}
			seen[symbol] = struct{}{}
			statement, err := panicnilruntime.Build(
				factory,
				symbol,
				errorContract.ExportedName(),
				valueContract.ExportedName(),
				runtimeValueContract.ExportedName(),
				interfaceValueContract.ExportedName(),
			)
			if err != nil {
				return nil, err
			}
			definition, err := NewDefinition(symbol, statement)
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	return nil, &AssemblyError{
		Module: module,
		Reason: "panic runtime module is invalid",
	}
}
