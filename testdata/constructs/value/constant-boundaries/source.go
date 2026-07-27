package constantboundaries

const FloatUntyped = 3.14

const ComplexUntyped = 1 + 2i

func Float() float64 {
	return FloatUntyped
}

func Complex() complex128 {
	return ComplexUntyped
}
