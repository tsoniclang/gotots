package semantic

import "fmt"

type TypeKind uint8

const (
	TypeInvalid TypeKind = iota
	TypeBasic
	TypeNamed
	TypeAlias
	TypeParameter
	TypePointer
	TypeSlice
	TypeArray
	TypeMap
	TypeChannel
	TypeSignature
	TypeStruct
	TypeInterface
	TypeTuple
	TypeUnion

	typeKindCount = TypeUnion
)

func (kind TypeKind) Valid() bool {
	return kind > TypeInvalid && kind <= typeKindCount
}

type BasicKind uint8

const (
	BasicInvalid BasicKind = iota
	BasicBool
	BasicInt
	BasicInt8
	BasicInt16
	BasicInt32
	BasicInt64
	BasicUint
	BasicUint8
	BasicUint16
	BasicUint32
	BasicUint64
	BasicUintptr
	BasicFloat32
	BasicFloat64
	BasicComplex64
	BasicComplex128
	BasicString
	BasicUnsafePointer
	BasicUntypedBool
	BasicUntypedInt
	BasicUntypedRune
	BasicUntypedFloat
	BasicUntypedComplex
	BasicUntypedString
	BasicUntypedNil

	basicKindCount = BasicUntypedNil
)

func (kind BasicKind) Valid() bool {
	return kind > BasicInvalid && kind <= basicKindCount
}

type ChannelDirection uint8

const (
	ChannelInvalid ChannelDirection = iota
	ChannelSendReceive
	ChannelSendOnly
	ChannelReceiveOnly
)

func (direction ChannelDirection) Valid() bool {
	return direction >= ChannelSendReceive &&
		direction <= ChannelReceiveOnly
}

func (kind TypeKind) String() string {
	if !kind.Valid() {
		return fmt.Sprintf("semantic.TypeKind(%d)", uint8(kind))
	}
	return typeKindNames[kind]
}

var typeKindNames = [typeKindCount + 1]string{
	TypeBasic:     "basic",
	TypeNamed:     "named",
	TypeAlias:     "alias",
	TypeParameter: "type-parameter",
	TypePointer:   "pointer",
	TypeSlice:     "slice",
	TypeArray:     "array",
	TypeMap:       "map",
	TypeChannel:   "channel",
	TypeSignature: "signature",
	TypeStruct:    "struct",
	TypeInterface: "interface",
	TypeTuple:     "tuple",
	TypeUnion:     "union",
}
