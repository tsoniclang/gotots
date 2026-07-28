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
	RuntimeModuleComplex
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
	RuntimeComplex64        RuntimeSymbol = 800
	RuntimeComplex128       RuntimeSymbol = 801
	RuntimeComplexDivide    RuntimeSymbol = 802
	RuntimeComplex64Add     RuntimeSymbol = 810
	RuntimeComplex64Sub     RuntimeSymbol = 811
	RuntimeComplex64Mul     RuntimeSymbol = 812
	RuntimeComplex64Div     RuntimeSymbol = 813
	RuntimeComplex64Neg     RuntimeSymbol = 814
	RuntimeComplex64Equal   RuntimeSymbol = 815
	RuntimeComplex128Add    RuntimeSymbol = 820
	RuntimeComplex128Sub    RuntimeSymbol = 821
	RuntimeComplex128Mul    RuntimeSymbol = 822
	RuntimeComplex128Div    RuntimeSymbol = 823
	RuntimeComplex128Neg    RuntimeSymbol = 824
	RuntimeComplex128Equal  RuntimeSymbol = 825
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
	case RuntimeComplex64:
		return runtimeContract(
			RuntimeModuleComplex,
			"runtime/complex.ts",
			"GoComplex64",
			true,
			RuntimeFloat32Round,
		), nil
	case RuntimeComplex128:
		return runtimeContract(
			RuntimeModuleComplex,
			"runtime/complex.ts",
			"GoComplex128",
			true,
		), nil
	case RuntimeComplexDivide:
		return runtimeContract(
			RuntimeModuleComplex,
			"runtime/complex.ts",
			"goComplexDivide",
			false,
		), nil
	case RuntimeComplex64Add:
		return complexOperationContract(
			"goComplex64Add",
			RuntimeComplex64,
		)
	case RuntimeComplex64Sub:
		return complexOperationContract(
			"goComplex64Subtract",
			RuntimeComplex64,
		)
	case RuntimeComplex64Mul:
		return complexOperationContract(
			"goComplex64Multiply",
			RuntimeComplex64,
		)
	case RuntimeComplex64Div:
		return complexOperationContract(
			"goComplex64Divide",
			RuntimeComplex64,
			RuntimeComplexDivide,
		)
	case RuntimeComplex64Neg:
		return complexOperationContract(
			"goComplex64Negate",
			RuntimeComplex64,
		)
	case RuntimeComplex64Equal:
		return complexOperationContract(
			"goComplex64Equal",
			RuntimeComplex64,
		)
	case RuntimeComplex128Add:
		return complexOperationContract(
			"goComplex128Add",
			RuntimeComplex128,
		)
	case RuntimeComplex128Sub:
		return complexOperationContract(
			"goComplex128Subtract",
			RuntimeComplex128,
		)
	case RuntimeComplex128Mul:
		return complexOperationContract(
			"goComplex128Multiply",
			RuntimeComplex128,
		)
	case RuntimeComplex128Div:
		return complexOperationContract(
			"goComplex128Divide",
			RuntimeComplex128,
			RuntimeComplexDivide,
		)
	case RuntimeComplex128Neg:
		return complexOperationContract(
			"goComplex128Negate",
			RuntimeComplex128,
		)
	case RuntimeComplex128Equal:
		return complexOperationContract(
			"goComplex128Equal",
			RuntimeComplex128,
		)
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
