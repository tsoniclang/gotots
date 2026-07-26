package runtime

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimearray "github.com/tsoniclang/gotots/internal/emit/runtime/array"
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
	if module == api.RuntimeModuleArray &&
		len(symbols) == 1 &&
		symbols[0] == api.RuntimeArray {
		statement, err := runtimearray.Build(factory)
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
