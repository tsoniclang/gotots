package mathx

const (
	Offset        int32 = 2
	unusedUntyped       = 3
)

var unsupportedValue int32

func Even(value int32) int32 {
	if value == 0 {
		return Offset
	}
	return Odd(value - 1)
}

func Odd(value int32) int32 {
	if value == 0 {
		return 0
	}
	return Even(value - 1)
}

func UnusedMath(value int32) int32 {
	return value + 300
}
