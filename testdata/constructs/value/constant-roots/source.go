package constantroots

const (
	Zero = iota
	One
	Two
)

const Width = 64

const Label = "edge"

const Enabled = true

const unexported = 7

func Widths() (int8, int32, uint16) {
	return Width, Width, Width
}

func Flags() (bool, string) {
	return Enabled, Label
}

func Counters() (int, int, int) {
	return Zero, One, Two
}

func Hidden() int {
	return unexported
}
