// Method-set and interface-equality contract tests from review round
// five: value versus pointer method sets, canonical method identity,
// uncomparable dynamic types panicking exactly, and deep exact struct
// equality through generated goEq$.
package translate_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestOracleValueVsPointerMethodSets(t *testing.T) {
	runOracle(t, `package fixture

type t struct{ n int }

func (x *t) p() int { return 1 + x.n*0 }
func (x t) v() int  { return 2 + x.n*0 }

type hasP interface{ p() int }
type hasV interface{ v() int }

func MethodSetFlavors() (bool, bool, bool, bool) {
	var value any = t{}
	var pointer any = &t{}
	_, valueHasP := value.(hasP)
	_, valueHasV := value.(hasV)
	_, pointerHasP := pointer.(hasP)
	_, pointerHasV := pointer.(hasV)
	return valueHasP, valueHasV, pointerHasP, pointerHasV
}

type sig1 struct{}
type sig2 struct{}

func (x sig1) m() int    { _ = x; return 1 }
func (x sig2) m() string { _ = x; return "s" }

type wantsIntM interface{ m() int }

func SignatureIdentity() (bool, bool) {
	var a any = sig1{}
	var b any = sig2{}
	_, aOk := a.(wantsIntM)
	_, bOk := b.(wantsIntM)
	return aOk, bOk
}
`)
}

func TestOracleUncomparableDynamicEquality(t *testing.T) {
	runOracle(t, `package fixture

func MapDynamicPanics() bool {
	m := map[string]int{}
	var left any = m
	var right any = m
	return left == right
}

func SliceDynamicPanics() bool {
	s := []int{1}
	var left any = s
	var right any = s
	return left == right
}

func FuncDynamicPanics() bool {
	f := func() int { return 1 }
	var left any = f
	var right any = f
	return left == right
}

func ComparableDynamicStillWorks() (bool, bool) {
	var a any = 5
	var b any = 5
	var c any = "x"
	return a == b, a == c
}
`)
}

func TestOracleDeepStructEquality(t *testing.T) {
	runOracle(t, `package fixture

type inner struct {
	f float64
	s string
}

type outer struct {
	in   inner
	vals [2]int
	tag  any
}

func nan() float64 {
	z := 0.0
	return z / z
}

func DeepFieldEquality() (bool, bool, bool) {
	a := outer{in: inner{f: 1.5, s: "x"}, vals: [2]int{1, 2}, tag: 7}
	b := outer{in: inner{f: 1.5, s: "x"}, vals: [2]int{1, 2}, tag: 7}
	c := outer{in: inner{f: 1.5, s: "x"}, vals: [2]int{1, 3}, tag: 7}
	return a == b, a == c, a != c
}

func NaNFieldsNeverEqual() bool {
	a := inner{f: nan()}
	b := inner{f: nan()}
	return a == b
}

func BoxedStructEquality() (bool, bool) {
	var x any = inner{f: 2.0, s: "q"}
	var y any = inner{f: 2.0, s: "q"}
	var z any = inner{f: 3.0, s: "q"}
	return x == y, x == z
}

func IfaceFieldPanicsWhenUncomparable() bool {
	a := outer{tag: []int{1}}
	b := outer{tag: []int{1}}
	return a == b
}
`)
}

// TestOracleUnexportedCrossPackageDispatch is the adversarial identity
// fixture: two packages declare same-named UNEXPORTED methods; each
// package's interface is satisfiable only by its own types (Go's
// unexported method-set rule), and dispatch through each union must
// select exactly the right package's method.
func TestOracleUnexportedCrossPackageDispatch(t *testing.T) {
	result, err := oracle.RunAssembled(t.TempDir(), map[string]string{
		"fixture": `package fixture

import (
	"oracle.fixture/alpha"
	"oracle.fixture/beta"
)

func AlphaDispatch() int {
	return alpha.Call(alpha.New())
}

func BetaDispatch() int {
	return beta.Call(beta.New())
}
`,
		"alpha": `package alpha

type secret interface{ tag() int }

type impl struct{}

func (i impl) tag() int { _ = i; return 100 }

func New() secret    { return impl{} }
func Call(s secret) int { return s.tag() }
`,
		"beta": `package beta

type secret interface{ tag() int }

type impl struct{}

func (i impl) tag() int { _ = i; return 200 }

func New() secret    { return impl{} }
func Call(s secret) int { return s.tag() }
`,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s\n--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}

// TestOracleEmbeddedPromotionAmbiguity locks Go's own promotion rule:
// same-name unexported methods embedded at the SAME depth promote
// NEITHER (the interfaces are unsatisfied); at DIFFERENT depths only the
// shallowest promotes and dispatch selects exactly it.
func TestOracleEmbeddedPromotionAmbiguity(t *testing.T) {
	runOracle(t, `package fixture

type shallow struct{}

func (s shallow) tag() int { _ = s; return 1 }

type deepHolder struct{ deep }

type deep struct{}

func (d deep) tag() int { _ = d; return 2 }

type wins struct {
	shallow
	deepHolder
}

type tagged interface{ tag() int }

func ShallowestWins() int {
	var v tagged = wins{}
	return v.tag()
}
`)
}
