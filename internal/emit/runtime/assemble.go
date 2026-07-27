package runtime

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
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
		statements, err := stringruntime.Build(factory, symbols)
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
		definition, err := NewDefinition(
			api.RuntimePointer,
			pointerruntime.Build(factory, contract.ExportedName()),
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
