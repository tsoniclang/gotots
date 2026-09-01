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
	SymbolBool
	SymbolInt8
	SymbolUint8
	SymbolInt16
	SymbolUint16
	SymbolInt32
	SymbolUint32
	SymbolInt64
	SymbolUint64
	SymbolFloat32
	SymbolFloat64
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
	case SymbolBool:
		return typeDeclaration("bool"), nil
	case SymbolInt8:
		return typeDeclaration("int8"), nil
	case SymbolUint8:
		return typeDeclaration("uint8"), nil
	case SymbolInt16:
		return typeDeclaration("int16"), nil
	case SymbolUint16:
		return typeDeclaration("uint16"), nil
	case SymbolInt32:
		return typeDeclaration("int32"), nil
	case SymbolUint32:
		return typeDeclaration("uint32"), nil
	case SymbolInt64:
		return typeDeclaration("int64"), nil
	case SymbolUint64:
		return typeDeclaration("uint64"), nil
	case SymbolFloat32:
		return typeDeclaration("float32"), nil
	case SymbolFloat64:
		return typeDeclaration("float64"), nil
	default:
		return Declaration{}, fmt.Errorf(
			"resolve Tsonic core symbol: invalid symbol %d",
			symbol,
		)
	}
}

func typeDeclaration(export string) Declaration {
	return Declaration{
		module: "@tsonic/core/types.js",
		export: export,
		phase:  PhaseType,
	}
}

func value(export string) Declaration {
	return Declaration{
		module: "@tsonic/core/lang.js",
		export: export,
		phase:  PhaseValue,
	}
}
