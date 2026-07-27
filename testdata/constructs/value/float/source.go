package float

const Pi = 3.14159

const Neg = -4.5

const TypedWidth float64 = 1.5

func Constant() float64 {
	return Pi
}

func NegativeConstant() float64 {
	return Neg
}

func TypedConstant() float64 {
	return TypedWidth
}

func Literal() float64 {
	return 2.5
}

func WholeLiteral() float64 {
	return 8
}

func Zero() float64 {
	var value float64
	return value
}

func Large() float64 {
	return 1e300
}

func Subnormal() float64 {
	return 5e-324
}

func Rounded() float32 {
	return 0.1
}

func LocalConstant() float32 {
	const scale float32 = 2.5
	return scale
}
