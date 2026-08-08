package runtimecontract

type RuntimeSymbolContract struct {
	module       RuntimeModule
	outputPath   string
	exportedName string
	typeUsable   bool
	dependencies []RuntimeSymbol
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
	case RuntimeDenseIndex:
		return runtimeContract(
			RuntimeModuleDenseIndex,
			"runtime/dense-index.ts",
			"GoDenseIndex",
			false,
			RuntimePanic,
		), nil
	case RuntimeArray:
		return runtimeContract(
			RuntimeModuleArray,
			"runtime/array.ts",
			"GoArray",
			true,
			RuntimePanic,
			RuntimeDenseIndex,
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
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"RuntimeSlice",
			true,
			RuntimePanic,
			RuntimeDenseIndex,
		), nil
	case RuntimeSliceAddress:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAddress",
			false,
			RuntimeSlice,
		), nil
	case RuntimeSliceStorage:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceAllocate",
			false,
			RuntimeSlice,
		), nil
	case RuntimeSliceProjection:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"RuntimeSliceProjection",
			true,
			RuntimeSlice,
		), nil
	case RuntimeSliceArrayPointer:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceArrayPointer",
			false,
			RuntimeSlice,
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
	case RuntimeSliceRegion:
		return runtimeContract(
			RuntimeModuleSlice,
			"runtime/slice.ts",
			"goSliceRegion",
			false,
			RuntimeSlice,
			RuntimePanic,
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
	case RuntimeMapValue:
		return runtimeContract(
			RuntimeModuleMap,
			"runtime/map.ts",
			"GoMapValue",
			true,
		), nil
	case RuntimePanic:
		return runtimeContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"GoPanic",
			true,
			RuntimeInterfaceValue,
			RuntimePanicValue,
		), nil
	case RuntimePanicValue:
		return runtimeContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"GoRuntimePanicValue",
			true,
			RuntimeInterfaceValue,
			RuntimeErrorMethodToken,
			RuntimeRuntimeErrorToken,
		), nil
	case RuntimeRecovery:
		return runtimeContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"GoRecovery",
			true,
			RuntimePanic,
			RuntimeInterfaceValue,
		), nil
	case RuntimeDeferPop:
		return runtimeContract(
			RuntimeModulePanic,
			"runtime/panic.ts",
			"goDeferPop",
			false,
			RuntimePanic,
		), nil
	case RuntimeDeferredRegistry:
		return runtimeContract(
			RuntimeModuleDeferredRegistry,
			"runtime/deferred-registry.ts",
			"GoDeferredRegistry",
			true,
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
		return runtimeContract(
			RuntimeModulePanicNil,
			"runtime/panic-nil.ts",
			"GoPanicNilValue",
			true,
			RuntimePanicNilError,
			RuntimePanicValue,
			RuntimeInterfaceValue,
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
	case RuntimeNumberIntDivide:
		return runtimeContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goNumberIntegerDivide",
			false,
			RuntimePanic,
		), nil
	case RuntimeNumberIntRemainder:
		return runtimeContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goNumberIntegerRemainder",
			false,
			RuntimePanic,
		), nil
	case RuntimeIntegerNormalizeSigned64:
		return runtimeContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goInt64",
			false,
		), nil
	case RuntimeIntegerNormalizeUnsigned64:
		return runtimeContract(
			RuntimeModuleInteger,
			"runtime/integer.ts",
			"goUint64",
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
	case RuntimeEmptyStruct:
		return runtimeContract(
			RuntimeModuleStruct,
			"runtime/struct.ts",
			"GoEmptyStruct",
			true,
		), nil
	default:
		if contract, ok := interfaceRuntimeContract(symbol); ok {
			return contract, nil
		}
		return concurrencyRuntimeContract(symbol)
	}
}
