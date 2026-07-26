package mathx

const (
	Offset        int = 2
	unusedUntyped     = 3
)

var unsupportedValue int

func Even(value int) int {
	if value == 0 {
		return Offset
	}
	return Odd(value - 1)
}

func Odd(value int) int {
	if value == 0 {
		return 0
	}
	return Even(value - 1)
}

func UnusedMath(value int) int {
	return value + 300
}
