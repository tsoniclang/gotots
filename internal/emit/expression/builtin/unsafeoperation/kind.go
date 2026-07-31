package unsafeoperation

import "go/types"

type Kind uint8

const (
	Invalid Kind = iota
	String
	Slice
	StringData
	SliceData
	Sizeof
	Alignof
	Offsetof
)

var objectKinds = map[types.Object]Kind{
	types.Unsafe.Scope().Lookup("String"):     String,
	types.Unsafe.Scope().Lookup("Slice"):      Slice,
	types.Unsafe.Scope().Lookup("StringData"): StringData,
	types.Unsafe.Scope().Lookup("SliceData"):  SliceData,
	types.Unsafe.Scope().Lookup("Sizeof"):     Sizeof,
	types.Unsafe.Scope().Lookup("Alignof"):    Alignof,
	types.Unsafe.Scope().Lookup("Offsetof"):   Offsetof,
}

func Classify(builtin *types.Builtin) Kind {
	if builtin == nil {
		return Invalid
	}
	return objectKinds[builtin]
}

func (k Kind) Runtime() bool {
	return k >= String && k <= SliceData
}

func (k Kind) Constant() bool {
	return k >= Sizeof && k <= Offsetof
}
