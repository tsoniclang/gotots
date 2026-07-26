package localvariables

func Compute(input int32) int32 {
	var base int32 = input
	{
		var (
			base        int32 = base + 1
			left, right int32 = base, base + 1
			π           int32 = left + right
		)
		left, right = right, left
		return π
	}
}

func LateOuter(input int32) int32 {
	{
		var value int32 = input + 1
		input = value
	}
	var value int32 = input + 2
	var class int32 = value + 3
	var arguments int32 = class + 4
	return arguments
}
