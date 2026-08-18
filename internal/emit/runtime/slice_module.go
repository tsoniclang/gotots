package runtime

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildSlice(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
) ([]Definition, error) {
	if symbols[0] != api.RuntimeSlice {
		return nil, &AssemblyError{
			Module: api.RuntimeModuleSlice,
			Reason: "slice runtime requires RuntimeSlice first",
		}
	}
	capabilities := runtimeslice.Capabilities{}
	seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	for _, symbol := range symbols {
		if _, duplicate := seen[symbol]; duplicate {
			return nil, &AssemblyError{
				Module: api.RuntimeModuleSlice,
				Symbol: symbol,
				Reason: "slice runtime symbol is duplicated",
			}
		}
		seen[symbol] = struct{}{}
		switch symbol {
		case api.RuntimeSlice:
		case api.RuntimeSliceAddress:
			capabilities.Address = true
		case api.RuntimeSliceArrayPointer:
			capabilities.ArrayPointer = true
		case api.RuntimeArraySlice:
			capabilities.ArrayView = true
		case api.RuntimeSliceStorage:
			capabilities.Storage = true
		case api.RuntimeSliceProjection:
		case api.RuntimeSliceAppendSlice:
			capabilities.AppendSlice = true
		case api.RuntimeSliceClear:
			capabilities.Clear = true
		case api.RuntimeSliceRegion:
			capabilities.Region = true
		default:
			return nil, &api.RuntimeSymbolError{Symbol: symbol}
		}
	}
	sliceContract, err := api.RuntimeContract(api.RuntimeSlice)
	if err != nil {
		return nil, err
	}
	panicContract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		return nil, err
	}
	pointerName := ""
	addressName := ""
	pointerProjectName := ""
	if capabilities.Address {
		pointerContract, pointerErr := tsoniccore.Resolve(tsoniccore.SymbolPointer)
		if pointerErr != nil {
			return nil, pointerErr
		}
		pointerName = pointerContract.Export()
		addressContract, addressErr := tsoniccore.Resolve(
			tsoniccore.SymbolAddressOf,
		)
		if addressErr != nil {
			return nil, addressErr
		}
		addressName = addressContract.Export()
		pointerProjectContract, pointerProjectErr := tsoniccore.Resolve(
			tsoniccore.SymbolProjectPointer,
		)
		if pointerProjectErr != nil {
			return nil, pointerProjectErr
		}
		pointerProjectName = pointerProjectContract.Export()
	}
	class, err := NewDefinition(
		api.RuntimeSlice,
		runtimeslice.BuildWithCapabilities(
			factory,
			sliceContract.ExportedName(),
			panicContract.ExportedName(),
			pointerName,
			addressName,
			capabilities,
		),
	)
	if err != nil {
		return nil, err
	}
	definitions := []Definition{class}
	for _, symbol := range symbols[1:] {
		if symbol == api.RuntimeSliceProjection {
			projectionContract, err := api.RuntimeContract(symbol)
			if err != nil {
				return nil, err
			}
			definition, err := NewDefinition(
				symbol,
				runtimeslice.BuildProjection(
					factory,
					projectionContract.ExportedName(),
					sliceContract.ExportedName(),
					panicContract.ExportedName(),
					pointerName,
					pointerProjectName,
					capabilities,
				),
			)
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
			continue
		}
		statement, err := buildSliceOperation(
			factory,
			symbol,
			sliceContract.ExportedName(),
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

func buildSliceOperation(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	sliceName string,
) (tsgo.Statement, error) {
	if symbol != api.RuntimeSliceAddress &&
		symbol != api.RuntimeSliceArrayPointer &&
		symbol != api.RuntimeArraySlice &&
		symbol != api.RuntimeSliceRegion {
		return runtimeslice.BuildOperation(factory, symbol)
	}
	addressContract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	pointerContract, err := tsoniccore.Resolve(tsoniccore.SymbolPointer)
	if err != nil {
		return nil, err
	}
	if symbol == api.RuntimeSliceArrayPointer {
		addressOfContract, err := tsoniccore.Resolve(
			tsoniccore.SymbolAddressOf,
		)
		if err != nil {
			return nil, err
		}
		projectContract, err := tsoniccore.Resolve(
			tsoniccore.SymbolProjectPointer,
		)
		if err != nil {
			return nil, err
		}
		arrayContract, err := api.RuntimeContract(api.RuntimeArray)
		if err != nil {
			return nil, err
		}
		arrayViewContract, err := api.RuntimeContract(api.RuntimeArrayView)
		if err != nil {
			return nil, err
		}
		return runtimeslice.BuildArrayPointer(
			factory,
			addressContract.ExportedName(),
			sliceName,
			pointerContract.Export(),
			addressOfContract.Export(),
			projectContract.Export(),
			arrayContract.ExportedName(),
			arrayViewContract.ExportedName(),
		), nil
	}
	if symbol == api.RuntimeArraySlice {
		arrayContract, err := api.RuntimeContract(api.RuntimeArray)
		if err != nil {
			return nil, err
		}
		locationContract, err := api.RuntimeContract(api.RuntimeArrayLocation)
		if err != nil {
			return nil, err
		}
		return runtimeslice.BuildArraySlice(
			factory,
			addressContract.ExportedName(),
			sliceName,
			arrayContract.ExportedName(),
			locationContract.ExportedName(),
		), nil
	}
	if symbol == api.RuntimeSliceRegion {
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		return runtimeslice.BuildRegion(
			factory,
			addressContract.ExportedName(),
			sliceName,
			panicContract.ExportedName(),
		), nil
	}
	return runtimeslice.BuildAddress(
		factory,
		addressContract.ExportedName(),
		sliceName,
		pointerContract.Export(),
	), nil
}
