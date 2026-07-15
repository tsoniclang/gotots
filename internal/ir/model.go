// Package ir is the typed body intermediate representation: Go semantic
// operations with explicit types and evaluation order, built from go/ast
// plus go/types evidence and consumed by representation planning and
// lowering.
//
// The IR is closed over the reviewed semantic subset. Constructs outside
// it fail with stable GOTOTS_UNSUPPORTED_* diagnostics carrying the exact
// source span — they are never approximated, skipped, or passed through.
package ir

import "fmt"

// Kind is the exact Go semantic type class of an IR value.
type Kind int

const (
	KindInvalid Kind = iota
	KindBool
	KindString
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindInt // 64-bit under the linux-amd64 profile
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindUint // 64-bit under the linux-amd64 profile
	KindUintptr
	KindFloat32
	KindFloat64
	// KindPointer is a pointer to a named struct of the translated unit,
	// carried as direct object identity with undefined for nil.
	KindPointer
	// KindStruct is a named struct type of the translated unit; values of
	// this kind appear only behind pointers and as receivers in the
	// reviewed subset (value copies have their own future lowering).
	KindStruct
	// KindMap is a Go map, carried as Map | undefined with exact nil, zero,
	// comma-ok, and write-panic behavior through the language ABI.
	KindMap
	// KindSlice is a Go slice; its representation is selected per value
	// flow by the planner (native array or GoSliceView).
	KindSlice
	// KindFunc is a first-class function value, carried as a JS closure
	// with undefined for nil. Go and JS both capture variables by
	// reference, so closure semantics coincide.
	KindFunc
)

// Type is the resolved semantic type of an IR value with its canonical Go
// spelling for provenance. Composite kinds carry their structure.
type Type struct {
	Kind Kind
	Go   string // canonical Go type string
	// Named is the struct type name for KindStruct and for the element of
	// a KindPointer.
	Named string
	// Pkg is the Go package path declaring Named. The emitter compares it
	// against the emitting module to select local or imported spelling.
	Pkg string
	// Elem is the pointee (KindPointer), value (KindMap), or element
	// (KindSlice) type.
	Elem *Type
	// Key is the key type of a KindMap.
	Key *Type
	// Sig is the signature of a KindFunc.
	Sig *FuncSig
}

// FuncSig is the shape of a function value's type.
type FuncSig struct {
	Params  []Type
	Results []Type
}

// Scope is the set of Go package paths translated together as one unit.
// References into any scope package resolve to generated modules; every
// reference outside the scope fails closed.
type Scope map[string]bool

// Signed reports whether the kind is a signed integer.
func (k Kind) Signed() bool {
	switch k {
	case KindInt8, KindInt16, KindInt32, KindInt64, KindInt:
		return true
	}
	return false
}

// Unsigned reports whether the kind is an unsigned integer.
func (k Kind) Unsigned() bool {
	switch k {
	case KindUint8, KindUint16, KindUint32, KindUint64, KindUint, KindUintptr:
		return true
	}
	return false
}

// Integer reports whether the kind is any integer.
func (k Kind) Integer() bool { return k.Signed() || k.Unsigned() }

// Wide64 reports whether the kind needs a 64-bit exact carrier.
func (k Kind) Wide64() bool {
	switch k {
	case KindInt64, KindInt, KindUint64, KindUint, KindUintptr:
		return true
	}
	return false
}

// Float reports whether the kind is a floating-point type.
func (k Kind) Float() bool { return k == KindFloat32 || k == KindFloat64 }

// Nilable reports whether the kind's zero value is Go nil (carried as
// undefined).
func (k Kind) Nilable() bool {
	return k == KindPointer || k == KindMap || k == KindSlice || k == KindFunc
}

// Bits returns the integer width in bits.
func (k Kind) Bits() int {
	switch k {
	case KindInt8, KindUint8:
		return 8
	case KindInt16, KindUint16:
		return 16
	case KindInt32, KindUint32:
		return 32
	case KindInt64, KindInt, KindUint64, KindUint, KindUintptr:
		return 64
	}
	return 0
}

// Span is an exact source location.
type Span struct {
	File string
	Line int
	Col  int
}

// Func is one translated function or method.
type Func struct {
	ID       string // census declaration identity
	Package  string
	Name     string
	Exported bool
	Span     Span
	// Receiver is set for methods: a pointer-to-struct parameter bound to
	// the generated class instance.
	Receiver *Var
	Params   []Var
	Results  []Var
	Body     *Block
	// BodyHash matches the census body record for drift detection.
	BodyHash string
	// Operations is the sorted set of IR operation names the body uses,
	// recorded in the proof chain.
	Operations []string
}

// Var is a parameter, result, or local.
type Var struct {
	Name string
	Type Type
}

// Struct is one named struct type of the translated package, generated as
// a class with ordered typed fields.
type Struct struct {
	ID       string
	Name     string
	Exported bool
	Span     Span
	Fields   []Var
	Methods  []*Func
}

// Unsupported is the stable fail-closed diagnostic for a construct outside
// the reviewed subset.
type Unsupported struct {
	Code      string // GOTOTS_UNSUPPORTED_{STATEMENT,EXPRESSION,TYPE,DECLARATION,OPERATION}
	Construct string
	Span      Span
}

func (u *Unsupported) Error() string {
	return fmt.Sprintf("%s:\n%s at %s:%d:%d", u.Code, u.Construct, u.Span.File, u.Span.Line, u.Span.Col)
}
