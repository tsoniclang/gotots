package runtime

import (
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
		case api.RuntimeSliceMakeWith:
			capabilities.AggregateMake = true
		case api.RuntimeSliceNilWith:
			capabilities.AggregateNil = true
		case api.RuntimeSliceLiteralWith:
			capabilities.AggregateLiteral = true
		case api.RuntimeSliceAppendWith:
			capabilities.AggregateAppend = true
		case api.RuntimeSliceCopyWith:
			capabilities.AggregateCopy = true
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
	class, err := NewDefinition(
		api.RuntimeSlice,
		runtimeslice.BuildWithCapabilities(
			factory,
			sliceContract.ExportedName(),
			panicContract.ExportedName(),
			capabilities,
		),
	)
	if err != nil {
		return nil, err
	}
	definitions := []Definition{class}
	for _, symbol := range symbols[1:] {
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
	if symbol != api.RuntimeSliceAddress {
		return runtimeslice.BuildAggregateOperation(factory, symbol)
	}
	addressContract, err := api.RuntimeContract(api.RuntimeSliceAddress)
	if err != nil {
		return nil, err
	}
	pointerContract, err := api.RuntimeContract(api.RuntimePointer)
	if err != nil {
		return nil, err
	}
	return runtimeslice.BuildAddress(
		factory,
		addressContract.ExportedName(),
		sliceName,
		pointerContract.ExportedName(),
	), nil
}
