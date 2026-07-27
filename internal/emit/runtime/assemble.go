package runtime

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimearray "github.com/tsoniclang/gotots/internal/emit/runtime/array"
	floatruntime "github.com/tsoniclang/gotots/internal/emit/runtime/float"
	integerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/integer"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	stringruntime "github.com/tsoniclang/gotots/internal/emit/runtime/string"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Definition struct {
	symbol    api.RuntimeSymbol
	statement tsgo.Statement
}

func NewDefinition(
	symbol api.RuntimeSymbol,
	statement tsgo.Statement,
) (Definition, error) {
	if _, err := api.RuntimeContract(symbol); err != nil {
		return Definition{}, err
	}
	if statement == nil {
		return Definition{}, &AssemblyError{
			Symbol: symbol,
			Reason: "target statement is nil",
		}
	}
	return Definition{symbol: symbol, statement: statement}, nil
}

func (d Definition) Symbol() api.RuntimeSymbol {
	return d.symbol
}

func (d Definition) Statement() tsgo.Statement {
	return d.statement
}

func Build(
	factory tsgo.Factory,
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
) ([]Definition, error) {
	if module == api.RuntimeModuleInvalid {
		return nil, &AssemblyError{Reason: "runtime module is invalid"}
	}
	if len(symbols) == 0 {
		return nil, &AssemblyError{Reason: "runtime symbol set is empty"}
	}
	if module == api.RuntimeModuleString {
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		statements, err := stringruntime.Build(
			factory,
			symbols,
			panicContract.ExportedName(),
		)
		if err != nil {
			return nil, err
		}
		if len(statements) != len(symbols) {
			return nil, &AssemblyError{
				Module: module,
				Reason: "string runtime returned a non-exact definition set",
			}
		}
		definitions := make([]Definition, 0, len(symbols))
		for index, symbol := range symbols {
			definition, err := NewDefinition(symbol, statements[index])
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModulePointer {
		if len(symbols) != 1 || symbols[0] != api.RuntimePointer {
			return nil, &AssemblyError{
				Module: module,
				Reason: "pointer runtime requires exactly RuntimePointer",
			}
		}
		contract, err := api.RuntimeContract(api.RuntimePointer)
		if err != nil {
			return nil, err
		}
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(
			api.RuntimePointer,
			pointerruntime.Build(
				factory,
				contract.ExportedName(),
				panicContract.ExportedName(),
			),
		)
		if err != nil {
			return nil, err
		}
		return []Definition{definition}, nil
	}
	if module == api.RuntimeModuleArray &&
		len(symbols) == 1 &&
		symbols[0] == api.RuntimeArray {
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		statement, err := runtimearray.Build(
			factory,
			panicContract.ExportedName(),
		)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(
			api.RuntimeArray,
			statement,
		)
		if err != nil {
			return nil, err
		}
		return []Definition{definition}, nil
	}
	if module == api.RuntimeModuleSlice &&
		(len(symbols) == 1 || len(symbols) == 2) &&
		symbols[0] == api.RuntimeSlice &&
		(len(symbols) == 1 || symbols[1] == api.RuntimeSliceAddress) {
		contract, err := api.RuntimeContract(api.RuntimeSlice)
		if err != nil {
			return nil, err
		}
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		address := len(symbols) == 2
		definition, err := NewDefinition(
			api.RuntimeSlice,
			runtimeslice.BuildWithAddress(
				factory,
				contract.ExportedName(),
				panicContract.ExportedName(),
				address,
			),
		)
		if err != nil {
			return nil, err
		}
		definitions := []Definition{definition}
		if address {
			addressContract, err := api.RuntimeContract(
				api.RuntimeSliceAddress,
			)
			if err != nil {
				return nil, err
			}
			pointerContract, err := api.RuntimeContract(api.RuntimePointer)
			if err != nil {
				return nil, err
			}
			addressDefinition, err := NewDefinition(
				api.RuntimeSliceAddress,
				runtimeslice.BuildAddress(
					factory,
					addressContract.ExportedName(),
					contract.ExportedName(),
					pointerContract.ExportedName(),
				),
			)
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, addressDefinition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModuleMap {
		return buildMap(factory, symbols)
	}
	if module == api.RuntimeModuleInteger {
		panicContract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		statements, err := integerruntime.Build(
			factory,
			symbols,
			panicContract.ExportedName(),
		)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		for index, symbol := range symbols {
			definition, err := NewDefinition(symbol, statements[index])
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModuleFloat {
		statements, err := floatruntime.Build(factory, symbols)
		if err != nil {
			return nil, err
		}
		definitions := make([]Definition, 0, len(symbols))
		for index, symbol := range symbols {
			definition, err := NewDefinition(symbol, statements[index])
			if err != nil {
				return nil, err
			}
			definitions = append(definitions, definition)
		}
		return definitions, nil
	}
	if module == api.RuntimeModulePanic &&
		len(symbols) == 1 &&
		symbols[0] == api.RuntimePanic {
		contract, err := api.RuntimeContract(api.RuntimePanic)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(
			api.RuntimePanic,
			panicruntime.Build(factory, contract.ExportedName()),
		)
		if err != nil {
			return nil, err
		}
		return []Definition{definition}, nil
	}
	return nil, &AssemblyError{
		Module: module,
		Reason: "runtime module owner is not installed",
	}
}

func buildMap(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
) ([]Definition, error) {
	panicContract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		return nil, err
	}
	result := make([]Definition, 0, len(symbols))
	for _, symbol := range symbols {
		statement, err := mapruntime.Build(
			factory,
			symbol,
			panicContract.ExportedName(),
		)
		if err != nil {
			return nil, err
		}
		definition, err := NewDefinition(symbol, statement)
		if err != nil {
			return nil, err
		}
		result = append(result, definition)
	}
	return result, nil
}

type AssemblyError struct {
	Module api.RuntimeModule
	Symbol api.RuntimeSymbol
	Reason string
}

func (e *AssemblyError) Error() string {
	return fmt.Sprintf(
		"assemble runtime module %d symbol %d: %s",
		e.Module,
		e.Symbol,
		e.Reason,
	)
}
