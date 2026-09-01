package sourcefact

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func FactSymbol(symbol api.RuntimeSymbol) (api.RuntimeSymbol, bool) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil || contract.Module() == api.RuntimeModuleSourceFact {
		return api.RuntimeInvalid, false
	}
	switch contract.Module() {
	case api.RuntimeModuleStorage:
		return api.RuntimeSourceStorageFact, true
	case api.RuntimeModuleArray,
		api.RuntimeModuleSlice,
		api.RuntimeModuleMap,
		api.RuntimeModuleChannel,
		api.RuntimeModuleStruct:
		if contract.TypeUsable() {
			return api.RuntimeSourceAggregateFact, true
		}
	case api.RuntimeModuleInterface,
		api.RuntimeModuleInterfaceValue:
		if contract.TypeUsable() {
			return api.RuntimeSourceInterfaceFact, true
		}
	case api.RuntimeModuleComplex:
		if contract.TypeUsable() {
			return api.RuntimeSourceBasicFact, true
		}
	}
	return api.RuntimeSourceOperationFact, true
}

func FactSymbols(symbol api.RuntimeSymbol) []api.RuntimeSymbol {
	primary, ok := FactSymbol(symbol)
	if !ok {
		return nil
	}
	result := []api.RuntimeSymbol{primary}
	contract, err := api.RuntimeContract(symbol)
	if err == nil && contract.TypeUsable() &&
		primary != api.RuntimeSourceOperationFact {
		result = append(result, api.RuntimeSourceOperationFact)
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func Identity(symbol api.RuntimeSymbol) (string, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return "", err
	}
	return contract.OutputPath() + "#" + contract.ExportedName(), nil
}
