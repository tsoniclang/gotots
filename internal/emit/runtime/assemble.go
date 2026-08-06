package runtime

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimearray "github.com/tsoniclang/gotots/internal/emit/runtime/array"
	complexruntime "github.com/tsoniclang/gotots/internal/emit/runtime/complex"
	conversionruntime "github.com/tsoniclang/gotots/internal/emit/runtime/conversion"
	deferredregistryruntime "github.com/tsoniclang/gotots/internal/emit/runtime/deferredregistry"
	emptystructruntime "github.com/tsoniclang/gotots/internal/emit/runtime/emptystruct"
	floatruntime "github.com/tsoniclang/gotots/internal/emit/runtime/float"
	indexedstorage "github.com/tsoniclang/gotots/internal/emit/runtime/indexedstorage"
	integerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/integer"
	interfaceruntime "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue"
	storagefacetruntime "github.com/tsoniclang/gotots/internal/emit/runtime/storagefacet"
	stringruntime "github.com/tsoniclang/gotots/internal/emit/runtime/string"
	unsafecodecruntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafecodec"
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
	concurrency api.ConcurrencySemantics,
) ([]Definition, error) {
	return BuildWithFeatures(factory, module, symbols, nil, concurrency)
}

func BuildWithFeatures(
	factory tsgo.Factory,
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
	features []api.RuntimeFeature,
	concurrency api.ConcurrencySemantics,
) ([]Definition, error) {
	if module == api.RuntimeModuleInvalid {
		return nil, &AssemblyError{Reason: "runtime module is invalid"}
	}
	if len(symbols) == 0 {
		return nil, &AssemblyError{Reason: "runtime symbol set is empty"}
	}
	if !concurrency.Valid() {
		return nil, &AssemblyError{Reason: "runtime concurrency profile is invalid"}
	}
	seenFeatures := make(map[api.RuntimeFeature]struct{}, len(features))
	for _, feature := range features {
		featureModule, ok := api.RuntimeFeatureModule(feature)
		if !ok || featureModule != module {
			return nil, &AssemblyError{
				Module: module,
				Reason: "runtime feature has foreign module ownership",
			}
		}
		if _, duplicate := seenFeatures[feature]; duplicate {
			return nil, &AssemblyError{
				Module: module,
				Reason: "runtime feature is duplicated",
			}
		}
		seenFeatures[feature] = struct{}{}
	}
	if module == api.RuntimeModuleScalar {
		if len(symbols) != 1 || symbols[0] != api.RuntimeAwaitable {
			return nil, &AssemblyError{
				Module: module,
				Reason: "scalar runtime requires its one exact symbol",
			}
		}
		definition, err := NewDefinition(
			api.RuntimeAwaitable,
			awaitableType(factory),
		)
		if err != nil {
			return nil, err
		}
		return []Definition{definition}, nil
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
	if module == api.RuntimeModuleStorage {
		definitions := make([]Definition, 0, len(symbols))
		for _, symbol := range symbols {
			statement, err := storagefacetruntime.Build(factory, symbol)
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
	if module == api.RuntimeModuleDeferredRegistry {
		if len(symbols) != 1 || symbols[0] != api.RuntimeDeferredRegistry {
			return nil, &AssemblyError{
				Module: module,
				Reason: "deferred registry runtime requires its one exact symbol",
			}
		}
		interfaceValue, err := api.RuntimeContract(api.RuntimeInterfaceValue)
		if err != nil {
			return nil, err
		}
		contract, err := api.RuntimeContract(api.RuntimeDeferredRegistry)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(
			api.RuntimeDeferredRegistry,
			deferredregistryruntime.Build(
				factory,
				contract.ExportedName(),
				interfaceValue.ExportedName(),
			),
		)
		if err != nil {
			return nil, err
		}
		return []Definition{definition}, nil
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
		return buildPointer(
			factory,
			symbols,
			slices.Contains(features, api.RuntimePointerFieldPath),
		)
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
				concurrency,
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
	if module == api.RuntimeModuleStruct {
		if len(symbols) != 1 || symbols[0] != api.RuntimeEmptyStruct {
			return nil, &AssemblyError{
				Module: module,
				Reason: "struct runtime requires exactly RuntimeEmptyStruct",
			}
		}
		contract, err := api.RuntimeContract(api.RuntimeEmptyStruct)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(
			api.RuntimeEmptyStruct,
			emptystructruntime.Build(factory, contract.ExportedName()),
		)
		if err != nil {
			return nil, err
		}
		return []Definition{definition}, nil
	}
	if module == api.RuntimeModuleChannel {
		return buildChannel(factory, symbols)
	}
	if module == api.RuntimeModuleUnsafePointer {
		if len(symbols) != 2 ||
			symbols[0] != api.RuntimeUnsafeCodec ||
			symbols[1] != api.RuntimeUnsafePointer {
			return nil, &AssemblyError{
				Module: module,
				Reason: "unsafe-pointer runtime requires codec before pointer",
			}
		}
		codecContract, err := api.RuntimeContract(api.RuntimeUnsafeCodec)
		if err != nil {
			return nil, err
		}
		pointerContract, err := api.RuntimeContract(api.RuntimeUnsafePointer)
		if err != nil {
			return nil, err
		}
		goPointerContract, err := api.RuntimeContract(api.RuntimePointer)
		if err != nil {
			return nil, err
		}
		pointerMemoryContract, err := api.RuntimeContract(
			api.RuntimePointerUnsafeMemory,
		)
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
		codecDefinition, err := NewDefinition(
			api.RuntimeUnsafeCodec,
			unsafecodecruntime.Build(
				factory,
				codecContract.ExportedName(),
				panicContract.ExportedName(),
			),
		)
		if err != nil {
			return nil, err
		}
		pointerDefinition, err := NewDefinition(
			api.RuntimeUnsafePointer,
			unsafepointerruntime.Build(
				factory,
				pointerContract.ExportedName(),
				codecContract.ExportedName(),
				panicContract.ExportedName(),
				goPointerContract.ExportedName(),
				pointerMemoryContract.ExportedName(),
				denseIndexContract.ExportedName(),
			),
		)
		if err != nil {
			return nil, err
		}
		return []Definition{codecDefinition, pointerDefinition}, nil
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
	if module == api.RuntimeModulePanic ||
		module == api.RuntimeModulePanicNil {
		return buildPanic(factory, module, symbols)
	}
	return nil, &AssemblyError{
		Module: module,
		Reason: "runtime module owner is not installed",
	}
}
