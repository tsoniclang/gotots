package translate_test

import "testing"

func TestOracleParamReprConversions(t *testing.T) {
	// The checker hashWrite32/hashWrite64 shapes: conversions OUT of a
	// carrier-uniform parameter (~int32|~uint32 share the number carrier;
	// the 64-bit set shares bigint) are target-driven wraps — one static
	// function for every possible binding, including sign reinterpretation.
	runOracle(t, `package fixture

type Flags uint32

func reinterpret32[T ~int32 | ~uint32](value T) uint32 {
	return uint32(value)
}

func low32of64[T ~int | ~uint | ~int64 | ~uint64](value T) uint32 {
	return uint32(uint64(value) & 0xFFFFFFFF)
}

func ParamReprConversionsOut() int {
	total := 0
	if reinterpret32(int32(-1)) == 4294967295 {
		total += 1
	}
	if reinterpret32(uint32(7)) == 7 {
		total += 10
	}
	if reinterpret32(Flags(9)) == 9 {
		total += 100
	}
	if low32of64(int64(-1)) == 4294967295 {
		total += 1000
	}
	if low32of64(uint64(0x1_0000_0005)) == 5 {
		total += 10000
	}
	if low32of64(int(70000)) == 70000 {
		total += 100000
	}
	return total
}
`)
}

func TestOracleParamReprExactInAndCompound(t *testing.T) {
	// The tsoptions floatOrInt32ToFlag and ast getCombinedFlags shapes:
	// an exact single-kind constraint admits conversions INTO the
	// parameter (float64 truncation toward zero) and compound bitwise
	// operations computed at the carrier.
	runOracle(t, `package fixture

type WatchKind int32

func fromFloat[T ~int32](value float64) T {
	return T(value)
}

func orAll[T ~uint32](values []T) T {
	var flags T
	for _, v := range values {
		flags |= v
	}
	return flags
}

func ParamReprIntoAndCompound() int {
	total := 0
	if fromFloat[WatchKind](7.9) == 7 {
		total += 1
	}
	if fromFloat[WatchKind](-3.2) == -3 {
		total += 10
	}
	if fromFloat[int32](2147483600.0) == 2147483600 {
		total += 100
	}
	if orAll([]uint32{1, 4, 16}) == 21 {
		total += 1000
	}
	combined := orAll([]uint32{0x8000_0000, 1})
	if combined == 0x8000_0001 {
		total += 10000
	}
	return total
}
`)
}
