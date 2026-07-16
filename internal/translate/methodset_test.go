// Method-set and interface-equality contract tests from review round
// five: value versus pointer method sets, canonical method identity,
// uncomparable dynamic types panicking exactly, and deep exact struct
// equality through generated goEq$.
package translate_test

import "testing"

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
