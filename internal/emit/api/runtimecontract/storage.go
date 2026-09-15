package runtimecontract

func storageTypeContract(
	module RuntimeModule,
	outputPath string,
	exportedName string,
	dependencies ...RuntimeSymbol,
) RuntimeSymbolContract {
	contract := runtimeContract(module, outputPath, exportedName, true, dependencies...)
	contract.typeOnly = true
	return contract
}
