package translate_test

import "testing"

func TestOracleStructValueCopies(t *testing.T) {
	runOracle(t, `package fixture

type Point struct {
	X int
	Y int
}

type Box struct {
	Min Point
	Max Point
	Tag string
}

func AssignCopies() (int, int) {
	a := Point{X: 1, Y: 2}
	b := a
	b.X = 99
	return a.X, b.X
}

func ReassignInPlace() (int, int) {
	v := Point{X: 1, Y: 2}
	p := &v
	v = Point{X: 7, Y: 8}
	return p.X, v.Y
}

func paramMutate(p Point) int {
	p.X = 100
	return p.X
}

func ParamCopies() (int, int) {
	v := Point{X: 5, Y: 6}
	inner := paramMutate(v)
	return inner, v.X
}

func makePoint() Point {
	return Point{X: 3, Y: 4}
}

func ResultIsFresh() (int, int) {
	a := makePoint()
	b := makePoint()
	a.X = 50
	return a.X, b.X
}

func ZeroValue() (int, int, string) {
	var b Box
	return b.Min.X, b.Max.Y, b.Tag
}

func NestedCopyIsDeep() (int, int) {
	outer := Box{Min: Point{X: 1, Y: 1}, Max: Point{X: 2, Y: 2}, Tag: "a"}
	copied := outer
	copied.Min.X = 42
	return outer.Min.X, copied.Min.X
}

func NestedReassignAliases() (int, int) {
	outer := Box{Min: Point{X: 1, Y: 1}, Max: Point{X: 2, Y: 2}, Tag: "a"}
	inner := &outer.Min
	outer = Box{Min: Point{X: 9, Y: 9}, Max: Point{X: 8, Y: 8}, Tag: "b"}
	return inner.X, outer.Min.Y
}

func FieldStoreCopies() (int, int) {
	v := Point{X: 1, Y: 2}
	b := Box{Min: Point{}, Max: Point{}, Tag: ""}
	b.Min = v
	v.X = 77
	return b.Min.X, v.X
}
`)
}

func TestOracleStructValueReceivers(t *testing.T) {
	runOracle(t, `package fixture

type Counter struct {
	Total int
}

func (c Counter) Peek(extra int) int {
	c.Total = c.Total + extra
	return c.Total
}

func (c *Counter) Bump(extra int) int {
	c.Total = c.Total + extra
	return c.Total
}

func ValueReceiverCopies() (int, int) {
	c := Counter{Total: 10}
	seen := c.Peek(5)
	return seen, c.Total
}

func ValueReceiverOnPointer() (int, int) {
	c := &Counter{Total: 20}
	seen := c.Peek(5)
	return seen, c.Total
}

func PointerReceiverOnValue() (int, int) {
	c := Counter{Total: 30}
	seen := c.Bump(5)
	return seen, c.Total
}

func PointerReceiverOnNil() int {
	var c *Counter
	return c.Peek(1)
}

func DerefCopies() (int, int) {
	c := &Counter{Total: 1}
	v := *c
	v.Total = 99
	return c.Total, v.Total
}

func DerefNilPanics() int {
	var c *Counter
	v := *c
	return v.Total
}

func PointeeStore() (int, int) {
	a := Counter{Total: 1}
	p := &a
	*p = Counter{Total: 42}
	return a.Total, p.Total
}

func AddrOfValue() int {
	c := Counter{Total: 5}
	p := &c
	p.Total = 6
	return c.Total
}
`)
}

func TestOracleStructValueElements(t *testing.T) {
	runOracle(t, `package fixture

type Point struct {
	X int
	Y int
}

func SliceElementsAreValues() (int, int) {
	points := []Point{{X: 1, Y: 1}, {X: 2, Y: 2}}
	v := points[0]
	v.X = 99
	points[1].Y = 42
	return points[0].X, points[1].Y
}

func SliceStoreOverwritesInPlace() (int, int) {
	points := []Point{{X: 1, Y: 1}}
	shared := points[0:1]
	points[0] = Point{X: 7, Y: 8}
	return shared[0].X, points[0].Y
}

func MakeDistinctZeros() (int, int) {
	points := make([]Point, 2)
	points[0].X = 5
	return points[0].X, points[1].X
}

func RangeBindsCopies() (int, int) {
	points := []Point{{X: 1, Y: 1}, {X: 2, Y: 2}}
	total := 0
	for _, p := range points {
		p.X = 100
		total = total + p.X
	}
	return total, points[0].X
}

func AppendCopiesValue() (int, int) {
	v := Point{X: 1, Y: 2}
	points := []Point{}
	points = append(points, v)
	v.X = 50
	return points[0].X, v.X
}

func MapValuesCopyOnRead() (int, int) {
	m := map[string]Point{"a": {X: 1, Y: 2}}
	v := m["a"]
	v.X = 99
	stored := m["a"]
	return stored.X, v.X
}

func MapStoreCopies() (int, int) {
	v := Point{X: 3, Y: 4}
	m := map[string]Point{}
	m["k"] = v
	v.X = 77
	read := m["k"]
	return read.X, v.X
}

func MapMissReturnsZero() (int, bool) {
	m := map[string]Point{}
	v, ok := m["absent"]
	return v.X, ok
}

func structParamsAndResults(a Point, b Point) (Point, int) {
	a.X = a.X + b.X
	return a, a.Y
}

func CallThrough() (int, int, int) {
	a := Point{X: 1, Y: 2}
	b := Point{X: 10, Y: 20}
	r, y := structParamsAndResults(a, b)
	r.X = r.X + 100
	return r.X, a.X, y
}
`)
}
