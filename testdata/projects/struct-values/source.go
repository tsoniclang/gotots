package structvalues

type Point struct {
	// X is intentionally documented; comments do not change representation.
	X       int32
	Visible bool
}

type Box struct {
	Point  Point
	Active bool
}

type Mirror struct {
	Point  Point
	Active bool
}

type Reserved struct {
	constructor int32
}

type Grouped struct {
	Left, Right int32
}

type Empty struct{}

func NewBox(value int32) Box {
	return Box{
		Active: value > 0,
		Point: Point{
			Visible: true,
			X:       value,
		},
	}
}

func Snapshot(value *Box) Point {
	return value.Point
}

func ReturnSnapshotResult() int32 {
	value := NewBox(1)
	snapshot := Snapshot(&value)
	value.Point.X = 2
	return snapshot.X*10 + value.Point.X
}

func ZeroIsFresh() bool {
	var left Box
	var right Box
	left.Point.X = 7
	return right.Point.X == 0
}

func CopyIsolated(value Box) int32 {
	copy := value
	copy.Point.X = copy.Point.X + 1
	return value.Point.X*10 + copy.Point.X
}

func AssignIsolated(value Box) int32 {
	var target Box
	target = value
	target.Point.X = target.Point.X + 2
	return value.Point.X*10 + target.Point.X
}

func MutateParameter(value Box) Box {
	value.Point.X = value.Point.X + 3
	return value
}

func ParameterIsolated(value Box) int32 {
	changed := MutateParameter(value)
	return value.Point.X*10 + changed.Point.X
}

func Equal(left, right Box) bool {
	return left == right
}

func (box Box) WithX(value int32) Box {
	box.Point.X = value
	return box
}

func Invoke(value Box, next int32) Box {
	return value.WithX(next)
}

func CopyResult() int32 {
	return CopyIsolated(NewBox(4))
}

func AssignResult() int32 {
	return AssignIsolated(NewBox(4))
}

func ParameterResult() int32 {
	return ParameterIsolated(NewBox(4))
}

func EqualSameResult() bool {
	return Equal(NewBox(4), NewBox(4))
}

func EqualDifferentResult() bool {
	return Equal(NewBox(4), NewBox(5))
}

func MethodResult() int32 {
	first := NewBox(4)
	changed := Invoke(first, 9)
	return changed.Point.X*10 + first.Point.X
}

func ReservedValue() int32 {
	value := Reserved{constructor: 7}
	return value.constructor
}

func PrimitiveZero() bool {
	var count int32
	var ready bool
	return count == 0 && ready == false
}

func Duplicate(value Box) (Box, Box) {
	return value, value
}

func MultipleResultIsolated() int32 {
	left, right := Duplicate(NewBox(4))
	left.Point.X = 8
	return left.Point.X*10 + right.Point.X
}

func ReadX(value Box) int32 {
	return value.Point.X
}

func CompositeArgument() int32 {
	return ReadX(Box{
		Active: true,
		Point: Point{
			Visible: true,
			X:       6,
		},
	})
}

func ReadXAfter(first int32, value Box) int32 {
	return first*10 + value.Point.X
}

func DirectValue() int32 {
	return 2
}

func CompositeSecondArgument() int32 {
	return ReadXAfter(DirectValue(), Box{
		Active: true,
		Point: Point{
			Visible: true,
			X:       6,
		},
	})
}

func CompositeField() int32 {
	return Box{
		Active: true,
		Point: Point{
			Visible: true,
			X:       7,
		},
	}.Point.X
}

func DirectVisible() bool {
	return true
}

func DirectX() int32 {
	return 6
}

func CompositeCalls() int32 {
	return Point{
		Visible: DirectVisible(),
		X:       DirectX(),
	}.X
}

func PositionalComposite() int32 {
	value := Point{8, true}
	return value.X
}

func OmittedComposite() bool {
	value := Point{X: 5}
	return value.X == 5 && !value.Visible
}

func NotEqual() bool {
	return NewBox(4) != NewBox(5)
}

func ExplicitVarCopy(value Box) int32 {
	var copied Box = value
	copied.Point.X = 6
	return value.Point.X*10 + copied.Point.X
}

func ExplicitVarCopyResult() int32 {
	return ExplicitVarCopy(NewBox(4))
}

func ParallelAssignment() int32 {
	left := NewBox(4)
	right := NewBox(9)
	left, right = right, left
	left.Point.X = 8
	return left.Point.X*10 + right.Point.X
}

func GroupedResult() int32 {
	value := Grouped{1, 2}
	return value.Left*10 + value.Right
}

func EmptyEqual() bool {
	var left Empty
	return left == Empty{}
}
