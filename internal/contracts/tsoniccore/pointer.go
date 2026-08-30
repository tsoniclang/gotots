package tsoniccore

import "fmt"

type Symbol uint8

const (
	SymbolInvalid Symbol = iota
	SymbolPointer
	SymbolAddressOf
	SymbolAllocatePointer
	SymbolLoadPointer
	SymbolStorePointer
	SymbolEqualPointer
	SymbolHashPointer
	SymbolProjectPointer
	SymbolBindPointer
	SymbolRawPointer
	SymbolBindRawPointer
	SymbolEqualRawPointer
	SymbolHashRawPointer
)

type Phase uint8

const (
	PhaseInvalid Phase = iota
	PhaseType
	PhaseValue
)

type Declaration struct {
	module string
	export string
	phase  Phase
}

func (d Declaration) Module() string {
	return d.module
}

func (d Declaration) Export() string {
	return d.export
}

func (d Declaration) Phase() Phase {
	return d.phase
}

func Resolve(symbol Symbol) (Declaration, error) {
	switch symbol {
	case SymbolPointer:
		return Declaration{
			module: "@tsonic/core/types.js",
			export: "Pointer",
			phase:  PhaseType,
		}, nil
	case SymbolAddressOf:
		return value("addressOf"), nil
	case SymbolAllocatePointer:
		return value("allocatePointer"), nil
	case SymbolLoadPointer:
		return value("loadPointer"), nil
	case SymbolStorePointer:
		return value("storePointer"), nil
	case SymbolEqualPointer:
		return value("equalPointer"), nil
	case SymbolHashPointer:
		return value("hashPointer"), nil
	case SymbolProjectPointer:
		return value("projectPointer"), nil
	case SymbolBindPointer:
		return value("bindPointer"), nil
	case SymbolRawPointer:
		return Declaration{
			module: "@tsonic/core/types.js",
			export: "RawPointer",
			phase:  PhaseType,
		}, nil
	case SymbolBindRawPointer:
		return value("bindRawPointer"), nil
	case SymbolEqualRawPointer:
		return value("equalRawPointer"), nil
	case SymbolHashRawPointer:
		return value("hashRawPointer"), nil
	default:
		return Declaration{}, fmt.Errorf(
			"resolve Tsonic core symbol: invalid symbol %d",
			symbol,
		)
	}
}

func value(export string) Declaration {
	return Declaration{
		module: "@tsonic/core/lang.js",
		export: export,
		phase:  PhaseValue,
	}
}
