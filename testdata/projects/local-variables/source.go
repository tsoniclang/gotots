package localvariables

func Compute(input int) int {
	var base int = input
	{
		var (
			base        int = base + 1
			left, right int = base, base + 1
			π           int = left + right
		)
		left, right = right, left
		return π
	}
}

func LateOuter(input int) int {
	{
		var value int = input + 1
		input = value
	}
	var value int = input + 2
	var class int = value + 3
	var arguments int = class + 4
	return arguments
}
