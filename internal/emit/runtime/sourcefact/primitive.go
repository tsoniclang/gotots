package sourcefact

import (
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const runtimePrimitiveSchema = "gotots-go-runtime-primitive-fact-v2"

type Primitive struct {
	Kind    string
	Role    string
	Width   uint8
	Signed  string
	Carrier string
	Shared  tsoniccore.Symbol
}

func DescribePrimitive(
	alias api.PrimitiveAlias,
	abi api.ScalarABI,
) (Primitive, error) {
	name, keyword, err := api.PrimitiveAliasRepresentation(alias, abi)
	if err != nil {
		return Primitive{}, err
	}
	carrier, err := carrierName(keyword)
	if err != nil {
		return Primitive{}, err
	}
	primitive := Primitive{Kind: name, Role: "fixed", Signed: "none", Carrier: carrier}
	switch alias {
	case api.PrimitiveBool:
		primitive.Role = "boolean"
	case api.PrimitiveString:
		primitive.Role = "go-string"
		primitive.Width = 8
	case api.PrimitiveFloat32:
		primitive.Role = "float"
		primitive.Width = 32
		primitive.Signed = "signed"
	case api.PrimitiveFloat64:
		primitive.Role = "float"
		primitive.Width = 64
		primitive.Signed = "signed"
	case api.PrimitiveInt8:
		primitive.Width, primitive.Signed = 8, "signed"
	case api.PrimitiveInt16:
		primitive.Width, primitive.Signed = 16, "signed"
	case api.PrimitiveInt32:
		primitive.Width, primitive.Signed = 32, "signed"
	case api.PrimitiveInt64:
		primitive.Width, primitive.Signed = 64, "signed"
	case api.PrimitiveUint8:
		primitive.Width, primitive.Signed = 8, "unsigned"
	case api.PrimitiveUint16:
		primitive.Width, primitive.Signed = 16, "unsigned"
	case api.PrimitiveUint32:
		primitive.Width, primitive.Signed = 32, "unsigned"
	case api.PrimitiveUint64:
		primitive.Width, primitive.Signed = 64, "unsigned"
	case api.PrimitiveInt:
		primitive.Role = "native-int"
		primitive.Width = uint8(abi.NativeIntegerWidth())
		primitive.Signed = "signed"
	case api.PrimitiveUint:
		primitive.Role = "native-uint"
		primitive.Width = uint8(abi.NativeIntegerWidth())
		primitive.Signed = "unsigned"
	case api.PrimitiveUintptr:
		primitive.Role = "uintptr"
		primitive.Width = uint8(abi.NativeIntegerWidth())
		primitive.Signed = "unsigned"
	default:
		return Primitive{}, &PrimitiveError{Alias: alias}
	}
	primitive.Shared = sharedPrimitive(alias, abi, carrier)
	if primitive.Shared != tsoniccore.SymbolInvalid {
		primitive.Width = 0
		primitive.Signed = ""
	}
	return primitive, nil
}

func (p Primitive) RequiresCompanion() bool {
	return p.Shared == tsoniccore.SymbolInvalid ||
		p.Role == "native-int" || p.Role == "native-uint" ||
		p.Role == "uintptr" || p.Role == "go-string"
}

func (p Primitive) SharedDeclaration() (tsoniccore.Declaration, bool, error) {
	if p.Shared == tsoniccore.SymbolInvalid {
		return tsoniccore.Declaration{}, false, nil
	}
	declaration, err := tsoniccore.Resolve(p.Shared)
	if err != nil {
		return tsoniccore.Declaration{}, false, err
	}
	return declaration, true, nil
}

func PrimitiveArguments(
	factory tsgo.Factory,
	primitive Primitive,
) ([]tsgo.Expression, error) {
	shared := ""
	declaration, selected, err := primitive.SharedDeclaration()
	if err != nil {
		return nil, err
	}
	if selected {
		shared = declaration.Module() + "#" + declaration.Export()
	}
	return []tsgo.Expression{
		factory.StringLiteral(runtimePrimitiveSchema, tsgo.TokenFlagsNone),
		factory.StringLiteral(primitive.Kind, tsgo.TokenFlagsNone),
		factory.StringLiteral(primitive.Role, tsgo.TokenFlagsNone),
		factory.StringLiteral(shared, tsgo.TokenFlagsNone),
		factory.NumericLiteral(strconv.FormatUint(uint64(primitive.Width), 10), tsgo.TokenFlagsNone),
		factory.StringLiteral(primitive.Signed, tsgo.TokenFlagsNone),
		factory.StringLiteral(primitive.Carrier, tsgo.TokenFlagsNone),
	}, nil
}

func sharedPrimitive(
	alias api.PrimitiveAlias,
	abi api.ScalarABI,
	carrier string,
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
		if carrier == "bigint" {
			return tsoniccore.SymbolInt64
		}
	case api.PrimitiveUint8:
		return tsoniccore.SymbolUint8
	case api.PrimitiveUint16:
		return tsoniccore.SymbolUint16
	case api.PrimitiveUint32:
		return tsoniccore.SymbolUint32
	case api.PrimitiveUint64:
		if carrier == "bigint" {
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
		if carrier == "bigint" {
			return tsoniccore.SymbolInt64
		}
	case api.PrimitiveUint, api.PrimitiveUintptr:
		if abi.NativeIntegerWidth() == api.NativeIntegerWidth32 {
			return tsoniccore.SymbolUint32
		}
		if carrier == "bigint" {
			return tsoniccore.SymbolUint64
		}
	}
	return tsoniccore.SymbolInvalid
}

func carrierName(keyword tsgo.KeywordTypeSyntaxKind) (string, error) {
	switch keyword {
	case tsgo.KeywordTypeSyntaxKindBooleanKeyword:
		return "boolean", nil
	case tsgo.KeywordTypeSyntaxKindStringKeyword:
		return "string", nil
	case tsgo.KeywordTypeSyntaxKindNumberKeyword:
		return "number", nil
	case tsgo.KeywordTypeSyntaxKindBigIntKeyword:
		return "bigint", nil
	default:
		return "", &PrimitiveError{}
	}
}

type PrimitiveError struct {
	Alias api.PrimitiveAlias
}

func (e *PrimitiveError) Error() string {
	return "describe Go source primitive: unsupported alias " + strconv.Itoa(int(e.Alias))
}
