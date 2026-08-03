package classmembers

func Audit() []int32 {
	counter := Counter{Value: 1}
	direct := counter.Bump()
	bound := counter.Bump
	firstBound := bound()
	secondBound := bound()
	expression := Counter.Bump
	expressionResult := expression(counter)
	beforeReset := counter.Read()
	captured := counter.Capture()
	counter.Reset(9)
	return []int32{
		direct,
		counter.Value - 8,
		firstBound,
		secondBound,
		expressionResult,
		beforeReset,
		captured,
		counter.Value,
	}
}
