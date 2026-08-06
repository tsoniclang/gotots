package runtime

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildPointer(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
	fieldPath bool,
) ([]Definition, error) {
	if len(symbols) == 0 || symbols[0] != api.RuntimePointer {
		return nil, &AssemblyError{
			Module: api.RuntimeModulePointer,
			Reason: "pointer runtime requires exactly RuntimePointer",
		}
	}
	seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	for _, symbol := range symbols {
		if _, duplicate := seen[symbol]; duplicate {
			return nil, &AssemblyError{
				Module: api.RuntimeModulePointer,
				Symbol: symbol,
				Reason: "pointer runtime symbol is duplicated",
			}
		}
		seen[symbol] = struct{}{}
		if symbol != api.RuntimePointer &&
			symbol != api.RuntimePointerHash &&
			symbol != api.RuntimePointerRegion &&
			symbol != api.RuntimePointerUnsafeMemory &&
			symbol != api.RuntimePointerProjection {
			return nil, &api.RuntimeSymbolError{Symbol: symbol}
		}
	}
	contract, err := api.RuntimeContract(api.RuntimePointer)
	if err != nil {
		return nil, err
	}
	panicContract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		return nil, err
	}
	denseIndexContract, err := api.RuntimeContract(api.RuntimeDenseIndex)
	if err != nil {
		return nil, err
	}
	definition, err := NewDefinition(
		api.RuntimePointer,
		pointerruntime.BuildWithCapabilities(
			factory,
			contract.ExportedName(),
			panicContract.ExportedName(),
			denseIndexContract.ExportedName(),
			pointerruntime.Capabilities{
				FieldPath: fieldPath,
				Projection: slices.Contains(
					symbols,
					api.RuntimePointerProjection,
				),
				Region: slices.Contains(symbols, api.RuntimePointerRegion) ||
					slices.Contains(symbols, api.RuntimePointerUnsafeMemory),
				UnsafeMemory: slices.Contains(
					symbols,
					api.RuntimePointerUnsafeMemory,
				),
			},
		),
	)
	if err != nil {
		return nil, err
	}
	definitions := []Definition{definition}
	for _, symbol := range symbols[1:] {
		selected, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		statement, err := buildPointerOperation(
			factory,
			symbol,
			selected.ExportedName(),
			contract.ExportedName(),
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

func buildPointerOperation(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	functionName string,
	pointerName string,
) (tsgo.Statement, error) {
	switch symbol {
	case api.RuntimePointerHash:
		mapHashContract, err := api.RuntimeContract(api.RuntimeMapHash)
		if err != nil {
			return nil, err
		}
		return pointerruntime.Hash(
			factory,
			functionName,
			pointerName,
			mapHashContract.ExportedName(),
		), nil
	case api.RuntimePointerRegion:
		return pointerruntime.Region(factory, functionName, pointerName), nil
	case api.RuntimePointerUnsafeMemory:
		return pointerruntime.UnsafeMemory(factory, functionName, pointerName), nil
	case api.RuntimePointerProjection:
		return pointerruntime.Project(factory, functionName, pointerName), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}
