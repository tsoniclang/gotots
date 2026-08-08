package runtimecontract

import (
	"fmt"
	"slices"
)

type RuntimeModule uint8

const (
	RuntimeModuleInvalid          RuntimeModule = 0
	RuntimeModuleString           RuntimeModule = 1
	RuntimeModuleArray            RuntimeModule = 3
	RuntimeModuleSlice            RuntimeModule = 4
	RuntimeModuleMap              RuntimeModule = 5
	RuntimeModulePanic            RuntimeModule = 6
	RuntimeModuleInteger          RuntimeModule = 7
	RuntimeModuleFloat            RuntimeModule = 8
	RuntimeModuleComplex          RuntimeModule = 9
	RuntimeModuleConversion       RuntimeModule = 10
	RuntimeModuleInterface        RuntimeModule = 11
	RuntimeModuleInterfaceValue   RuntimeModule = 12
	RuntimeModulePanicNil         RuntimeModule = 13
	RuntimeModuleChannel          RuntimeModule = 14
	RuntimeModuleDenseIndex       RuntimeModule = 16
	RuntimeModuleUnsafe           RuntimeModule = 17
	RuntimeModuleStruct           RuntimeModule = 18
	RuntimeModuleStorage          RuntimeModule = 19
	RuntimeModuleDeferredRegistry RuntimeModule = 20
	RuntimeModuleScalar           RuntimeModule = 21
)

func runtimeContract(
	module RuntimeModule,
	outputPath string,
	exportedName string,
	typeUsable bool,
	dependencies ...RuntimeSymbol,
) RuntimeSymbolContract {
	return RuntimeSymbolContract{
		module:       module,
		outputPath:   outputPath,
		exportedName: exportedName,
		typeUsable:   typeUsable,
		dependencies: slices.Clone(dependencies),
	}
}

func unsafeRuntimeContract(
	symbol RuntimeSymbol,
) (RuntimeSymbolContract, bool) {
	var contract RuntimeSymbolContract
	switch symbol {
	case RuntimeUnsafeString:
		contract = runtimeContract(
			RuntimeModuleUnsafe,
			"runtime/unsafe.ts",
			"goUnsafeString",
			false,
			RuntimeSlice,
			RuntimePanic,
		)
	default:
		return RuntimeSymbolContract{}, false
	}
	return contract, true
}

func concurrencyRuntimeContract(
	symbol RuntimeSymbol,
) (RuntimeSymbolContract, error) {
	switch symbol {
	case RuntimeChannel:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoChannel",
			true,
			RuntimeReceiveChannel,
			RuntimeSendChannel,
			RuntimeSelectCase,
			RuntimePanic,
			RuntimeDenseIndex,
		), nil
	case RuntimeReceiveChannel:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoReceiveChannel",
			true,
			RuntimeSelectCase,
		), nil
	case RuntimeSendChannel:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoSendChannel",
			true,
			RuntimeSelectCase,
		), nil
	case RuntimeSelectCase:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoSelectCase",
			true,
		), nil
	case RuntimeSelect:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"goSelect",
			false,
			RuntimeSelectReady,
			RuntimeSelectAttempt,
		), nil
	case RuntimeScheduler:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"GoScheduler",
			true,
			RuntimePanic,
		), nil
	case RuntimeSelectReady:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"goSelectReady",
			false,
			RuntimeSelectAttempt,
		), nil
	case RuntimeSelectAttempt:
		return runtimeContract(
			RuntimeModuleChannel,
			"runtime/channel.ts",
			"goSelectAttempt",
			false,
			RuntimeSelectCase,
		), nil
	default:
		return RuntimeSymbolContract{}, &RuntimeSymbolError{Symbol: symbol}
	}
}

func complexOperationContract(
	exportedName string,
	dependencies ...RuntimeSymbol,
) (RuntimeSymbolContract, error) {
	return runtimeContract(
		RuntimeModuleComplex,
		"runtime/complex.ts",
		exportedName,
		false,
		dependencies...,
	), nil
}

func (c RuntimeSymbolContract) Module() RuntimeModule {
	return c.module
}

func (c RuntimeSymbolContract) OutputPath() string {
	return c.outputPath
}

func (c RuntimeSymbolContract) ExportedName() string {
	return c.exportedName
}

func (c RuntimeSymbolContract) Dependencies() []RuntimeSymbol {
	return slices.Clone(c.dependencies)
}

func (c RuntimeSymbolContract) TypeUsable() bool {
	return c.typeUsable
}

type RuntimeSymbolError struct {
	Symbol RuntimeSymbol
}

func (e *RuntimeSymbolError) Error() string {
	return fmt.Sprintf("runtime symbol %d is invalid", e.Symbol)
}

func interfaceRuntimeContract(
	symbol RuntimeSymbol,
) (RuntimeSymbolContract, bool) {
	var contract RuntimeSymbolContract
	switch symbol {
	case RuntimeInterfaceValue:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoInterfaceValue",
			true,
		)
	case RuntimeInterfaceNonNil:
		contract = runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"goInterfaceNonNil",
			false,
			RuntimeInterfaceValue,
			RuntimePanic,
		)
	case RuntimeInterfaceEqual:
		contract = runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"goInterfaceEqual",
			false,
			RuntimeInterfaceValue,
		)
	case RuntimeErrorMethodToken:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoErrorMethodToken",
			false,
		)
	case RuntimeRuntimeErrorToken:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoRuntimeErrorMethodToken",
			false,
		)
	case RuntimeBuiltinErrorType:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoError",
			true,
			RuntimeInterfaceValue,
			RuntimeErrorMethodToken,
			RuntimeAwaitable,
		)
	case RuntimeBuiltinErrorContract:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoError$contract",
			false,
			RuntimeErrorMethodToken,
		)
	case RuntimeBuiltinErrorGuard:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoError$is",
			false,
			RuntimeBuiltinErrorType,
			RuntimeBuiltinErrorContract,
		)
	case RuntimeErrorType:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoRuntimeError",
			true,
			RuntimeInterfaceValue,
			RuntimeErrorMethodToken,
			RuntimeRuntimeErrorToken,
			RuntimeAwaitable,
		)
	case RuntimeErrorContract:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoRuntimeError$contract",
			false,
			RuntimeErrorMethodToken,
			RuntimeRuntimeErrorToken,
		)
	case RuntimeErrorGuard:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoRuntimeError$is",
			false,
			RuntimeErrorType,
			RuntimeErrorContract,
		)
	case RuntimeInterfaceFormat:
		contract = runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"GoInterfaceFormat",
			false,
			RuntimePanic,
		)
	case RuntimeProviderInterfaceBridge:
		contract = runtimeContract(
			RuntimeModuleInterfaceValue,
			"runtime/interface-value.ts",
			"GoProviderInterfaceBridge",
			true,
			RuntimeInterfaceValue,
		)
	default:
		return RuntimeSymbolContract{}, false
	}
	return contract, true
}
