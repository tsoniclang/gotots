// Kind predicates and source spans.
package ir

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
	return k == KindPointer || k == KindMap || k == KindSlice || k == KindFunc || k == KindIface || k == KindChan
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
