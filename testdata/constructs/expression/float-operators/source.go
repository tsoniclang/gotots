package floatops

func Add(a, b float64) float64 {
	return a + b
}

func Sub(a, b float64) float64 {
	return a - b
}

func Mul(a, b float64) float64 {
	return a * b
}

func Div(a, b float64) float64 {
	return a / b
}

func Negate(a float64) float64 {
	return -a
}

func Identity(a float64) float64 {
	return +a
}

func ConstantNeg() float64 {
	return -4.5
}

func Less(a, b float64) bool {
	return a < b
}

func LessEqual(a, b float64) bool {
	return a <= b
}

func Greater(a, b float64) bool {
	return a > b
}

func GreaterEqual(a, b float64) bool {
	return a >= b
}

func Equal(a, b float64) bool {
	return a == b
}

func NotEqual(a, b float64) bool {
	return a != b
}

func Add32(a, b float32) float32 {
	return a + b
}

func Sub32(a, b float32) float32 {
	return a - b
}

func Mul32(a, b float32) float32 {
	return a * b
}

func Div32(a, b float32) float32 {
	return a / b
}

func Negate32(a float32) float32 {
	return -a
}

func Identity32(a float32) float32 {
	return +a
}

func Less32(a, b float32) bool {
	return a < b
}

func LessEqual32(a, b float32) bool {
	return a <= b
}

func Greater32(a, b float32) bool {
	return a > b
}

func GreaterEqual32(a, b float32) bool {
	return a >= b
}

func Equal32(a, b float32) bool {
	return a == b
}

func NotEqual32(a, b float32) bool {
	return a != b
}

func Add32Case() float32 {
	return Add32(0.1, 0.2)
}

func Sub32Case() float32 {
	return Sub32(0.3, 0.2)
}

func Mul32Case() float32 {
	return Mul32(1.1, 1.1)
}

func Div32Case() float32 {
	return Div32(1, 3)
}

func Negate32Case() float32 {
	var value float32 = 0.5
	return Negate32(value)
}

func Identity32Case() float32 {
	var value float32 = -0.5
	return Identity32(value)
}

func Nested32Case() float32 {
	var left float32 = 0.1
	var middle float32 = 0.2
	var right float32 = 0.3
	return (left + middle) + right
}

func Overflow32Case() float32 {
	var value float32 = 3.4e38
	return value * 2
}

func Underflow32Case() float32 {
	var value float32 = 1e-45
	return value / 2
}

func Subnormal32Case() float32 {
	return 1e-45
}

func NegativeZero32Case() float32 {
	var zero float32
	return -zero
}

func Infinity32Case() float32 {
	var one float32 = 1
	var zero float32
	return one / zero
}

func NaN32Case() float32 {
	var zero float32
	return zero / zero
}

func Comparisons32Case() (bool, bool, bool, bool, bool, bool) {
	var one float32 = 1
	var two float32 = 2
	return Less32(one, two),
		LessEqual32(two, two),
		Greater32(two, one),
		GreaterEqual32(two, two),
		Equal32(one, one),
		NotEqual32(one, two)
}

func ComparisonEdges32Case() (bool, bool, bool, bool, bool) {
	var one float32 = 1
	var zero float32
	negativeZero := Negate32(zero)
	nan := Div32(zero, zero)
	return Equal32(zero, negativeZero),
		Equal32(nan, nan),
		NotEqual32(nan, nan),
		Less32(nan, one),
		Greater32(nan, one)
}
