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

// A method present by NAME but with the WRONG SIGNATURE does not satisfy
// the target interface; Go still reports "missing method Convert". The
// diagnostic compares canonical method IDENTITIES (name + signature), not
// names, so it reproduces Go's exact message rather than treating the
// wrong-signature Convert as present.
func TestOracleFailedAssertionWrongSignatureIsMissing(t *testing.T) {
	runOracle(t, `package fixture

type Wanted interface{ Convert(int) string }

type Wrong struct{}

func (Wrong) Convert(s string) string { return s }

func FailedAssert() string {
	var x any = Wrong{}
	w := x.(Wanted)
	return w.Convert(1)
}
`)
}
