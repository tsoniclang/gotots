package complex

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	symbols []api.RuntimeSymbol,
) ([]tsgo.Statement, error) {
	if len(symbols) == 0 {
		return nil, &BuildError{Reason: "runtime symbol set is empty"}
	}
	roundContract, err := api.RuntimeContract(api.RuntimeFloat32Round)
	if err != nil {
		return nil, err
	}
	divideContract, err := api.RuntimeContract(api.RuntimeComplexDivide)
	if err != nil {
		return nil, err
	}
	seen := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	statements := make([]tsgo.Statement, 0, len(symbols))
	for _, symbol := range symbols {
		if _, duplicate := seen[symbol]; duplicate {
			return nil, &BuildError{
				Symbol: symbol,
				Reason: "runtime symbol is duplicated",
			}
		}
		seen[symbol] = struct{}{}
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		if contract.Module() != api.RuntimeModuleComplex {
			return nil, &BuildError{
				Symbol: symbol,
				Reason: "runtime symbol belongs to another module",
			}
		}
		statement, ok := buildSymbol(
			factory,
			symbol,
			contract.ExportedName(),
			roundContract.ExportedName(),
			divideContract.ExportedName(),
		)
		if !ok {
			return nil, &BuildError{
				Symbol: symbol,
				Reason: "runtime complex operation is not installed",
			}
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

func buildSymbol(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	name string,
	roundName string,
	divideName string,
) (tsgo.Statement, bool) {
	switch symbol {
	case api.RuntimeComplex64:
		return buildClass(
			factory,
			name,
			"goComplex64Brand",
			roundName,
		), true
	case api.RuntimeComplex128:
		return buildClass(
			factory,
			name,
			"goComplex128Brand",
			"",
		), true
	case api.RuntimeComplexDivide:
		return buildDivision(factory, name), true
	}
	className, ok := operationClass(symbol)
	if !ok {
		return nil, false
	}
	return buildOperation(factory, symbol, name, className, divideName)
}

func operationClass(symbol api.RuntimeSymbol) (string, bool) {
	var classSymbol api.RuntimeSymbol
	switch symbol {
	case api.RuntimeComplex64Add,
		api.RuntimeComplex64Sub,
		api.RuntimeComplex64Mul,
		api.RuntimeComplex64Div,
		api.RuntimeComplex64Neg,
		api.RuntimeComplex64Equal:
		classSymbol = api.RuntimeComplex64
	case api.RuntimeComplex128Add,
		api.RuntimeComplex128Sub,
		api.RuntimeComplex128Mul,
		api.RuntimeComplex128Div,
		api.RuntimeComplex128Neg,
		api.RuntimeComplex128Equal:
		classSymbol = api.RuntimeComplex128
	default:
		return "", false
	}
	contract, err := api.RuntimeContract(classSymbol)
	if err != nil {
		return "", false
	}
	return contract.ExportedName(), true
}

type BuildError struct {
	Symbol api.RuntimeSymbol
	Reason string
}

func (e *BuildError) Error() string {
	return fmt.Sprintf(
		"build complex runtime symbol %d: %s",
		e.Symbol,
		e.Reason,
	)
}
