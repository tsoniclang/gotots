// A failed interface-to-interface assertion panics with Go's exact
// message, including the ": missing method X" suffix. Go names the FIRST
// method the dynamic type lacks in the target interface's sorted
// method-set order; the differential oracle compares the panic message
// verbatim, so these pin both the suffix and its selection order.
package translate_test

import "testing"

func TestOracleFailedAssertionNamesMissingMethod(t *testing.T) {
	runOracle(t, `package fixture

type Reader interface{ Read() int }
type ReadWriter interface {
	Read() int
	Write() int
}

type onlyReader struct{ n int }

func (o onlyReader) Read() int { return o.n }

func FailedAssert() int {
	var r Reader = onlyReader{n: 5}
	rw := r.(ReadWriter)
	return rw.Write()
}
`)
}

func TestOracleFailedAssertionMissingMethodOrder(t *testing.T) {
	runOracle(t, `package fixture

type Base interface{ Alpha() int }
type Full interface {
	Alpha() int
	Beta() int
	Zeta() int
}

type hasAlphaZeta struct{}

func (hasAlphaZeta) Alpha() int { return 1 }
func (hasAlphaZeta) Zeta() int  { return 2 }

func MissingMiddle() int {
	var b Base = hasAlphaZeta{}
	return b.(Full).Beta()
}

type hasNothingExtra struct{}

func (hasNothingExtra) Alpha() int { return 9 }

func MissingTwo() int {
	var b Base = hasNothingExtra{}
	return b.(Full).Zeta()
}
`)
}
