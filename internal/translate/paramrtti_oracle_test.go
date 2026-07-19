package translate_test

import "testing"

func TestOracleParamRttiBoxing(t *testing.T) {
	// The checker asRecursionId and diagnosticwriter ToDiagnostics
	// shapes: a bare type-parameter value boxed into an interface with
	// the binding's runtime identity (the rt$P triple each call site
	// passes). Type asserts and interface equality must observe the
	// binding's exact dynamic type.
	runOracle(t, `package fixture

type node struct {
	id int
}

type recursionID struct {
	value any
}

func asID[T *node | int](value T) recursionID {
	return recursionID{value: value}
}

func boxAll[T any](values []T) []any {
	result := make([]any, len(values))
	for i, v := range values {
		result[i] = v
	}
	return result
}

func ParamRttiBoxing() int {
	total := 0
	n := &node{id: 7}
	a := asID(n)
	bID := asID(n)
	if a == bID {
		total += 1
	}
	if a != asID(&node{id: 7}) {
		total += 10
	}
	if p, ok := a.value.(*node); ok && p.id == 7 {
		total += 100
	}
	c := asID(41)
	if v, ok := c.value.(int); ok && v == 41 {
		total += 1000
	}
	if c == asID(41) {
		total += 10000
	}
	boxed := boxAll([]string{"x", "y"})
	if s, ok := boxed[1].(string); ok && s == "y" {
		total += 100000
	}
	ints := boxAll([]int{5})
	if v, ok := ints[0].(int); ok && v == 5 {
		total += 1000000
	}
	return total
}
`)
}

func TestOracleParamRttiForwarding(t *testing.T) {
	// A generic caller forwarding its own parameter into an
	// rtti-required position passes its rt$P slot through (the
	// propagated requirement), and interface method dispatch through
	// the boxed value selects the binding's method.
	runOracle(t, `package fixture

type shape interface {
	area() int
}

type square struct {
	side int
}

func (s square) area() int {
	return s.side * s.side
}

type line struct {
	length int
}

func (l line) area() int {
	return 0
}

func toShape[T shape](v T) shape {
	return v
}

func viaForward[T shape](v T) shape {
	return toShape(v)
}

func ParamRttiForwarding() int {
	total := 0
	s := viaForward(square{side: 3})
	if s.area() == 9 {
		total += 1
	}
	l := viaForward(line{length: 5})
	if l.area() == 0 {
		total += 10
	}
	if _, ok := s.(square); ok {
		total += 100
	}
	return total
}
`)
}
