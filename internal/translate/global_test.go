// Generated host-global capture proof (reviewer): a legal Go binding may
// carry the spelling of a JavaScript host global the emitter injects
// (Number, String, ...). Every injection is globalThis-qualified, so a
// source Number/String — local, parameter, or package-level — cannot
// capture the host conversion. These fixtures declare the colliding names
// and exercise the exact conversions; a bare injection would fail strict
// tsc (calling the local as a function) or diverge.
package translate_test

import "testing"

func TestOracleLocalNumberShadowsHostGlobal(t *testing.T) {
	// float64(int64) injects the host Number(); a local named Number must
	// not capture it.
	runOracle(t, `package fixture

func LocalNumberShadow() float64 {
	Number := 7
	value := int64(3)
	return float64(value) + float64(Number)
}
`)
}

func TestOracleParamNumberShadowsHostGlobal(t *testing.T) {
	runOracle(t, `package fixture

func widen(Number int64) float64 { return float64(Number) }

func ParamNumberShadow() float64 { return widen(9) }
`)
}

func TestOraclePackageNumberShadowsHostGlobal(t *testing.T) {
	runOracle(t, `package fixture

var Number int64 = 5

func PackageNumberShadow() float64 { return float64(Number) + 1.5 }
`)
}

func TestOraclePackageStringShadowsHostGlobal(t *testing.T) {
	// A struct used as a map key generates goKey$, which injects the host
	// String() over an integer field. A package-level value named String
	// must not capture it.
	runOracle(t, `package fixture

func String() int { return 100 }

type point struct{ n int }

func PackageStringShadow() int {
	m := map[point]int{{n: 5}: 42}
	return m[point{n: 5}] + String()
}
`)
}
