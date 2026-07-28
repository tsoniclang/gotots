package complexvalues

const NarrowConstant complex64 = 1.25 - 2.5i
const WideConstant = 3.5 + 4.25i
const ImaginaryConstant = 7i

func Constant64() complex64 {
	return NarrowConstant
}

func Constant128() complex128 {
	return WideConstant
}

func Imaginary128() complex128 {
	return ImaginaryConstant
}

func Folded128() complex128 {
	return (1 + 2i) * (3 - 4i)
}

func Zero64() complex64 {
	var value complex64
	return value
}

func Zero128() complex128 {
	var value complex128
	return value
}

func Construct64(realPart, imaginaryPart float32) complex64 {
	return complex(realPart, imaginaryPart)
}

func Construct128(realPart, imaginaryPart float64) complex128 {
	return complex(realPart, imaginaryPart)
}

func Real64(value complex64) float32 {
	return real(value)
}

func Imag64(value complex64) float32 {
	return imag(value)
}

func Real128(value complex128) float64 {
	return real(value)
}

func Imag128(value complex128) float64 {
	return imag(value)
}

func ConstantReal() float64 {
	return real(1 + 2i)
}

func ConstantImag() float64 {
	return imag(1 + 2i)
}

var EvaluationOrder string

func firstComponent() float64 {
	EvaluationOrder = EvaluationOrder + "real"
	return 1
}

func secondComponent() float64 {
	EvaluationOrder = EvaluationOrder + "-imag"
	return 2
}

func ConstructInOrder() complex128 {
	EvaluationOrder = ""
	return complex(firstComponent(), secondComponent())
}

func ObservedOrder() string {
	return EvaluationOrder
}

func Add64(left, right complex64) complex64 {
	return left + right
}

func Subtract64(left, right complex64) complex64 {
	return left - right
}

func Multiply64(left, right complex64) complex64 {
	return left * right
}

func Divide64(left, right complex64) complex64 {
	return left / right
}

func Negate64(value complex64) complex64 {
	return -value
}

func Identity64(value complex64) complex64 {
	return +value
}

func Equal64(left, right complex64) bool {
	return left == right
}

func NotEqual64(left, right complex64) bool {
	return left != right
}

func Add128(left, right complex128) complex128 {
	return left + right
}

func Subtract128(left, right complex128) complex128 {
	return left - right
}

func Multiply128(left, right complex128) complex128 {
	return left * right
}

func Divide128(left, right complex128) complex128 {
	return left / right
}

func Negate128(value complex128) complex128 {
	return -value
}

func Identity128(value complex128) complex128 {
	return +value
}

func Equal128(left, right complex128) bool {
	return left == right
}

func NotEqual128(left, right complex128) bool {
	return left != right
}
