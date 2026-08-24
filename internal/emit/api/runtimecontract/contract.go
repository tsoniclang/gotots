package runtimecontract

import "slices"

type RuntimeSymbolContract struct {
	module       RuntimeModule
	outputPath   string
	exportedName string
	typeUsable   bool
	dependencies []RuntimeSymbol
	invocation   RuntimeInvocationContract
}

type RuntimeInvocationContract struct {
	exactImplementation    bool
	inputParameters        []uint32
	resultOriginParameters []uint32
}

func (c RuntimeInvocationContract) Valid() bool {
	return c.exactImplementation ||
		len(c.inputParameters) != 0 ||
		len(c.resultOriginParameters) != 0
}

func (c RuntimeInvocationContract) ExactImplementation() bool {
	return c.exactImplementation
}

func (c RuntimeInvocationContract) InputParameters() []uint32 {
	return slices.Clone(c.inputParameters)
}

func (c RuntimeInvocationContract) ResultOriginParameters() []uint32 {
	return slices.Clone(c.resultOriginParameters)
}

func (c RuntimeSymbolContract) Invocation() (
	RuntimeInvocationContract,
	bool,
) {
	return c.invocation, c.invocation.Valid()
}

func RuntimeContract(symbol RuntimeSymbol) (RuntimeSymbolContract, error) {
	if contract, ok := unsafeRuntimeContract(symbol); ok {
		return contract, nil
	}
	switch symbol {
	case RuntimeAwaitable:
		return runtimeContract(
			RuntimeModuleScalar,
			"runtime/scalars.ts",
			"Awaitable",
			true,
		), nil
	case RuntimeStringIndex:
		return runtimeFunctionContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringIndex",
			RuntimePanic,
		), nil
	case RuntimeStringSlice:
		return runtimeFunctionContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringSlice",
			RuntimePanic,
		), nil
	case RuntimeStringMax:
		return runtimeFunctionContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringMax",
		), nil
	case RuntimeStringMin:
		return runtimeFunctionContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringMin",
		), nil
	case RuntimeStringEncodeRune:
		return runtimeFunctionContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringEncodeRune",
		), nil
	case RuntimeStringDecodeRune:
		return runtimeFunctionContract(
			RuntimeModuleString,
			"runtime/string.ts",
			"goStringDecodeRune",
		), nil
	case RuntimeArray:
		return runtimeClassContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"GoArray",
			RuntimePanic,
		), nil
	case RuntimeArrayAllocate:
		return runtimeFunctionContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"goArrayAllocate",
			RuntimeArray,
		), nil
	case RuntimeArrayView:
		return runtimeFunctionContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"goArrayView",
			RuntimeArray,
		), nil
	case RuntimeArrayLocation:
		return runtimeFunctionContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"goArrayLocation",
			RuntimeArray,
		), nil
	case RuntimeArrayPacked:
		return runtimeFunctionContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"goArrayPacked",
			RuntimeArray,
		), nil
	case RuntimeStorageTypeToken:
		return runtimeContract(
			RuntimeModuleStorage,
			"runtime/storage.ts",
			"$goStorageType",
			true,
		), nil
	case RuntimeStoredValue:
		return runtimeContract(
			RuntimeModuleStorage,
			"runtime/storage.ts",
			"GoStoredValue",
			true,
			RuntimeStorageTypeToken,
		), nil
	case RuntimeStorageType:
		return runtimeContract(
			RuntimeModuleStorage,
			"runtime/storage.ts",
			"GoStorage",
			true,
			RuntimeStoredValue,
		), nil
	case RuntimeContainerStorageToken:
		return runtimeContract(
			RuntimeModuleStorage,
			"runtime/storage.ts",
			"$goContainerStorageType",
			true,
		), nil
	case RuntimeContainerStoredValue:
		return runtimeContract(
			RuntimeModuleStorage,
			"runtime/storage.ts",
			"GoContainerStoredValue",
			true,
			RuntimeContainerStorageToken,
		), nil
	case RuntimeContainerStorageType:
		return runtimeContract(
			RuntimeModuleStorage,
			"runtime/storage.ts",
			"GoContainerStorage",
			true,
			RuntimeContainerStoredValue,
		), nil
	case RuntimeSlice:
		return runtimeClassContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"RuntimeSlice",
			RuntimePanic,
		), nil
	case RuntimeSliceAddress:
		return runtimeFunctionContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAddress",
			RuntimeSlice,
		), nil
	case RuntimeSliceStorage:
		return runtimeFunctionContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAllocate",
			RuntimeSlice,
		), nil
	case RuntimeSliceProjection:
		return runtimeClassContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"RuntimeSliceProjection",
			RuntimeSlice,
		), nil
	case RuntimeSliceArrayPointer:
		return runtimeFunctionContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceArrayPointer",
			RuntimeSlice,
			RuntimeArray,
			RuntimeArrayView,
		), nil
	case RuntimeArraySlice:
		return runtimeFunctionContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goArraySlice",
			RuntimeSlice,
			RuntimeArray,
			RuntimeArrayLocation,
		), nil
	case RuntimeSliceAppendSlice:
		return runtimeFunctionContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAppendSlice",
			RuntimeSlice,
		), nil
	case RuntimeSliceClear:
		return runtimeFunctionContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceClear",
			RuntimeSlice,
		), nil
	case RuntimeSliceRegion:
		return runtimeFunctionContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceRegion",
			RuntimeSlice,
			RuntimePanic,
		), nil
	case RuntimeMap:
		return runtimeClassContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"GoMap",
			RuntimePanic,
			RuntimeMapValue,
		), nil
	case RuntimeMapHash:
		return runtimeCallableOwnerContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"GoMapHash",
			false,
		), nil
	case RuntimeMapClear:
		return runtimeFunctionContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"goMapClear",
			RuntimeMap,
		), nil
	case RuntimeMapKeys:
		return runtimeFunctionContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"goMapKeys",
			RuntimeMap,
		), nil
	case RuntimeMapValue:
		return runtimeContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"GoMapValue",
			true,
		), nil
	case RuntimePanic:
		return runtimeClassContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"GoPanic",
			RuntimeInterfaceValue,
			RuntimePanicValue,
		), nil
	case RuntimePanicValue:
		return runtimeClassContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"GoRuntimePanicValue",
			RuntimeInterfaceValue,
			RuntimeErrorMethodToken,
			RuntimeRuntimeErrorToken,
		), nil
	case RuntimeRecovery:
		return runtimeClassContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"GoRecovery",
			RuntimePanic,
			RuntimeInterfaceValue,
		), nil
	case RuntimeDeferPop:
		return withInvocationContract(runtimeContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"goDeferPop",
			false,
			RuntimePanic,
		), true, nil, nil), nil
	case RuntimeDeferredRegistry:
		return runtimeClassContract(
			RuntimeModuleDeferredRegistry,
			"runtime/deferred-registry.ts",
			"GoDeferredRegistry",
			RuntimeInterfaceValue,
		), nil
	case RuntimePanicNilError:
		return runtimeContract(
			RuntimeModulePanicNil,
			"runtime/panic-nil.ts",
			"GoPanicNilError",
			true,
		), nil
	case RuntimePanicNilValue:
		return runtimeClassContract(
			RuntimeModulePanicNil,
			"runtime/panic-nil.ts",
			"GoPanicNilValue",
			RuntimePanicNilError,
			RuntimePanicValue,
			RuntimeInterfaceValue,
		), nil
	case RuntimeIntegerDivide:
		return runtimeFunctionContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goIntegerDivide",
			RuntimePanic,
		), nil
	case RuntimeIntegerRemainder:
		return runtimeFunctionContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goIntegerRemainder",
			RuntimePanic,
		), nil
	case RuntimeIntegerMax:
		return runtimeFunctionContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goIntegerMax",
		), nil
	case RuntimeIntegerMin:
		return runtimeFunctionContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goIntegerMin",
		), nil
	case RuntimeNumberIntDivide:
		return runtimeFunctionContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goNumberIntegerDivide",
			RuntimePanic,
		), nil
	case RuntimeNumberIntRemainder:
		return runtimeFunctionContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goNumberIntegerRemainder",
			RuntimePanic,
		), nil
	case RuntimeIntegerNormalizeSigned64:
		return runtimeFunctionContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goInt64",
		), nil
	case RuntimeIntegerNormalizeUnsigned64:
		return runtimeFunctionContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goUint64",
		), nil
	case RuntimeFloat32Round:
		return runtimeFunctionContract(
			RuntimeModuleFloat,
			"runtime/float.ts",
			"goFloat32",
		), nil
	case RuntimeComplex64:
		return runtimeClassContract(
			RuntimeModuleComplex,
			"runtime/complex.ts",
			"GoComplex64",
			RuntimeFloat32Round,
		), nil
	case RuntimeComplex128:
		return runtimeClassContract(
			RuntimeModuleComplex,
			"runtime/complex.ts",
			"GoComplex128",
		), nil
	case RuntimeComplexDivide:
		return runtimeFunctionContract(
			RuntimeModuleComplex,
			"runtime/complex.ts",
			"goComplexDivide",
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
		return runtimeFunctionContract(
			RuntimeModuleConversion,
			"runtime/conversion.ts",
			"goNumberToBigInt",
		), nil
	case RuntimeEmptyStruct:
		return runtimeClassContract(
			RuntimeModuleStruct,
			"runtime/struct.ts",
			"GoEmptyStruct",
		), nil
	default:
		if contract, ok := interfaceRuntimeContract(symbol); ok {
			return contract, nil
		}
		return concurrencyRuntimeContract(symbol)
	}
}
