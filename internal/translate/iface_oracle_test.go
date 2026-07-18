package translate_test

import "testing"

func TestOracleInterfaceValues(t *testing.T) {
	runOracle(t, `package fixture

type Shape interface {
	Area() int
	Name() string
}

type Square struct {
	Side int
}

func (s *Square) Area() int {
	return s.Side * s.Side
}

func (s *Square) Name() string {
	return "square"
}

type Circle struct {
	R int
}

func (c *Circle) Area() int {
	return 3 * c.R * c.R
}

func (c *Circle) Name() string {
	return "circle"
}

func Dispatch() (int, string, int, string) {
	var s Shape = &Square{Side: 3}
	var c Shape = &Circle{R: 2}
	return s.Area(), s.Name(), c.Area(), c.Name()
}

func DispatchThroughSlice() int {
	shapes := []Shape{&Square{Side: 2}, &Circle{R: 1}}
	total := 0
	for _, shape := range shapes {
		total = total + shape.Area()
	}
	return total
}

func NilInterface() (bool, bool) {
	var s Shape
	var direct any
	return s == nil, direct == nil
}

// NilPointerInInterface proves the trap: an interface holding a nil
// pointer is not the nil interface.
func NilPointerInInterface() (bool, bool) {
	var p *Square
	var s Shape = p
	return s == nil, p == nil
}

func NilInterfaceCallPanics() int {
	var s Shape
	return s.Area()
}

func interfaceArgsAndResults(s Shape) Shape {
	return s
}

func PassThrough() string {
	result := interfaceArgsAndResults(&Circle{R: 5})
	return result.Name()
}

func AnyBoxing() (bool, bool) {
	var a any = 42
	var b any = "text"
	return a != nil, b != nil
}
`)
}

func TestOracleTypeAssertions(t *testing.T) {
	runOracle(t, `package fixture

type Animal interface {
	Sound() string
}

type Dog struct {
	Loud bool
}

func (d *Dog) Sound() string {
	return "woof"
}

type Cat struct {
	Naps int
}

func (c *Cat) Sound() string {
	return "meow"
}

func AssertHit() (string, bool) {
	var a Animal = &Dog{Loud: true}
	d := a.(*Dog)
	viaOk, ok := a.(*Dog)
	_ = viaOk
	return d.Sound(), ok
}

func AssertMiss() (bool, int) {
	var a Animal = &Dog{Loud: false}
	c, ok := a.(*Cat)
	if c != nil {
		return ok, -1
	}
	return ok, 0
}

// AssertMissPanics proves Go's exact interface-conversion message.
func AssertMissPanics() string {
	var a Animal = &Dog{Loud: false}
	c := a.(*Cat)
	return c.Sound()
}

func AssertNilPanics() string {
	var a Animal
	d := a.(*Dog)
	return d.Sound()
}

func AssertScalars() (int, string, bool, bool) {
	var a any = 7
	var b any = "words"
	n := a.(int)
	s := b.(string)
	_, notString := a.(string)
	_, isInt := a.(int)
	return n, s, notString, isInt
}

// AssertScalarMismatchPanics proves the message for predeclared types
// under an empty-interface source.
func AssertScalarMismatchPanics() int {
	var a any = "not a number"
	return a.(int)
}

type Meters int

func AssertNamedCarrier() (bool, bool) {
	var a any = Meters(5)
	_, isMeters := a.(Meters)
	_, isInt := a.(int)
	return isMeters, isInt
}
`)
}

func TestOracleTypeSwitch(t *testing.T) {
	runOracle(t, `package fixture

type Token interface {
	Kind() string
}

type Word struct {
	Text string
}

func (w *Word) Kind() string { return "word" }

type Number struct {
	Value int
}

func (n *Number) Kind() string { return "number" }

type Space struct {
	Width int
}

func (s *Space) Kind() string { return "space" }

func classify(t Token) string {
	switch v := t.(type) {
	case *Word:
		return "word:" + v.Text
	case *Number:
		if v.Value > 10 {
			return "big"
		}
		return "small"
	case nil:
		return "nil"
	default:
		return "other:" + v.Kind()
	}
}

func Classify() (string, string, string, string, string) {
	return classify(&Word{Text: "hi"}),
		classify(&Number{Value: 3}),
		classify(&Number{Value: 30}),
		classify(nil),
		classify(&Space{Width: 1})
}

func describeAny(v any) string {
	switch x := v.(type) {
	case int:
		if x > 0 {
			return "positive int"
		}
		return "other int"
	case string:
		return "string:" + x
	case bool, float64:
		if x == nil {
			return "impossible"
		}
		return "bool or float"
	}
	return "unmatched"
}

func DescribeAny() (string, string, string, string, string) {
	return describeAny(5), describeAny("go"), describeAny(true),
		describeAny(1.5), describeAny(int8(1))
}

func noBind(v any) int {
	switch v.(type) {
	case int:
		return 1
	case string:
		return 2
	}
	return 0
}

func NoBind() (int, int, int) {
	return noBind(1), noBind("x"), noBind(true)
}
`)
}

func TestOracleInterfaceCrossPackage(t *testing.T) {
	result, err := oracleRunModule(t, map[string]string{
		"helper": `package helper

type Greeter interface {
	Greet() string
}

type Formal struct {
	Title string
}

func (f *Formal) Greet() string {
	return "good day, " + f.Title
}

func MakeGreeter(title string) Greeter {
	return &Formal{Title: title}
}

func Describe(g Greeter) string {
	if g == nil {
		return "nobody"
	}
	return g.Greet()
}
`,
		"fixture": `package fixture

import "oracle.fixture/helper"

type Casual struct {
	Nick string
}

func (c *Casual) Greet() string {
	return "yo " + c.Nick
}

func CrossPackageDispatch() (string, string, string) {
	formal := helper.MakeGreeter("doctor")
	var casual helper.Greeter = &Casual{Nick: "sam"}
	return formal.Greet(), helper.Describe(casual), helper.Describe(nil)
}

// CrossPackageAssert proves rtti identity is shared across modules:
// the helper-made value asserts back to helper's concrete type here.
func CrossPackageAssert() (string, bool, bool) {
	g := helper.MakeGreeter("prof")
	f, isFormal := g.(*helper.Formal)
	_, isCasual := g.(*Casual)
	return f.Title, isFormal, isCasual
}
`,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}

func TestOracleInterfaceStructValues(t *testing.T) {
	runOracle(t, `package fixture

type Sized interface {
	Size() int
}

type Box struct {
	W int
	H int
}

func (b Box) Size() int {
	return b.W * b.H
}

func BoxedValueCopies() (int, int) {
	b := Box{W: 2, H: 3}
	var s Sized = b
	b.W = 100
	viaIface := s.Size()
	back := s.(Box)
	back.W = 50
	again := s.(Box)
	return viaIface, again.W
}
`)
}
