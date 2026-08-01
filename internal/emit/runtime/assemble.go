package runtime

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimearray "github.com/tsoniclang/gotots/internal/emit/runtime/array"
	complexruntime "github.com/tsoniclang/gotots/internal/emit/runtime/complex"
	conversionruntime "github.com/tsoniclang/gotots/internal/emit/runtime/conversion"
	floatruntime "github.com/tsoniclang/gotots/internal/emit/runtime/float"
	indexedstorage "github.com/tsoniclang/gotots/internal/emit/runtime/indexedstorage"
	integerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/integer"
	interfaceruntime "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	panicnilruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panicnil"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	stringruntime "github.com/tsoniclang/gotots/internal/emit/runtime/string"
	unsaferuntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafeoperation"
	unsafepointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafepointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Definition struct {
	symbol    api.RuntimeSymbol
	statement tsgo.Statement
}

func NewDefinition(
	symbol api.RuntimeSymbol,
	statement tsgo.Statement,
) (Definition, error) {
	if _, err := api.RuntimeContract(symbol); err != nil {
		return Definition{}, err
	}
	if statement == nil {
		return Definition{}, &AssemblyError{
			Symbol: symbol,
			Reason: "target statement is nil",
		}
	}
	return Definition{symbol: symbol, statement: statement}, nil
}

func (d Definition) Symbol() api.RuntimeSymbol {
	return d.symbol
}

func (d Definition) Statement() tsgo.Statement {
	return d.statement
}

func Build(
	factory tsgo.Factory,
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
) ([]Definition, error) {
	if module == api.RuntimeModuleInvalid {
		return nil, &AssemblyError{Reason: "runtime module is invalid"}
	}
	if len(symbols) == 0 {
		return nil, &AssemblyError{Reason: "runtime symbol set is empty"}
	}
	if module == api.RuntimeModuleString {
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		statements, err := stringruntime.Build(
			factory,
			symbols,
			panicContract.ExportedName(),
		)
		if err != nil {
			return nil, err
		}
		if len(statements) != len(symbols) {
			return nil, &AssemblyError{
				Module: module,
				Reason: "string runtime returned a non-exact definition set",
			}
		}
		definitions := make([]Definition, 0, len(symbols))
		for index, symbol := range symbols {
			definition, err := NewDefinition(symbol, statements[index])
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModuleDenseIndex {
		if len(symbols) != 1 || symbols[0] != api.RuntimeDenseIndex {
			return nil, &AssemblyError{
				Module: module,
				Reason: "dense-index runtime requires exactly RuntimeDenseIndex",
			}
		}
		contract, err := api.RuntimeContract(api.RuntimeDenseIndex)
		if err != nil {
			return nil, err
		}
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(
			api.RuntimeDenseIndex,
			indexedstorage.Build(
				factory,
				contract.ExportedName(),
				panicContract.ExportedName(),
			),
		)
		if err != nil {
			return nil, err
		}
		return []Definition{definition}, nil
	}
	if module == api.RuntimeModulePointer {
		if len(symbols) == 0 || symbols[0] != api.RuntimePointer {
			return nil, &AssemblyError{
				Module: module,
				Reason: "pointer runtime requires exactly RuntimePointer",
			}
		}
		seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
		for _, symbol := range symbols {
			if _, duplicate := seen[symbol]; duplicate {
				return nil, &AssemblyError{
					Module: module,
					Symbol: symbol,
					Reason: "pointer runtime symbol is duplicated",
				}
			}
			seen[symbol] = struct{}{}
			if symbol != api.RuntimePointer &&
				symbol != api.RuntimePointerHash &&
				symbol != api.RuntimePointerRegion {
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
					Region: slices.Contains(symbols, api.RuntimePointerRegion),
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
			var statement tsgo.Statement
			switch symbol {
			case api.RuntimePointerHash:
				mapHashContract, err := api.RuntimeContract(api.RuntimeMapHash)
				if err != nil {
					return nil, err
				}
				statement = pointerruntime.Hash(
					factory,
					selected.ExportedName(),
					contract.ExportedName(),
					mapHashContract.ExportedName(),
				)
			case api.RuntimePointerRegion:
				statement = pointerruntime.Region(
					factory,
					selected.ExportedName(),
					contract.ExportedName(),
				)
			}
			definition, err := NewDefinition(symbol, statement)
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModuleArray {
		if symbols[0] != api.RuntimeArray {
			return nil, &AssemblyError{
				Module: module,
				Reason: "array runtime requires RuntimeArray first",
			}
		}
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		denseIndexContract, err := api.RuntimeContract(api.RuntimeDenseIndex)
		if err != nil {
			return nil, err
		}
		statement, err := runtimearray.BuildWithCapabilities(
			factory,
			panicContract.ExportedName(),
			denseIndexContract.ExportedName(),
			runtimearray.Capabilities{
				Allocate: slices.Contains(
					symbols,
					api.RuntimeArrayAllocate,
				),
				View: slices.Contains(
					symbols,
					api.RuntimeArrayView,
				),
				Location: slices.Contains(
					symbols,
					api.RuntimeArrayLocation,
				),
			},
		)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(
			api.RuntimeArray,
			statement,
		)
		if err != nil {
			return nil, err
		}
		definitions := []Definition{definition}
		for _, symbol := range symbols[1:] {
			statement, err := runtimearray.BuildAggregateOperation(
				factory,
				symbol,
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
	if module == api.RuntimeModuleSlice {
		return buildSlice(factory, symbols)
	}
	if module == api.RuntimeModuleMap {
		return buildMap(factory, symbols)
	}
	if module == api.RuntimeModuleInteger {
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		statements, err := integerruntime.Build(
			factory,
			symbols,
			panicContract.ExportedName(),
		)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		for index, symbol := range symbols {
			definition, err := NewDefinition(symbol, statements[index])
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModuleFloat {
		statements, err := floatruntime.Build(factory, symbols)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		for index, symbol := range symbols {
			definition, err := NewDefinition(symbol, statements[index])
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModuleComplex {
		statements, err := complexruntime.Build(factory, symbols)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		for index, symbol := range symbols {
			definition, err := NewDefinition(symbol, statements[index])
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModuleConversion {
		statements, err := conversionruntime.Build(factory, symbols)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		for index, symbol := range symbols {
			definition, err := NewDefinition(symbol, statements[index])
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModuleInterface {
		valueContract, err := api.RuntimeContract(api.RuntimeInterfaceValue)
		if err != nil {
			return nil, err
		}
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		for _, symbol := range symbols {
			statement, err := interfaceruntime.Build(
				factory,
				symbol,
				valueContract.ExportedName(),
				panicContract.ExportedName(),
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
	if module == api.RuntimeModuleInterfaceValue {
		contract, err := api.RuntimeContract(api.RuntimeInterfaceValue)
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
					Reason: "interface-value runtime symbol is duplicated",
				}
			}
			seen[symbol] = struct{}{}
			statement, err := interfaceruntime.BuildValue(
				factory,
				symbol,
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
	if module == api.RuntimeModuleChannel {
		return buildChannel(factory, symbols)
	}
	if module == api.RuntimeModuleUnsafePointer {
		if len(symbols) != 1 || symbols[0] != api.RuntimeUnsafePointer {
			return nil, &AssemblyError{
				Module: module,
				Reason: "unsafe-pointer runtime requires exactly RuntimeUnsafePointer",
			}
		}
		contract, err := api.RuntimeContract(api.RuntimeUnsafePointer)
		if err != nil {
			return nil, err
		}
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		statement := unsafepointerruntime.Build(
			factory,
			contract.ExportedName(),
			panicContract.ExportedName(),
		)
		definition, err := NewDefinition(
			api.RuntimeUnsafePointer,
			statement,
		)
		if err != nil {
			return nil, err
		}
		return []Definition{definition}, nil
	}
	if module == api.RuntimeModuleUnsafe {
		definitions := make([]Definition, 0, len(symbols))
		seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
		for _, symbol := range symbols {
			if _, duplicate := seen[symbol]; duplicate {
				return nil, &AssemblyError{
					Module: module,
					Symbol: symbol,
					Reason: "unsafe runtime symbol is duplicated",
				}
			}
			seen[symbol] = struct{}{}
			statement, err := unsaferuntime.Build(factory, symbol)
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
		Reason: "runtime module owner is not installed",
	}
}
