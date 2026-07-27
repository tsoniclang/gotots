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
