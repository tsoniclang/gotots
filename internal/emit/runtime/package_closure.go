package runtime

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func dependencyClosure(
	requested map[api.RuntimeSymbol]struct{},
) (map[api.RuntimeSymbol]struct{}, error) {
	result := make(map[api.RuntimeSymbol]struct{}, len(requested))
	state := make(map[api.RuntimeSymbol]uint8, len(requested))
	var visit func(api.RuntimeSymbol) error
	visit = func(symbol api.RuntimeSymbol) error {
		switch state[symbol] {
		case 1:
			return &AssemblyError{
				Symbol: symbol,
				Reason: "runtime dependency graph contains a cycle",
			}
		case 2:
			return nil
		}
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return err
		}
		state[symbol] = 1
		dependencies := contract.Dependencies()
		slices.Sort(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[symbol] = 2
		result[symbol] = struct{}{}
		return nil
	}
	symbols := make([]api.RuntimeSymbol, 0, len(requested))
	for symbol := range requested {
		symbols = append(symbols, symbol)
	}
	slices.Sort(symbols)
	for _, symbol := range symbols {
		if err := visit(symbol); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func exactDefinitions(
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
	definitions []Definition,
) ([]tsgo.Statement, error) {
	requested := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	for _, symbol := range symbols {
		requested[symbol] = struct{}{}
	}
	bySymbol := make(map[api.RuntimeSymbol]tsgo.Statement, len(definitions))
	for _, definition := range definitions {
		symbol := definition.Symbol()
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		if contract.Module() != module {
			return nil, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "definition belongs to another runtime module",
			}
		}
		if _, ok := requested[symbol]; !ok {
			return nil, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "definition was not requested",
			}
		}
		if _, duplicate := bySymbol[symbol]; duplicate {
			return nil, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "runtime symbol has duplicate definitions",
			}
		}
		bySymbol[symbol] = definition.Statement()
	}
	statements := make([]tsgo.Statement, 0, len(symbols))
	for _, symbol := range symbols {
		statement := bySymbol[symbol]
		if statement == nil {
			return nil, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "requested runtime symbol has no definition",
			}
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

func moduleImports(
	factory tsgo.Factory,
	outputPath string,
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
) ([]tsgo.Statement, error) {
	placement := targetplacement.New()
	for _, symbol := range symbols {
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		for _, dependency := range contract.Dependencies() {
			dependencyContract, err := api.RuntimeContract(dependency)
			if err != nil {
				return nil, err
			}
			if dependencyContract.Module() == module {
				continue
			}
			modulePath, err := targetoutput.ModuleSpecifier(
				outputPath,
				dependencyContract.OutputPath(),
			)
			if err != nil {
				return nil, err
			}
			request, err := api.NewRuntimeImportRequest(
				factory,
				api.ImportPhaseValue,
				modulePath,
				dependency,
				dependencyContract.ExportedName(),
			)
			if err != nil {
				return nil, err
			}
			if err := placement.Apply([]api.RootRequest{request}); err != nil {
				return nil, err
			}
		}
	}
	if module == api.RuntimeModuleSlice &&
		(slices.Contains(symbols, api.RuntimeSliceAddress) ||
			slices.Contains(symbols, api.RuntimeSliceArrayPointer)) {
		for _, symbol := range []tsoniccore.Symbol{
			tsoniccore.SymbolPointer,
			tsoniccore.SymbolAddressOf,
			tsoniccore.SymbolProjectPointer,
		} {
			declaration, err := tsoniccore.Resolve(symbol)
			if err != nil {
				return nil, err
			}
			phase := api.ImportPhaseValue
			if declaration.Phase() == tsoniccore.PhaseType {
				phase = api.ImportPhaseType
			}
			request, err := api.NewImportRequest(
				factory,
				phase,
				declaration.Module(),
				declaration.Export(),
				declaration.Export(),
			)
			if err != nil {
				return nil, err
			}
			if err := placement.Apply([]api.RootRequest{request}); err != nil {
				return nil, err
			}
		}
	}
	if module == api.RuntimeModulePanicNil &&
		slices.Contains(symbols, api.RuntimePanicNilValue) {
		for _, symbol := range []tsoniccore.Symbol{
			tsoniccore.SymbolPointer,
			tsoniccore.SymbolAllocatePointer,
		} {
			declaration, err := tsoniccore.Resolve(symbol)
			if err != nil {
				return nil, err
			}
			phase := api.ImportPhaseValue
			if declaration.Phase() == tsoniccore.PhaseType {
				phase = api.ImportPhaseType
			}
			request, err := api.NewImportRequest(
				factory,
				phase,
				declaration.Module(),
				declaration.Export(),
				declaration.Export(),
			)
			if err != nil {
				return nil, err
			}
			if err := placement.Apply([]api.RootRequest{request}); err != nil {
				return nil, err
			}
		}
	}
	return placement.Statements(factory), nil
}
