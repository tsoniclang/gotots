package runtimecontract

func descriptorRuntimeContract(symbol RuntimeSymbol) (RuntimeSymbolContract, bool) {
	switch symbol {
	case RuntimeStringTextBacking:
		return runtimeContract(RuntimeModuleStringValue, "runtime/string-value.ts", "GoStringTextBacking", true), true
	case RuntimeStringPointerBacking:
		return runtimeContract(RuntimeModuleStringValue, "runtime/string-value.ts", "GoStringPointerBacking", true,
			RuntimeStorageRegion, RuntimeRegionAddress, RuntimeRegionRead), true
	case RuntimeStringValue:
		return runtimeContract(RuntimeModuleStringValue, "runtime/string-value.ts", "GoString", true,
			RuntimeStringTextBacking, RuntimeStringPointerBacking, RuntimeStorageRegion, RuntimePanic), true
	case RuntimeStorageRegion:
		contract := runtimeContract(RuntimeModuleMemoryView, "runtime/memory-view.ts", "GoStorageRegion", true)
		contract.typeOnly = true
		return contract, true
	case RuntimeRegionAddress:
		return runtimeContract(RuntimeModuleMemoryView, "runtime/memory-view.ts", "goRegionAddress", false, RuntimeStorageRegion), true
	case RuntimeRegionRead:
		return runtimeContract(RuntimeModuleMemoryView, "runtime/memory-view.ts", "goRegionRead", false, RuntimeStorageRegion, RuntimePanic), true
	case RuntimeRegionWrite:
		return runtimeContract(RuntimeModuleMemoryView, "runtime/memory-view.ts", "goRegionWrite", false, RuntimeStorageRegion), true
	case RuntimeRegionView:
		return runtimeContract(RuntimeModuleMemoryView, "runtime/memory-view.ts", "goRegionView", false, RuntimeStorageRegion), true
	}
	var name string
	switch symbol {
	case RuntimeSliceHeader32:
		name = "GoSliceHeader32"
	case RuntimeSliceHeader64:
		name = "GoSliceHeader64"
	case RuntimeStringHeader32:
		name = "GoStringHeader32"
	case RuntimeStringHeader64:
		name = "GoStringHeader64"
	default:
		return RuntimeSymbolContract{}, false
	}
	return runtimeContract(RuntimeModuleMemoryDescriptor, "runtime/memory-descriptor.ts", name, true), true
}
