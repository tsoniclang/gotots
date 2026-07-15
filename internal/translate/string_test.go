// Byte-string contract tests: differential oracles for indexing, byte
// slicing (including mid-rune cuts), byte-wise ordering, non-UTF-8
// constants, rune-decoding range, bounds panics, and length under the
// one-code-unit-per-byte carrier.
package translate_test

import "testing"

func TestOracleStringByteSemantics(t *testing.T) {
	runOracle(t, `package fixture

func IndexBytes() (int, int, int) {
	s := "héllo"
	return int(s[0]), int(s[1]), int(s[2])
}

func NonUTF8Constant() (int, int) {
	s := "\xff\xfe"
	return len(s), int(s[0])
}

func MultibyteLen() (int, int, int) {
	ascii := "hello"
	accented := "héllo"
	emoji := "a😀b"
	return len(ascii), len(accented), len(emoji)
}

func SliceBytes() (string, string, string) {
	s := "hello world"
	return s[0:5], s[6:], s[:0]
}

func MidRuneSlice() (int, string) {
	s := "héllo"
	cut := s[0:2]
	return len(cut), cut
}

func MidRuneRoundTrip() bool {
	s := "€uro"
	return s[0:1]+s[1:3] == s[0:3]
}

func Ordering() (bool, bool, bool, bool) {
	return "abc" < "abd", "abc" < "abcd", "\xff" > "z", "é" > "e"
}

func Concat() (string, int) {
	s := "a" + "é" + "\xff"
	return s, len(s)
}
`)
}

func TestOracleStringRangeAndPanics(t *testing.T) {
	runOracle(t, `package fixture

func RangeRunes() (int, int, int) {
	total, count, lastIndex := 0, 0, 0
	for i, r := range "héllo" {
		total += int(r)
		count++
		lastIndex = i
	}
	return total, count, lastIndex
}

func RangeInvalidUTF8() (int, int) {
	replacements, count := 0, 0
	for _, r := range "a\xffb\xc3(" {
		if r == 0xFFFD {
			replacements++
		}
		count++
	}
	return replacements, count
}

func RangeEmoji() (int, int) {
	var indexes []int
	for i := range "a😀b" {
		indexes = append(indexes, i)
	}
	return len(indexes), indexes[2]
}

func RangeIndexOnly() int {
	total := 0
	for i := range "abc" {
		total += i
	}
	return total
}

func IndexPanic() byte {
	s := "abc"
	i := 5
	return s[i]
}

func IndexNegativePanic() byte {
	s := "abc"
	i := -1
	return s[i]
}

func SliceHighPanic() string {
	s := "abc"
	j := 9
	return s[1:j]
}

func SliceInvertedPanic() string {
	s := "abcdef"
	i, j := 4, 2
	return s[i:j]
}

func SwitchOnString() int {
	s := "b\xff"
	switch s {
	case "a":
		return 1
	case "b\xff":
		return 2
	}
	return 3
}

func MapWithByteKeys() (int, bool) {
	m := map[string]int{"\xff": 10, "k": 20}
	v, ok := m["\xff"]
	return v, ok
}
`)
}
