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
	afterImplicitPointerReceiver := counter.Value
	pointer := &counter
	pointer.Reset(10)
	return []int32{
		direct,
		afterImplicitPointerReceiver - 8,
		firstBound,
		secondBound,
		expressionResult,
		beforeReset,
		captured,
		counter.Value,
	}
}
