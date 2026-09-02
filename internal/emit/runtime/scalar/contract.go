package scalar

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func SharedDeclaration(
	alias api.PrimitiveAlias,
	abi api.ScalarABI,
) (tsoniccore.Declaration, bool, error) {
	_, carrier, err := api.PrimitiveAliasRepresentation(alias, abi)
	if err != nil {
		return tsoniccore.Declaration{}, false, err
	}
	symbol := sharedSymbol(alias, abi, carrier)
	if symbol == tsoniccore.SymbolInvalid {
		return tsoniccore.Declaration{}, false, nil
	}
	declaration, err := tsoniccore.Resolve(symbol)
	return declaration, err == nil, err
}

func sharedSymbol(
	alias api.PrimitiveAlias,
	abi api.ScalarABI,
	carrier tsgo.KeywordTypeSyntaxKind,
) tsoniccore.Symbol {
	switch alias {
	case api.PrimitiveBool:
		return tsoniccore.SymbolBool
	case api.PrimitiveInt8:
		return tsoniccore.SymbolInt8
	case api.PrimitiveInt16:
		return tsoniccore.SymbolInt16
	case api.PrimitiveInt32:
		return tsoniccore.SymbolInt32
	case api.PrimitiveInt64:
		if carrier == tsgo.KeywordTypeSyntaxKindBigIntKeyword {
			return tsoniccore.SymbolInt64
		}
	case api.PrimitiveUint8:
		return tsoniccore.SymbolUint8
	case api.PrimitiveUint16:
		return tsoniccore.SymbolUint16
	case api.PrimitiveUint32:
		return tsoniccore.SymbolUint32
	case api.PrimitiveUint64:
		if carrier == tsgo.KeywordTypeSyntaxKindBigIntKeyword {
			return tsoniccore.SymbolUint64
		}
	case api.PrimitiveFloat32:
		return tsoniccore.SymbolFloat32
	case api.PrimitiveFloat64:
		return tsoniccore.SymbolFloat64
	case api.PrimitiveInt:
		if abi.NativeIntegerWidth() == api.NativeIntegerWidth32 {
			return tsoniccore.SymbolInt32
		}
		if carrier == tsgo.KeywordTypeSyntaxKindBigIntKeyword {
			return tsoniccore.SymbolInt64
		}
	case api.PrimitiveUint, api.PrimitiveUintptr:
		if abi.NativeIntegerWidth() == api.NativeIntegerWidth32 {
			return tsoniccore.SymbolUint32
		}
		if carrier == tsgo.KeywordTypeSyntaxKindBigIntKeyword {
			return tsoniccore.SymbolUint64
		}
	}
	return tsoniccore.SymbolInvalid
}
