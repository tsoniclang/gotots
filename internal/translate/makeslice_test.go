// make() and native-slice contract tests from review round five: exact
// makeslice panics (len/cap out of range), capacity-before-length
// evaluation, native []byte/[]rune conversions, and distinct fresh
// zeros for external and struct slice elements.
package translate_test

import "testing"

func TestOracleMakeSlicePanics(t *testing.T) {
	runOracle(t, `package fixture

func NegativeLengthPanics() int {
	n := -1
	s := make([]int, n)
	return len(s)
}

func CapBelowLengthPanics() int {
	l, c := 3, 1
	s := make([]int, l, c)
	return len(s)
}

func NegativeCapPanics() int {
	l, c := 1, -1
	s := make([]int, l, c)
	return len(s)
}

var order string

func length() int  { order += "L"; return 2 }
func capacity() int { order += "C"; return 5 }

func CapEvaluatedBeforeLength() (string, int, int) {
	order = ""
	s := make([]int, length(), capacity())
	s[0] = 1
	total := 0
	for _, v := range s {
		total += v
	}
	return order, len(s), total
}

func ValidNativeMake() (int, int) {
	s := make([]int, 3)
	s[0] = 7
	s = append(s, 9)
	return len(s), s[0] + s[3]
}
`)
}

func TestOracleNativeBytesConversion(t *testing.T) {
	runOracle(t, `package fixture

func NativeBytesToString() (string, int) {
	bytes := []byte{71, 111}
	bytes[0] = 103
	s := string(bytes)
	return s, len(bytes)
}

func NativeRunesToString() (string, int) {
	runes := []rune{'g', 'o'}
	runes[0] = 'G'
	s := string(runes)
	return s, len(runes)
}
`)
}

func TestOracleDistinctSliceZeros(t *testing.T) {
	runOracle(t, `package fixture

type cell struct{ n int }

func StructElementsDistinct() (int, int) {
	cells := make([]cell, 2)
	cells[0].n = 5
	return cells[0].n, cells[1].n
}

func StructMakeCapDistinct() (int, int) {
	cells := make([]cell, 2, 4)
	cells[1].n = 9
	return cells[0].n, cells[1].n
}
`)
}
