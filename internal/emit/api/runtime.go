package api

import (
	"fmt"
	"slices"
)

type RuntimeModule uint8

const (
	RuntimeModuleInvalid RuntimeModule = iota
	RuntimeModuleString
	RuntimeModulePointer
	RuntimeModuleArray
	RuntimeModuleSlice
	RuntimeModuleMap
	RuntimeModulePanic
	RuntimeModuleInteger
	RuntimeModuleFloat
)

type RuntimeSymbol uint16

const (
	RuntimeInvalid          RuntimeSymbol = 0
	RuntimeStringIndex      RuntimeSymbol = 1
	RuntimeStringSlice      RuntimeSymbol = 2
	RuntimePointer          RuntimeSymbol = 100
	RuntimeArray            RuntimeSymbol = 200
	RuntimeSlice            RuntimeSymbol = 300
	RuntimeSliceAddress     RuntimeSymbol = 301
	RuntimeMap              RuntimeSymbol = 400
	RuntimePanic            RuntimeSymbol = 500
	RuntimeIntegerDivide    RuntimeSymbol = 600
	RuntimeIntegerRemainder RuntimeSymbol = 601
	RuntimeFloat32Round     RuntimeSymbol = 700
)

type RuntimeSymbolContract struct {
	module       RuntimeModule
	outputPath   string
	exportedName string
	typeUsable   bool
	dependencies []RuntimeSymbol
}

func RuntimeContract(symbol RuntimeSymbol) (RuntimeSymbolContract, error) {
	switch symbol {
	case RuntimeStringIndex:
		return runtimeContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringIndex",
			false,
			RuntimePanic,
		), nil
	case RuntimeStringSlice:
		return runtimeContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringSlice",
			false,
			RuntimePanic,
		), nil
	case RuntimePointer:
		return runtimeContract(
			RuntimeModulePointer,
			"runtime/pointer.ts",
			"GoPointer",
			true,
			RuntimePanic,
		), nil
	case RuntimeArray:
		return runtimeContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"GoArray",
			true,
			RuntimePanic,
		), nil
	case RuntimeSlice:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"RuntimeSlice",
			true,
			RuntimePanic,
		), nil
	case RuntimeSliceAddress:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAddress",
			false,
			RuntimeSlice,
			RuntimePointer,
		), nil
	case RuntimeMap:
		return runtimeContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"GoMap",
			true,
			RuntimePanic,
		), nil
	case RuntimePanic:
		return runtimeContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"GoPanic",
			true,
		), nil
	case RuntimeIntegerDivide:
		return runtimeContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goIntegerDivide",
			false,
			RuntimePanic,
		), nil
	case RuntimeIntegerRemainder:
		return runtimeContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goIntegerRemainder",
			false,
			RuntimePanic,
		), nil
	case RuntimeFloat32Round:
		return runtimeContract(
			RuntimeModuleFloat,
			"runtime/float.ts",
			"goFloat32",
			false,
		), nil
	default:
		return RuntimeSymbolContract{}, &RuntimeSymbolError{Symbol: symbol}
	}
}

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

func (c RuntimeSymbolContract) AllowsImportPhase(phase ImportPhase) bool {
	return phase == ImportPhaseValue ||
		(phase == ImportPhaseType && c.typeUsable)
}

type RuntimeSymbolError struct {
	Symbol RuntimeSymbol
}

func (e *RuntimeSymbolError) Error() string {
	return fmt.Sprintf("runtime symbol %d is invalid", e.Symbol)
}
