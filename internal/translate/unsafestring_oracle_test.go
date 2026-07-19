package translate_test

import "testing"

func TestOracleUnsafeStringExactAndArraySwitch(t *testing.T) {
	// The tspath/vfs shapes: unsafe.String(&b[0], len(b)) is exactly
	// string(b) (the unsafe form only avoids a copy our carrier makes
	// anyway), and a [2]byte switch tag lowers to the boolean-switch
	// form over element-wise equality (the BOM sniff in decodeBytes).
	runOracle(t, `package fixture

import "unsafe"

func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

func sniff(bom [2]byte) int {
	switch bom {
	case [2]byte{0xFF, 0xFE}:
		return 1
	case [2]byte{0xFE, 0xFF}:
		return 2
	case [2]byte{0xEF, 0xBB}:
		return 3
	}
	return 0
}

func UnsafeStringAndArraySwitch() int {
	total := 0
	if bytesToString([]byte{0x68, 0x69}) == "hi" {
		total += 1
	}
	if bytesToString(nil) == "" {
		total += 10
	}
	if bytesToString([]byte{0xFF, 0x00, 0x41}) == "\xff\x00A" {
		total += 100
	}
	if sniff([2]byte{0xFF, 0xFE}) == 1 {
		total += 1000
	}
	if sniff([2]byte{0xFE, 0xFF}) == 2 {
		total += 10000
	}
	if sniff([2]byte{0x00, 0x42}) == 0 {
		total += 100000
	}
	return total
}
`)
}
