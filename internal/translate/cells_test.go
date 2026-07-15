// Boxed-variable cell contract tests: pointers to scalars, strings,
// slices, and nested pointers via address-taken locals and parameters,
// aliasing through calls and closures, pointer identity, nil panics,
// and new(T) cells.
package translate_test

import "testing"

func TestOraclePointerCells(t *testing.T) {
	runOracle(t, `package fixture

func addOne(p *int) {
	*p = *p + 1
}

func AliasThroughCall() int {
	x := 10
	addOne(&x)
	addOne(&x)
	return x
}

func boxedParam(start int) *int {
	start = start + 5
	return &start
}

func EscapingParam() (int, int) {
	p := boxedParam(1)
	q := boxedParam(1)
	*p = *p + 100
	return *p, *q
}

func PointerIdentity() (bool, bool) {
	x := 1
	y := 1
	p1 := &x
	p2 := &x
	q := &y
	return p1 == p2, p1 == q
}

func StringCell() string {
	s := "start"
	mutate := func(p *string) {
		*p = *p + "+mutated"
	}
	mutate(&s)
	return s
}

func SliceCell() (int, int) {
	values := []int{1, 2}
	grow := func(p *[]int) {
		*p = append(*p, 99)
	}
	grow(&values)
	grow(&values)
	return len(values), values[3]
}

func PointerToPointer() int {
	x := 7
	p := &x
	pp := &p
	**pp = 42
	return x
}

func NilCellDerefPanics() int {
	var p *int
	return *p
}

func NewCell() (int, int) {
	p := new(int)
	q := new(int)
	*p = 3
	return *p, *q
}

func ClosureSharesCell() (int, int) {
	count := 0
	p := &count
	bump := func() {
		count = count + 1
	}
	bump()
	*p = *p + 10
	bump()
	return count, *p
}

func CompoundOnBoxed() int {
	x := 5
	p := &x
	x += 2
	*p = *p * 3
	return x
}

func BoolAndFloatCells() (bool, float64) {
	b := false
	f := 1.5
	set := func(pb *bool, pf *float64) {
		*pb = true
		*pf = *pf * 2
	}
	set(&b, &f)
	return b, f
}
`)
}
