package api

import (
	"fmt"
	"slices"
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
