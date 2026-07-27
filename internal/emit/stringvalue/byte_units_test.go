package stringvalue

import "testing"

func TestByteCodeUnitsPreservesEveryByteExactly(t *testing.T) {
	source := make([]byte, 256)
	for index := range source {
		source[index] = byte(index)
	}
	target := []rune(byteCodeUnits(string(source)))
	if len(target) != len(source) {
		t.Fatalf("target code units = %d, want %d", len(target), len(source))
	}
	for index, value := range source {
		if target[index] != rune(value) {
			t.Fatalf("target code unit %d = %#x, want %#x", index, target[index], value)
		}
	}
}
