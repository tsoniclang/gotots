package conversion

func NarrowSigned(value int64) int8 {
	return int8(value)
}

func NarrowUnsigned(value int64) uint8 {
	return uint8(value)
}

func Sign32(value int32) uint32 {
	return uint32(value)
}

func Sign64(value int64) uint64 {
	return uint64(value)
}

func BackSign64(value uint64) int64 {
	return int64(value)
}

var EvaluationCount int32

func nextInteger() int64 {
	EvaluationCount++
	return -1
}

func Sign64EvaluatesOnce() int32 {
	EvaluationCount = 0
	converted := uint64(nextInteger())
	if converted == 0 {
		return -1
	}
	return EvaluationCount
}

func Widen(value int8) int64 {
	return int64(value)
}

func IntegerToFloat64(value int64) float64 {
	return float64(value)
}

func UnsignedToFloat32(value uint64) float32 {
	return float32(value)
}

func FloatToInt8(value float64) int8 {
	return int8(value)
}

func FloatToUint32(value float64) uint32 {
	return uint32(value)
}

func FloatToInt64(value float64) int64 {
	return int64(value)
}

func FloatToUint64(value float64) uint64 {
	return uint64(value)
}

func WidenFloat(value float32) float64 {
	return float64(value)
}

func NarrowFloat(value float64) float32 {
	return float32(value)
}

func WidenComplex(realPart, imaginaryPart float32) (float64, float64) {
	value := complex128(complex(realPart, imaginaryPart))
	return real(value), imag(value)
}

func NarrowComplex(realPart, imaginaryPart float64) (float32, float32) {
	value := complex64(complex(realPart, imaginaryPart))
	return real(value), imag(value)
}

func nextComplex() complex64 {
	EvaluationCount++
	return 1 + 2i
}

func ComplexEvaluatesOnce() int32 {
	EvaluationCount = 0
	converted := complex128(nextComplex())
	if real(converted) == 0 {
		return -1
	}
	return EvaluationCount
}

func ConstantInteger() int16 {
	return int16(255)
}

func ConstantFloat() float32 {
	return float32(16777217)
}

func ConstantComplex() complex64 {
	return complex64(1.5 - 2.25i)
}
