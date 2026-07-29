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
	RuntimeModuleConversion
	RuntimeModuleInterface
)

type RuntimeSymbol uint16

const (
	RuntimeInvalid           RuntimeSymbol = 0
	RuntimeStringIndex       RuntimeSymbol = 1
	RuntimeStringSlice       RuntimeSymbol = 2
	RuntimeStringMax         RuntimeSymbol = 3
	RuntimeStringMin         RuntimeSymbol = 4
	RuntimeStringEncodeRune  RuntimeSymbol = 5
	RuntimeStringDecodeRune  RuntimeSymbol = 6
	RuntimePointer           RuntimeSymbol = 100
	RuntimePointerHash       RuntimeSymbol = 101
	RuntimeArray             RuntimeSymbol = 200
	RuntimeArrayAllocate     RuntimeSymbol = 201
	RuntimeArrayView         RuntimeSymbol = 202
	RuntimeArrayLocation     RuntimeSymbol = 203
	RuntimeSlice             RuntimeSymbol = 300
	RuntimeSliceAddress      RuntimeSymbol = 301
	RuntimeSliceStorage      RuntimeSymbol = 302
	RuntimeSliceAddressView  RuntimeSymbol = 303
	RuntimeSliceArrayPointer RuntimeSymbol = 304
	RuntimeArraySlice        RuntimeSymbol = 305
	RuntimeSliceAppendSlice  RuntimeSymbol = 307
	RuntimeSliceClear        RuntimeSymbol = 308
	RuntimeMap               RuntimeSymbol = 400
	RuntimeMapHash           RuntimeSymbol = 401
	RuntimeMapClear          RuntimeSymbol = 402
	RuntimeMapKeys           RuntimeSymbol = 403
	RuntimePanic             RuntimeSymbol = 500
	RuntimeIntegerDivide     RuntimeSymbol = 600
	RuntimeIntegerRemainder  RuntimeSymbol = 601
	RuntimeIntegerMax        RuntimeSymbol = 602
	RuntimeIntegerMin        RuntimeSymbol = 603
	RuntimeFloat32Round      RuntimeSymbol = 700
	RuntimeComplex64         RuntimeSymbol = 800
	RuntimeComplex128        RuntimeSymbol = 801
	RuntimeComplexDivide     RuntimeSymbol = 802
	RuntimeComplex64Add      RuntimeSymbol = 810
	RuntimeComplex64Sub      RuntimeSymbol = 811
	RuntimeComplex64Mul      RuntimeSymbol = 812
	RuntimeComplex64Div      RuntimeSymbol = 813
	RuntimeComplex64Neg      RuntimeSymbol = 814
	RuntimeComplex64Equal    RuntimeSymbol = 815
	RuntimeComplex128Add     RuntimeSymbol = 820
	RuntimeComplex128Sub     RuntimeSymbol = 821
	RuntimeComplex128Mul     RuntimeSymbol = 822
	RuntimeComplex128Div     RuntimeSymbol = 823
	RuntimeComplex128Neg     RuntimeSymbol = 824
	RuntimeComplex128Equal   RuntimeSymbol = 825
	RuntimeNumberToBigInt    RuntimeSymbol = 900
	RuntimeInterfaceValue    RuntimeSymbol = 1000
	RuntimeInterfaceNonNil   RuntimeSymbol = 1001
	RuntimeInterfaceEqual    RuntimeSymbol = 1002
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
	case RuntimeStringMax:
		return runtimeContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringMax",
			false,
		), nil
	case RuntimeStringMin:
		return runtimeContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringMin",
			false,
		), nil
	case RuntimeStringEncodeRune:
		return runtimeContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringEncodeRune",
			false,
		), nil
	case RuntimeStringDecodeRune:
		return runtimeContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringDecodeRune",
			false,
		), nil
	case RuntimePointer:
		return runtimeContract(
			RuntimeModulePointer,
			"runtime/pointer.ts",
			"GoPointer",
			true,
			RuntimePanic,
		), nil
	case RuntimePointerHash:
		return runtimeContract(
			RuntimeModulePointer,
			"runtime/pointer.ts",
			"goPointerHash",
			false,
			RuntimePointer,
			RuntimeMapHash,
		), nil
	case RuntimeArray:
		return runtimeContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"GoArray",
			true,
			RuntimePanic,
		), nil
	case RuntimeArrayAllocate:
		return runtimeContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"goArrayAllocate",
			false,
			RuntimeArray,
		), nil
	case RuntimeArrayView:
		return runtimeContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"goArrayView",
			false,
			RuntimeArray,
		), nil
	case RuntimeArrayLocation:
		return runtimeContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"goArrayLocation",
			false,
			RuntimeArray,
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
	case RuntimeSliceStorage:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAllocate",
			false,
			RuntimeSlice,
		), nil
	case RuntimeSliceAddressView:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAddressView",
			false,
			RuntimeSlice,
			RuntimePointer,
		), nil
	case RuntimeSliceArrayPointer:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceArrayPointer",
			false,
			RuntimeSlice,
			RuntimePointer,
			RuntimeArray,
			RuntimeArrayView,
		), nil
	case RuntimeArraySlice:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goArraySlice",
			false,
			RuntimeSlice,
			RuntimeArray,
			RuntimeArrayLocation,
		), nil
	case RuntimeSliceAppendSlice:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAppendSlice",
			false,
			RuntimeSlice,
		), nil
	case RuntimeSliceClear:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceClear",
			false,
			RuntimeSlice,
		), nil
	case RuntimeMap:
		return runtimeContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"GoMap",
			true,
			RuntimePanic,
		), nil
	case RuntimeMapHash:
		return runtimeContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"GoMapHash",
			false,
		), nil
	case RuntimeMapClear:
		return runtimeContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"goMapClear",
			false,
			RuntimeMap,
		), nil
	case RuntimeMapKeys:
		return runtimeContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"goMapKeys",
			false,
			RuntimeMap,
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
	case RuntimeIntegerMax:
		return runtimeContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goIntegerMax",
			false,
		), nil
	case RuntimeIntegerMin:
		return runtimeContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goIntegerMin",
			false,
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
	case RuntimeNumberToBigInt:
		return runtimeContract(
			RuntimeModuleConversion,
			"runtime/conversion.ts",
			"goNumberToBigInt",
			false,
		), nil
	case RuntimeInterfaceValue:
		return runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"GoInterfaceValue",
			true,
		), nil
	case RuntimeInterfaceNonNil:
		return runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"goInterfaceNonNil",
			false,
			RuntimeInterfaceValue,
			RuntimePanic,
		), nil
	case RuntimeInterfaceEqual:
		return runtimeContract(
			RuntimeModuleInterface,
			"runtime/interface.ts",
			"goInterfaceEqual",
			false,
			RuntimeInterfaceValue,
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
