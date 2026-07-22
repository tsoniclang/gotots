package catalog

import "fmt"

// PredeclaredKind is the closed catalog of predeclared universe identifiers
// (types, constants, nil, and built-in functions). Values are explicit and
// permanent; the reconciliation gate joins this catalog bijectively against
// the toolchain's types.Universe scope.
type PredeclaredKind uint16

// PredeclaredClass is the closed class of one predeclared identifier.
type PredeclaredClass uint8

const (
	PredeclaredClassInvalid PredeclaredClass = iota
	PredeclaredClassType
	PredeclaredClassConstant
	PredeclaredClassNil
	PredeclaredClassFunction

	numPredeclaredClasses
)

var predeclaredClassNames = [numPredeclaredClasses]string{
	PredeclaredClassType: "type", PredeclaredClassConstant: "constant",
	PredeclaredClassNil: "nil", PredeclaredClassFunction: "function",
}

// Valid reports whether c names a class.
func (c PredeclaredClass) Valid() bool {
	return c > PredeclaredClassInvalid && c < numPredeclaredClasses
}

// String renders c for reports.
func (c PredeclaredClass) String() string {
	if c.Valid() {
		return predeclaredClassNames[c]
	}
	return fmt.Sprintf("catalog.PredeclaredClass(%d)", uint8(c))
}

// Explicit, permanent predeclared identities. Do not renumber; append only.
const (
	PredeclaredInvalid PredeclaredKind = 0

	PredeclaredAny        PredeclaredKind = 1
	PredeclaredBool       PredeclaredKind = 2
	PredeclaredByte       PredeclaredKind = 3
	PredeclaredComparable PredeclaredKind = 4
	PredeclaredComplex128 PredeclaredKind = 5
	PredeclaredComplex64  PredeclaredKind = 6
	PredeclaredError      PredeclaredKind = 7
	PredeclaredFloat32    PredeclaredKind = 8
	PredeclaredFloat64    PredeclaredKind = 9
	PredeclaredInt        PredeclaredKind = 10
	PredeclaredInt16      PredeclaredKind = 11
	PredeclaredInt32      PredeclaredKind = 12
	PredeclaredInt64      PredeclaredKind = 13
	PredeclaredInt8       PredeclaredKind = 14
	PredeclaredRune       PredeclaredKind = 15
	PredeclaredString     PredeclaredKind = 16
	PredeclaredUint       PredeclaredKind = 17
	PredeclaredUint16     PredeclaredKind = 18
	PredeclaredUint32     PredeclaredKind = 19
	PredeclaredUint64     PredeclaredKind = 20
	PredeclaredUint8      PredeclaredKind = 21
	PredeclaredUintptr    PredeclaredKind = 22

	PredeclaredTrue  PredeclaredKind = 23
	PredeclaredFalse PredeclaredKind = 24
	PredeclaredIota  PredeclaredKind = 25

	PredeclaredNil PredeclaredKind = 26

	PredeclaredAppend  PredeclaredKind = 27
	PredeclaredCap     PredeclaredKind = 28
	PredeclaredClear   PredeclaredKind = 29
	PredeclaredClose   PredeclaredKind = 30
	PredeclaredComplex PredeclaredKind = 31
	PredeclaredCopy    PredeclaredKind = 32
	PredeclaredDelete  PredeclaredKind = 33
	PredeclaredImag    PredeclaredKind = 34
	PredeclaredLen     PredeclaredKind = 35
	PredeclaredMake    PredeclaredKind = 36
	PredeclaredMax     PredeclaredKind = 37
	PredeclaredMin     PredeclaredKind = 38
	PredeclaredNew     PredeclaredKind = 39
	PredeclaredPanic   PredeclaredKind = 40
	PredeclaredPrint   PredeclaredKind = 41
	PredeclaredPrintln PredeclaredKind = 42
	PredeclaredReal    PredeclaredKind = 43
	PredeclaredRecover PredeclaredKind = 44

	// predeclaredCount is the highest assigned identity; append-only.
	predeclaredCount = 44
)

type predeclaredDescriptor struct {
	name  string
	class PredeclaredClass
}

var predeclaredTable = [predeclaredCount + 1]predeclaredDescriptor{
	PredeclaredAny: {"any", PredeclaredClassType}, PredeclaredBool: {"bool", PredeclaredClassType},
	PredeclaredByte: {"byte", PredeclaredClassType}, PredeclaredComparable: {"comparable", PredeclaredClassType},
	PredeclaredComplex128: {"complex128", PredeclaredClassType}, PredeclaredComplex64: {"complex64", PredeclaredClassType},
	PredeclaredError: {"error", PredeclaredClassType}, PredeclaredFloat32: {"float32", PredeclaredClassType},
	PredeclaredFloat64: {"float64", PredeclaredClassType}, PredeclaredInt: {"int", PredeclaredClassType},
	PredeclaredInt16: {"int16", PredeclaredClassType}, PredeclaredInt32: {"int32", PredeclaredClassType},
	PredeclaredInt64: {"int64", PredeclaredClassType}, PredeclaredInt8: {"int8", PredeclaredClassType},
	PredeclaredRune: {"rune", PredeclaredClassType}, PredeclaredString: {"string", PredeclaredClassType},
	PredeclaredUint: {"uint", PredeclaredClassType}, PredeclaredUint16: {"uint16", PredeclaredClassType},
	PredeclaredUint32: {"uint32", PredeclaredClassType}, PredeclaredUint64: {"uint64", PredeclaredClassType},
	PredeclaredUint8: {"uint8", PredeclaredClassType}, PredeclaredUintptr: {"uintptr", PredeclaredClassType},
	PredeclaredTrue: {"true", PredeclaredClassConstant}, PredeclaredFalse: {"false", PredeclaredClassConstant},
	PredeclaredIota:   {"iota", PredeclaredClassConstant},
	PredeclaredNil:    {"nil", PredeclaredClassNil},
	PredeclaredAppend: {"append", PredeclaredClassFunction}, PredeclaredCap: {"cap", PredeclaredClassFunction},
	PredeclaredClear: {"clear", PredeclaredClassFunction}, PredeclaredClose: {"close", PredeclaredClassFunction},
	PredeclaredComplex: {"complex", PredeclaredClassFunction}, PredeclaredCopy: {"copy", PredeclaredClassFunction},
	PredeclaredDelete: {"delete", PredeclaredClassFunction}, PredeclaredImag: {"imag", PredeclaredClassFunction},
	PredeclaredLen: {"len", PredeclaredClassFunction}, PredeclaredMake: {"make", PredeclaredClassFunction},
	PredeclaredMax: {"max", PredeclaredClassFunction}, PredeclaredMin: {"min", PredeclaredClassFunction},
	PredeclaredNew: {"new", PredeclaredClassFunction}, PredeclaredPanic: {"panic", PredeclaredClassFunction},
	PredeclaredPrint: {"print", PredeclaredClassFunction}, PredeclaredPrintln: {"println", PredeclaredClassFunction},
	PredeclaredReal: {"real", PredeclaredClassFunction}, PredeclaredRecover: {"recover", PredeclaredClassFunction},
}

// Valid reports whether k names a predeclared identifier.
func (k PredeclaredKind) Valid() bool { return k >= 1 && k <= predeclaredCount }

// Name is the source spelling.
func (k PredeclaredKind) Name() string {
	if !k.Valid() {
		return ""
	}
	return predeclaredTable[k].name
}

// Class is the identifier class.
func (k PredeclaredKind) Class() PredeclaredClass {
	if !k.Valid() {
		return PredeclaredClassInvalid
	}
	return predeclaredTable[k].class
}

// String renders k for reports.
func (k PredeclaredKind) String() string {
	if name := k.Name(); name != "" {
		return name
	}
	return fmt.Sprintf("catalog.PredeclaredKind(%d)", uint16(k))
}

// AllPredeclared returns every predeclared identifier in ascending identity
// order.
func AllPredeclared() []PredeclaredKind {
	out := make([]PredeclaredKind, 0, predeclaredCount)
	for id := 1; id <= predeclaredCount; id++ {
		out = append(out, PredeclaredKind(id))
	}
	return out
}
