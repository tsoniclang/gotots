package translate_test

import "testing"

func TestOracleNamedCarrierTypes(t *testing.T) {
	runOracle(t, `package fixture

type Tristate int8

const (
	TristateUnknown Tristate = iota
	TristateFalse
	TristateTrue
)

func (t Tristate) IsTrue() bool {
	return t == TristateTrue
}

func (t Tristate) Negate() Tristate {
	switch t {
	case TristateTrue:
		return TristateFalse
	case TristateFalse:
		return TristateTrue
	}
	return t
}

type Flags uint32

func (f Flags) Has(mask Flags) bool {
	return f&mask != 0
}

type Name string

func (n Name) Empty() bool {
	return n == ""
}

func CarrierMethods() (bool, bool, bool, bool) {
	state := TristateTrue
	return state.IsTrue(), state.Negate().IsTrue(),
		Flags(6).Has(Flags(2)), Name("x").Empty()
}

func CarrierConversions() (bool, string) {
	raw := "hello"
	named := Name(raw)
	back := string(named)
	return named.Empty(), back
}

func CarrierArithmetic() (int8, uint32) {
	t := TristateTrue
	t = t + 1
	f := Flags(1)
	f = f << 3
	return int8(t), uint32(f)
}

func CarrierZero() (bool, bool) {
	var t Tristate
	var n Name
	return t == TristateUnknown, n.Empty()
}
`)
}

func TestOracleBuiltinStatements(t *testing.T) {
	runOracle(t, `package fixture

func PanicString() int {
	panic("boom with spaces")
}

func PanicInt() int {
	code := 42
	panic(code)
}

func PanicInt8() int {
	var code int8 = -7
	panic(code)
}

func PanicBool() int {
	panic(true)
}

func ClearMap() (int, int) {
	m := map[string]int{"a": 1, "b": 2}
	before := len(m)
	clear(m)
	return before, len(m)
}

func ClearNilMap() int {
	var m map[string]int
	clear(m)
	return len(m)
}

func CopySlices() (int, int, int, int) {
	src := []int{1, 2, 3}
	dst := make([]int, 2)
	n := copy(dst, src)
	return int(n), dst[0], dst[1], src[2]
}

func CopyOverlapping() (int, int, int, int) {
	s := []int{1, 2, 3, 4}
	copy(s[1:], s[0:3])
	return s[0], s[1], s[2], s[3]
}

func CopyDiscarded() int {
	dst := make([]int, 1)
	copy(dst, []int{9, 8})
	return dst[0]
}
`)
}

func TestOracleRangeIntAndAppendSpread(t *testing.T) {
	runOracle(t, `package fixture

func RangeInt() (int, int) {
	total := 0
	iterations := 0
	for i := range 5 {
		total = total + i
		iterations++
	}
	return total, iterations
}

func RangeIntNoVar() int {
	count := 0
	for range 3 {
		count++
	}
	return count
}

func RangeIntZeroAndNegative() (int, int) {
	a, b := 0, 0
	for range 0 {
		a++
	}
	n := -4
	for range n {
		b++
	}
	return a, b
}

func rangeInt32(n int32) int32 {
	var total int32
	for i := range n {
		total = total + i
	}
	return total
}

func RangeNarrow() int32 {
	return rangeInt32(4)
}

func AppendSpread() (int, int, int) {
	a := []int{1, 2}
	b := []int{3, 4, 5}
	a = append(a, b...)
	return len(a), a[2], a[4]
}

func AppendSpreadNil() (int, int) {
	var a []int
	var b []int
	a = append(a, b...)
	c := append([]int{7}, a...)
	return len(a), c[0]
}

func AppendSelf() (int, int, int) {
	s := make([]int, 2, 8)
	s[0], s[1] = 1, 2
	s = append(s, s...)
	return len(s), s[2], s[3]
}
`)
}

func TestOracleStructElementBuiltins(t *testing.T) {
	runOracle(t, `package fixture

type Point struct {
	X int
	Y int
}

func CopyStructElements() (int, int, int) {
	src := []Point{{X: 1, Y: 1}, {X: 2, Y: 2}}
	dst := make([]Point, 2)
	alias := dst[0:1]
	n := copy(dst, src)
	src[0].X = 99
	return int(n), dst[0].X, alias[0].X
}

func AppendSpreadStructCopies() (int, int) {
	src := []Point{{X: 5, Y: 5}}
	var dst []Point
	dst = append(dst, src...)
	src[0].X = 77
	return dst[0].X, src[0].X
}
`)
}
