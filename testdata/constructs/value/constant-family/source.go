package constantfamily

const (
	Alpha = iota
	Beta
	Gamma
)

const (
	First, Second = 10, 20
	Third, Fourth
)

const Scale = 100

const Huge = 1 << 63

const Text = "hi"

const Flag = true

const Letter = 'A'

const TypedWidth int32 = 40

const TypedEnabled bool = true

func Enum() (int, int, int) {
	return Alpha, Beta, Gamma
}

func Inherited() (int, int, int, int) {
	return First, Second, Third, Fourth
}

func MultipleTargets() (int8, int32, int64, uint16) {
	return Scale, Scale, Scale, Scale
}

func Argument() int32 {
	return takeInt32(Scale)
}

func takeInt32(value int32) int32 {
	return value
}

func Assignment() uint16 {
	var value uint16 = Scale
	return value
}

func Case(value int32) int32 {
	switch value {
	case Scale:
		return 1
	default:
		return 2
	}
}

func Conversion() int64 {
	return int64(Scale)
}

func Untyped() (int, string, bool) {
	return Scale, Text, Flag
}

func Typed() (int32, bool) {
	return TypedWidth, TypedEnabled
}

func RuneValue() rune {
	return Letter
}

func Local() (int, int) {
	const (
		low = iota
		high
	)
	return low, high
}

func HugeAsUint() uint64 {
	return Huge
}
